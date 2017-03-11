package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
)

// Node represents a peer on the network running Overlay.
type Node struct {
	Pub      *crypto.Pub
	shared   *crypto.Shared
	ToAddr   *rnet.Addr
	FromAddr *rnet.Addr
}

// Shared gets the shared key for a node
func (n *Node) Shared(priv *crypto.Priv) *crypto.Shared {
	if n.shared == nil {
		n.shared = n.Pub.Precompute(priv)
	}
	return n.shared
}
