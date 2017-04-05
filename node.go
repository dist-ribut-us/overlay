package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
	"sync"
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

type nodes struct {
	sync.RWMutex
	nByID   map[string]*node
	nByAddr map[string]*node
	beacons []*node
}

func newNodes() *nodes {
	return &nodes{
		nByID:   make(map[string]*node),
		nByAddr: make(map[string]*node),
	}
}

func (ns *nodes) nodeByAddr(addr *rnet.Addr) (*node, bool) {
	ns.RLock()
	n, ok := ns.nByAddr[addr.String()]
	ns.RUnlock()
	return n, ok
}

func (ns *nodes) nodeByID(id *crypto.ID) (*node, bool) {
	ns.RLock()
	n, ok := ns.nByID[id.String()]
	ns.RUnlock()
	return n, ok
}

func (ns *nodes) addNode(n *node) {
	id := n.Pub.ID()
	if _, ok := ns.nodeByID(id); ok {
		return
	}
	idStr := id.String()
	ns.Lock()
	ns.nByID[idStr] = n
	if n.FromAddr != nil {
		ns.nByAddr[n.FromAddr.String()] = n
	}
	ns.Unlock()
}

func (ns *nodes) addBeacon(pub *crypto.SignPub, addr *rnet.Addr) {
	n := &node{
		Pub:      pub,
		FromAddr: addr,
		ToAddr:   addr,
	}
	ns.addNode(n)
	ns.Lock()
	ns.beacons = append(ns.beacons, n)
	ns.Unlock()
}
