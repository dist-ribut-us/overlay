package overlay

import (
	"github.com/dist-ribut-us/rnet"
)

const (
	Handshake = byte(iota)
	Message
)

var handlers = map[byte]func(*Server, []byte, *rnet.Addr){
	Handshake: (*Server).handshake,
	Message:   (*Server).message,
}

func (s *Server) Receive(pkt []byte, addr *rnet.Addr) {
	if len(pkt) < 1 {
		return
	}
	if handler, ok := handlers[pkt[0]]; ok {
		handler(s, pkt[1:], addr)
	}
}
