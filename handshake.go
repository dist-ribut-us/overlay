package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/rnet"
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

func (s *Server) handleHandshakeRequest(hs []byte, addr *rnet.Addr) {
	signPub, theirXchgPub, ok := validateHandshake(hs, nil)
	if !ok {
		log.Info(log.Lbl("handshake_validation_failed"), addr)
		return
	}
	log.Info(log.Lbl("handshake_success"), addr)

	myXchgPub, myXchgPriv := crypto.GenerateXchgKeypair()

	// TODO: hand simultaneous request

	s.AddNode(&Node{
		id:       signPub.ID(),
		Pub:      signPub,
		Shared:   theirXchgPub.Shared(myXchgPriv),
		FromAddr: addr,
		ToAddr:   addr, // This may not be right, but it's a good guess
	})

	resp := buildHandshake(handshakeResponse, myXchgPub, s.key)
	log.Info(log.Lbl("sending_handshake_resp"), addr)
	log.Error(s.net.Send(resp, addr))
}

func (s *Server) handleHandshakeResponse(hs []byte, addr *rnet.Addr) {
	signPub, xchgPub, ok := validateHandshake(hs, nil)
	if !ok {
		log.Info(log.Lbl("handshake_validation_failed"), addr)
		return
	}
	log.Info(log.Lbl("handshake_success"), addr, signPub, xchgPub)
	id := signPub.ID()
	idStr := id.String()
	xchgPriv, ok := s.xchgCache[idStr]
	if !ok {
		log.Info(log.Lbl("handshake_response_from_unrequested"), addr)
	}
	delete(s.xchgCache, idStr)
	node, ok := s.NodeByID(id)
	if !ok {
		log.Info(log.Lbl("handshake_response_from_unknown"), addr)
	}
	node.Shared = xchgPub.Shared(xchgPriv)
}

// Handshake sends a handshake packet to the specified node. The handshake
// packet will send the public key and sign it with a shared key. The receiver
// will also see what address the message came from.
func (s *Server) Handshake(node *Node) error {
	xchgPub, xchgPriv := crypto.GenerateXchgKeypair()
	hs := buildHandshake(handshakeRequest, xchgPub, s.key)
	s.xchgCache[node.ID().String()] = xchgPriv
	log.Info(log.Lbl("sending_handshake"), node.ToAddr)
	return s.net.Send(hs, node.ToAddr)
}
