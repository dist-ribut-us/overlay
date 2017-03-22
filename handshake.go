package overlay

import (
	"bytes"
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/rnet"
)

func buildHandshake(pub *crypto.Pub, shared *crypto.Shared) []byte {
	box := shared.Seal(pub.Slice(), nil)
	hs := make([]byte, 1+crypto.KeyLength+len(box))
	hs[0] = handshake
	copy(hs[1:], pub.Slice())
	copy(hs[1+crypto.KeyLength:], box)
	return hs
}

func validateHandshake(hs []byte, priv *crypto.Priv) (*crypto.Pub, *crypto.Shared, bool) {
	if len(hs) < 1+crypto.KeyLength {
		return nil, nil, false
	}
	pub := crypto.PubFromSlice(hs[:crypto.KeyLength])
	shared := pub.Precompute(priv)
	box, err := shared.Open(hs[crypto.KeyLength:])
	if err != nil || !bytes.Equal(hs[:crypto.KeyLength], box) {
		return nil, nil, false
	}
	return pub, shared, true
}

func (s *Server) handshake(hs []byte, addr *rnet.Addr) {
	pub, shared, ok := validateHandshake(hs[1:], s.priv)
	if !ok {
		log.Info(log.Lbl("handshake_validation_failed"), addr)
		return
	}
	log.Info(log.Lbl("handshake_success"), addr)
	s.AddNode(&Node{
		Pub:      pub,
		shared:   shared,
		FromAddr: addr,
		ToAddr:   addr, // This may not be right, but it's a good guess
	})
}

// Handshake sends a handshake packet to the specified node. The handshake
// packet will send the public key and sign it with a shared key. The receiver
// will also see what address the message came from.
func (s *Server) Handshake(node *Node) error {
	hs := buildHandshake(s.pub, node.Shared(s.priv))
	log.Info(log.Lbl("sending_handshake"), node.ToAddr)
	return s.net.Send(hs, node.ToAddr)
}
