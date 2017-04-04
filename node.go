package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
	"time"
)

// Node represents a peer on the network running Overlay.
type Node struct {
	Pub        *crypto.SignPub
	id         *crypto.ID
	Shared     *crypto.Symmetric
	ToAddr     *rnet.Addr
	FromAddr   *rnet.Addr
	TTL        time.Duration
	liveTil    time.Time
	hsCallback func()
}

// ID returns the ID for the public key
func (n *Node) ID() *crypto.ID {
	if n.id == nil {
		n.id = n.Pub.ID()
	}
	return n.id
}

// Live returns true if the connection is still alive
func (n *Node) Live() bool {
	return n.liveTil.After(time.Now())
}
