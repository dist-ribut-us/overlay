package overlay

import (
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/dist-ribut-us/rnet"
)

// IPCSend is a raw send call to the ipc
func (s *Server) IPCSend(msg []byte, port rnet.Port) {
	s.ipc.Send(msg, port)
}

// Query takes a type and a body and sends an IPC query. For valid body values
// see dist-ribut-us/message.SetBody.
func (s *Server) Query(t message.Type, body interface{}) *ipc.Base {
	return s.ipc.Query(t, body)
}

// handleIPCMessage responds to a message received over ipc
func (s *Server) handleIPCMessage(msg *ipc.Message) {
	log.Info("msg_on_overlay_ipc")
	b, err := msg.ToBase()
	if log.Error(err) {
		return
	}
	if b.IsQuery() {
		s.handleQuery(b)
	} else {
		s.handleOther(b)
	}
}

func (s *Server) handleQuery(q *ipc.Base) {
	switch t := q.GetType(); t {
	case message.Ping:
		q.Respond([]byte{q.Body[0] + 1})
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
	default:
		log.Info(log.Lbl("unknown_type"), t)
	}
}
