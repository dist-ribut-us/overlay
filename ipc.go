package overlay

import (
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"os"
)

const serviceID uint32 = 4200394536

// handleIPCMessage responds to a message received over ipc
func (s *Server) handleIPCMessage(b *ipc.Base) {
	if b.IsToNet() {
		b.UnsetFlag(message.ToNet)
		n, ok := s.nByAddr[b.GetAddr().String()]
		if !ok {
			log.Info(log.Lbl("unknown_node"), b.GetAddr().String())
			return
		}
		s.netSend(b.Header, n, true, b.Port())
	} else if b.IsQuery() {
		s.handleQuery(b)
	} else {
		s.handleOther(b)
	}
}

func (s *Server) handleQuery(q *ipc.Base) {
	switch t := q.GetType(); t {
	case message.Ping:
		q.Respond([]byte{q.Body[0] + 1})
	case message.GetPubKey:
		q.Respond(s.key.Pub().Slice())
	case message.GetPort:
		q.Respond(uint32(s.net.Port()))
	case message.SessionData:
		s.handleSessionDataQuery(q)
	default:
		log.Info(log.Lbl("unknown_query_type"), t)
	}
}

func (s *Server) handleOther(b *ipc.Base) {
	switch t := b.GetType(); t {
	case message.RegisterService:
		s.handleRegisterService(b)
	case message.AddBeacon:
		s.handleAddBeacon(b)
	case message.Die:
		os.Exit(0)
	default:
		log.Info(log.Lbl("unknown_type"), t)
	}
}

func (s *Server) handleRegisterService(b *ipc.Base) {
	id := b.BodyToUint32()
	log.Info(log.Lbl("registered_service"), id, b.Port())
	s.servicesMux.Lock()
	s.services[id] = b.Port()
	s.servicesMux.Unlock()
}
