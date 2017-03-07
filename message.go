package overlay

import (
	"github.com/dist-ribut-us/rnet"
)

func (s *Server) message(cPkt []byte, addr *rnet.Addr) {
	node, ok := s.NodeByAddr(addr)
	if !ok {
		return
	}

	pPkt, err := node.Shared(s.priv).Open(cPkt)
	if err != nil {
		return
	}

	s.packeter.Receive(pPkt, addr)
}

func (s *Server) Send(msg []byte, node *Node) {
	pkts, err := s.packeter.Make(msg, s.loss, s.reliability)
	if err != nil {
		return
	}
	shared := node.Shared(s.priv)

	for i, pkt := range pkts {
		pkts[i] = append([]byte{Message}, shared.Seal(pkt, nil)...)
	}

	s.Server.SendAll(pkts, node.ToAddr)
}
