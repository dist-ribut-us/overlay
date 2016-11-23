package overlay

import (
	"bytes"
	"fmt"
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
)

func buildHandshake(pub *crypto.Pub, shared *crypto.Shared) []byte {
	box := shared.Seal(pub.Slice())
	hs := make([]byte, 1+crypto.KeyLength+len(box))
	hs[0] = Handshake
	copy(hs[1:], pub.Slice())
	copy(hs[1+crypto.KeyLength:], box)
	return hs
}

func validateHandshake(hs []byte, priv *crypto.Priv) (*crypto.Pub, *crypto.Shared, bool) {
	if len(hs) < 1+crypto.KeyLength {
		return nil, nil, false
	}
	pub, err := crypto.PubFromSlice(hs[:crypto.KeyLength])
	if err != nil {
		return nil, nil, false
	}
	shared := pub.Precompute(priv)
	box, err := shared.Open(hs[crypto.KeyLength:])
	if err != nil || !bytes.Equal(hs[:crypto.KeyLength], box) {
		return nil, nil, false
	}
	return pub, shared, true
}

func (s *Server) handshake(hs []byte, addr *rnet.Addr) {
	pub, shared, ok := validateHandshake(hs, s.priv)
	if !ok {
		return
	}

	fmt.Println("Added Node")

	s.AddNode(&Node{
		Pub:      pub,
		shared:   shared,
		FromAddr: addr,
	})
}

// Handshake sends a handshake packet to the specified node. The handshake
// packet will send the public key and sign it with a shared key. The receiver
// will also see what address the message came from.
func (s *Server) Handshake(node *Node) {
	hs := buildHandshake(s.pub, node.Shared(s.priv))
	s.Server.Send(hs, node.ToAddr)
}
