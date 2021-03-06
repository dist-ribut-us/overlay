package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/errors"
	"github.com/dist-ribut-us/ipcrouter"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/dist-ribut-us/overlay/overlaymessages"
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
	keypair, _ := s.xchgCache.get(id.String())
	if keypair == nil {
		keypair = crypto.GenerateXchgPair()
	}

	if n, ok := s.nodeByAddr(addr); ok {
		if n.Pub != nil && *n.Pub != *signPub {
			log.Error(ErrBadSignPub)
			return
		}
		n.Shared = keypair.Shared(xchgPub)
		n.liveTil = time.Now().Add(time.Duration(s.NodeTTL) * time.Second)
	} else {
		n := &node{
			cachedID: id,
			Pub:      signPub,
			Shared:   keypair.Shared(xchgPub),
			FromAddr: addr,
			ToAddr:   addr, // This may not be right, but it's a good guess
			liveTil:  time.Now().Add(time.Duration(s.NodeTTL) * time.Second),
		}
		s.addNode(n)
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
	keypair, ok := s.xchgCache.get(idStr)
	if !ok {
		log.Info(log.Lbl("handshake_response_from_unrequested"), addr)
	}
	n, ok := s.nodeByID(id)
	if !ok {
		log.Info(log.Lbl("handshake_response_from_unknown"), addr)
	}
	n.Shared = keypair.Shared(xchgPub)

	n.liveTil = time.Now().Add(time.Duration(s.NodeTTL) * time.Second)

	s.router.
		Query(message.SessionData, s.NodeTTL).
		SetService(overlaymessages.ServiceID).
		SendToNet(n.ToAddr, func(r ipcrouter.NetResponse) {
			ttl := r.BodyToUint32()
			if ttl > s.NodeTTL {
				ttl = s.NodeTTL
			}
			n.TTL = time.Duration(ttl) * time.Second
			n.liveTil = time.Now().Add(n.TTL)
		})

	if n.hsCallback != nil {
		go n.hsCallback()
		n.hsCallback = nil
	}
}

func (s *Server) sendHandshakeRequest(n *node, callback func()) error {
	id := n.id()
	idStr := id.String()

	keypair, ok := s.xchgCache.get(idStr)
	if !ok {
		keypair = crypto.GenerateXchgPair()
		s.xchgCache.set(idStr, keypair)
		go s.removeXchgPair(idStr)
	}

	hs := buildHandshake(handshakeRequest, keypair.Pub(), s.key)
	n.hsCallback = callback

	log.Info(log.Lbl("sending_handshake_request"), n.ToAddr)
	err := s.net.Send(hs, n.ToAddr)
	return err
}

var removeKeyDelay = time.Second * 2

func (s *Server) removeXchgPair(id string) {
	time.Sleep(removeKeyDelay)
	s.xchgCache.delete(id)
}

func (s *Server) handleSessionDataQuery(q ipcrouter.NetQuery) {
	nodeID, err := crypto.IDFromSlice(q.GetNodeID())
	if log.Error(err) {
		return
	}
	n, ok := s.nodeByID(nodeID)
	if !ok {
		return
	}
	ttl := q.BodyToUint32()
	if ttl > s.NodeTTL {
		ttl = s.NodeTTL
	}
	n.TTL = time.Duration(ttl) * time.Second
	n.liveTil = time.Now().Add(n.TTL)

	q.Respond(s.NodeTTL)
}
