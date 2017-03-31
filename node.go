package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
)

// Node represents a peer on the network running Overlay.
type Node struct {
	Pub      *crypto.SignPub
	id       *crypto.ID
	Shared   *crypto.Symmetric
	ToAddr   *rnet.Addr
	FromAddr *rnet.Addr
	beacon   bool
}

// ID returns the ID for the public key
func (n *Node) ID() *crypto.ID {
	if n.id == nil {
		n.id = n.Pub.ID()
	}
	return n.id
}
