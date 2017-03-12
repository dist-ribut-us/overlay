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

// SendQuery sends a query to another process and registers a callback to handle
// the response
func (s *Server) SendQuery(q *ipc.Query, port rnet.Port, callback func(r *ipc.Wrapper)) {
	s.ipc.SendQuery(q, port, callback)
}

// HandleMessage responds to a message received over ipc
func (s *Server) HandleMessage(msg *ipc.Message) {
	log.Info("msg_on_overlay_ipc")
	w, err := msg.Unwrap()
	if log.Error(err) {
		return
	}
	switch w.Type {
	default:
		log.Info(log.Lbl("overlay_unknown_type"), w.Type)
	}
}
