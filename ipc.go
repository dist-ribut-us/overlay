package overlay

import (
	"github.com/dist-ribut-us/ipcrouter"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"os"
)

// handleIPCMessage responds to a message received over ipc
func (s *Server) handleIPCMessage(b *ipcrouter.Base) {
	if b.IsQuery() {
		s.handleQuery(b)
	} else {
		s.handleOther(b)
	}
}

func (s *Server) handleToNet(b *ipcrouter.Base) {
	b.UnsetFlag(message.ToNet)
	n, ok := s.nByAddr[b.GetAddr().String()]
	if !ok {
		log.Info(log.Lbl("send_to_unknown_node"), b.GetAddr().String(), b.GetType32(), s.router.Port())
		return
	}
	s.netSend(b.Header, n, true, b.Port())
}

func (s *Server) handleQuery(q *ipcrouter.Base) {
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

func (s *Server) handleOther(b *ipcrouter.Base) {
	switch t := b.GetType(); t {
	case message.RegisterService:
		s.handleRegisterService(b)
	case message.AddBeacon:
		s.handleAddBeacon(b)
	case message.Die:
		os.Exit(0)
	case message.StaticKey:
		s.LoadKey()
	case message.RandomKey:
		s.RandomKey()
	default:
		log.Info(log.Lbl("unknown_type"), t)
	}
}

func (s *Server) handleRegisterService(b *ipcrouter.Base) {
	id := b.BodyToUint32()
	log.Info(log.Lbl("registered_service"), id, b.Port())
	s.services.set(id, b.Port())
}
