package overlay

import (
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/rnet"
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
		log.Info(log.Lbl("overlay_unknown_type"), b.Type)
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
