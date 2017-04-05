package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/errors"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/dist-ribut-us/rnet"
	"time"
)

const (
	hsMsgLen  = 1 + crypto.KeyLength*2
	hsFullLen = hsMsgLen + crypto.SignatureLength
)

func buildHandshake(kind byte, xchg *crypto.XchgPub, sign *crypto.SignPriv) []byte {
	hs := make([]byte, hsMsgLen, hsFullLen)
	hs[0] = kind
	copy(hs[1:], xchg.Slice())
	copy(hs[1+crypto.KeyLength:], sign.Pub().Slice())
	return append(hs, sign.Sign(hs)...)
}

func validateHandshake(hs []byte, expectedSignPub *crypto.SignPub) (*crypto.SignPub, *crypto.XchgPub, bool) {
	if len(hs) < hsFullLen {
		return nil, nil, false
	}
	signPub := crypto.SignPubFromSlice(hs[1+crypto.KeyLength : hsMsgLen])
	if (expectedSignPub != nil && *signPub != *expectedSignPub) || !signPub.Verify(hs[:hsMsgLen], hs[hsMsgLen:]) {
		return nil, nil, false
	}

	xchgPub := crypto.XchgPubFromSlice(hs[1 : crypto.KeyLength+1])
	return signPub, xchgPub, true
}

// ErrBadSignPub is returned if a node id does not match
var ErrBadSignPub = errors.String("Public Signature Key does not match")

func (s *Server) handleHandshakeRequest(hs []byte, addr *rnet.Addr) {
	signPub, xchgPub, ok := validateHandshake(hs, nil)
	if !ok {
		log.Info(log.Lbl("handshake_validation_failed"), addr)
		return
	}
	log.Info(log.Lbl("handshake_request_success"), addr)

	id := signPub.ID()
	// in the unlikely case that we both made the request at the same time
	s.cacheMux.RLock()
	keypair := s.xchgCache[id.String()]
	s.cacheMux.RUnlock()
	if keypair == nil {
		keypair = crypto.GenerateXchgPair()
	}

	if node, ok := s.nodeByAddr(addr); ok {
		if node.Pub != nil && *node.Pub != *signPub {
			log.Error(ErrBadSignPub)
			return
		}
		node.Shared = keypair.Shared(xchgPub)
		node.liveTil = time.Now().Add(time.Duration(s.NodeTTL) * time.Second)
	} else {
		node := &Node{
			id:       id,
			Pub:      signPub,
			Shared:   keypair.Shared(xchgPub),
			FromAddr: addr,
			ToAddr:   addr, // This may not be right, but it's a good guess
			liveTil:  time.Now().Add(time.Duration(s.NodeTTL) * time.Second),
		}
		s.AddNode(node)
	}

	resp := buildHandshake(handshakeResponse, keypair.Pub(), s.key)
	log.Info(log.Lbl("sending_handshake_resp"), addr)
	log.Error(s.net.Send(resp, addr))
}

// how long a node stays live after a handshake regardless of TTL
var handshakeLiveBuffer = time.Second * 10

func (s *Server) handleHandshakeResponse(hs []byte, addr *rnet.Addr) {
	// TODO: validate addr matches node
	signPub, xchgPub, ok := validateHandshake(hs, nil)
	if !ok {
		log.Info(log.Lbl("handshake_validation_failed"), addr)
		return
	}
	log.Info(log.Lbl("handshake_response_success"), addr)
	id := signPub.ID()
	idStr := id.String()
	s.cacheMux.RLock()
	keypair, ok := s.xchgCache[idStr]
	s.cacheMux.RUnlock()
	if !ok {
		log.Info(log.Lbl("handshake_response_from_unrequested"), addr)
	}
	node, ok := s.nodeByID(id)
	if !ok {
		log.Info(log.Lbl("handshake_response_from_unknown"), addr)
	}
	node.Shared = keypair.Shared(xchgPub)

	node.liveTil = time.Now().Add(time.Duration(s.NodeTTL) * time.Second)

	s.ipc.
		Query(message.SessionData, s.NodeTTL).
		ToNet(s.ipc.Port(), node.ToAddr, serviceID).
		Send(func(r *ipc.Base) {
			ttl := r.BodyToUint32()
			if ttl > s.NodeTTL {
				ttl = s.NodeTTL
			}
			node.TTL = time.Duration(ttl) * time.Second
			node.liveTil = time.Now().Add(node.TTL)
		})

	if node.hsCallback != nil {
		go node.hsCallback()
		node.hsCallback = nil
	}
}

func (s *Server) sendHandshakeRequest(node *Node, callback func()) error {
	id := node.ID()
	idStr := id.String()

	s.cacheMux.Lock()
	keypair, ok := s.xchgCache[idStr]
	if !ok {
		keypair = crypto.GenerateXchgPair()
		s.xchgCache[idStr] = keypair
		go s.removeXchgPair(idStr)
	}
	s.cacheMux.Unlock()

	hs := buildHandshake(handshakeRequest, keypair.Pub(), s.key)
	node.hsCallback = callback

	log.Info(log.Lbl("sending_handshake_request"), node.ToAddr)
	err := s.net.Send(hs, node.ToAddr)
	return err
}

var removeKeyDelay = time.Second * 2

func (s *Server) removeXchgPair(id string) {
	time.Sleep(removeKeyDelay)
	s.cacheMux.Lock()
	delete(s.xchgCache, id)
	s.cacheMux.Unlock()
}

func (s *Server) handleSessionDataQuery(q *ipc.Base) {
	nodeID, err := crypto.IDFromSlice(q.NodeID)
	if log.Error(err) {
		return
	}
	node, ok := s.nodeByID(nodeID)
	if !ok {
		return
	}
	ttl := q.BodyToUint32()
	if ttl > s.NodeTTL {
		ttl = s.NodeTTL
	}
	node.TTL = time.Duration(ttl) * time.Second
	node.liveTil = time.Now().Add(node.TTL)

	q.Respond(s.NodeTTL)
}
