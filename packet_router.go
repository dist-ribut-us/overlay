package overlay

import (
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/rnet"
)

const (
	handshakeRequest = byte(iota)
	handshakeResponse
	encSymmetric
)

var handlers = map[byte]func(*Server, []byte, *rnet.Addr){
	handshakeRequest:  (*Server).handleHandshakeRequest,
	handshakeResponse: (*Server).handleHandshakeResponse,
	encSymmetric:      (*Server).message,
}

// Receive fulfills PacketHandler allowing the server to handle network packets
func (s *Server) Receive(pkt []byte, addr *rnet.Addr) {
	if len(pkt) < 1 {
		log.Info(log.Lbl("empty_packet"), addr)
		return
	}
	handler, ok := handlers[pkt[0]]
	if !ok {
		log.Info(log.Lbl("unknown_packet_type"), pkt[0], addr)
		return
	}
	handler(s, pkt, addr)
}
