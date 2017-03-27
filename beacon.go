package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
)

var beaconBkt = []byte("beacon")

type beacon struct {
	overlay *Server
	node    *Node
}

func (s *Server) addBeacon(b *ipc.Base) {
	addr := b.GetAddr()
	if addr == nil {
		log.Info(log.Lbl("cannot_add_beacon_addr_is_nil"))
		return
	}
	pub := crypto.XchgPubFromSlice(b.Body)
	n := &Node{
		Pub:      pub,
		FromAddr: addr,
		ToAddr:   addr,
		beacon:   true,
	}
	s.AddNode(n)
	s.Handshake(n)
	log.Info(log.Lbl("added_beacon"), addr, pub)

	bcn := &beacon{
		node:    n,
		overlay: s,
	}
	s.beacons = append(s.beacons, bcn)
}

func (b *beacon) save() {
	buf := message.FromAddr(b.node.ToAddr).Marshal()
	key := b.node.Pub.Slice()
	b.overlay.forest.SetValue(beaconBkt, key, buf)
}

func (s *Server) loadBeacons() {
	for key, val, err := s.forest.First(beaconBkt); key != nil && !log.Error(err); key, val, err = s.forest.Next(beaconBkt, key) {
		addr := message.UnmarshalAddrpb(val).GetAddr()
		n := &Node{
			Pub:      crypto.XchgPubFromSlice(key),
			ToAddr:   addr,
			FromAddr: addr,
			beacon:   true,
		}
		b := &beacon{
			overlay: s,
			node:    n,
		}
		s.beacons = append(s.beacons, b)
	}
}
