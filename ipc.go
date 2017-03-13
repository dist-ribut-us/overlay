package overlay

import (
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/rnet"
	"github.com/dist-ribut-us/serial"
)

// IPCSend sends a message to a process
func (s *Server) IPCSend(msg []byte, port rnet.Port) {
	s.ipc.Send(msg, port)
}

// Query wraps the ipc.Proc query method
func (s *Server) Query(t uint32, body []byte) *ipc.Base {
	return s.ipc.Query(t, body)
}

// HandleMessage responds to a message received over ipc
func (s *Server) HandleMessage(msg *ipc.Message) {
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
	switch q.Type {
	case ipc.TPing:
		q.Respond([]byte{q.Body[0] + 1})
	default:
		log.Info(log.Lbl("unknown_query_type"), q.Type)
	}
}

func (s *Server) handleOther(b *ipc.Base) {
	switch b.Type {
	case ipc.TRegister:
		id := serial.UnmarshalUint32(b.Body)
		log.Info(log.Lbl("registered_service"), id, b.Port())
		s.services[id] = b.Port()
	default:
		log.Info(log.Lbl("unknown_type"), b.Type)
	}
}
