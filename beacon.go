package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/dist-ribut-us/rnet"
)

var beaconBkt = []byte("beacon")

func (s *Server) handleAddBeacon(b *ipc.Base) {
	addr := b.GetAddr()
	if addr == nil {
		log.Info(log.Lbl("cannot_add_beacon_addr_is_nil"))
		return
	}
	pub := crypto.SignPubFromSlice(b.Body)
	s.addBeacon(pub, addr)
}

func (s *Server) saveBeacon(b *node) {
	buf := message.FromAddr(b.ToAddr).Marshal()
	key := b.Pub.Slice()
	s.forest.SetValue(beaconBkt, key, buf)
}

func (s *Server) loadBeacons() {
	for key, val, err := s.forest.First(beaconBkt); key != nil && !log.Error(err); key, val, err = s.forest.Next(beaconBkt, key) {
		pub := crypto.SignPubFromSlice(key)
		addr := message.UnmarshalAddrpb(val).GetAddr()
		s.addBeacon(pub, addr)
	}
}

func (s *Server) addBeacon(pub *crypto.SignPub, addr *rnet.Addr) {
	n := &node{
		Pub:      pub,
		FromAddr: addr,
		ToAddr:   addr,
	}
	s.addNode(n)
	s.nodesMux.Lock()
	s.beacons = append(s.beacons, n)
	s.nodesMux.Unlock()
}
