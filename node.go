package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
	"time"
)

type node struct {
	Pub        *crypto.SignPub
	cachedID   *crypto.ID
	Shared     *crypto.Symmetric
	ToAddr     *rnet.Addr
	FromAddr   *rnet.Addr
	TTL        time.Duration
	liveTil    time.Time
	hsCallback func()
}

func (n *node) id() *crypto.ID {
	if n.cachedID == nil {
		n.cachedID = n.Pub.ID()
	}
	return n.cachedID
}

func (n *node) live() bool {
	return n.liveTil.After(time.Now())
}
