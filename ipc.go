package overlay

import (
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"os"
)

// Query takes a type and a body and sends an IPC query. For valid body values
// see dist-ribut-us/message.SetBody.
func (s *Server) Query(t message.Type, body interface{}) *ipc.Base {
	return s.ipc.Query(t, body)
}

// handleIPCMessage responds to a message received over ipc
func (s *Server) handleIPCMessage(b *ipc.Base) {
	log.Info("msg_on_overlay_ipc")
	if b.IsToNet() {
		node, ok := s.nByAddr[b.GetAddr().String()]
		if !ok {
			log.Info(log.Lbl("unknown_node"), b.GetAddr().String())
			return
		}
		s.NetSend(b.Header, node, true, b.Port())
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
		q.Respond(s.pub.Slice())
	case message.GetPort:
		q.Respond(uint32(s.NetPort()))
	default:
		log.Info(log.Lbl("unknown_query_type"), t)
	}
}

func (s *Server) handleOther(b *ipc.Base) {
	switch t := b.GetType(); t {
	case message.RegisterService:
		id := b.BodyToUint32()
		log.Info(log.Lbl("registered_service"), id, b.Port())
		s.services[id] = b.Port()
	case message.AddBeacon:
		s.addBeacon(b)
	case message.Die:
		os.Exit(0)
	default:
		log.Info(log.Lbl("unknown_type"), t)
	}
}
