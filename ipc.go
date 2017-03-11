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

// HandleMessage responds to a message received over ipc
func (s *Server) HandleMessage(msg *ipc.Message) {
	log.Info("msg_on_overlay_ipc")
	w, err := msg.Unwrap()
	if log.Error(err) {
		return
	}
	switch w.Type {
	case ipc.Type_RESPONSE:
		r := w.Response
		if r == nil {
			log.Info(log.Lbl("nil_response_from"), w.Port())
			return
		}
		log.Info("port", serial.UnmarshalUint16(r.Body))
	default:
		log.Info(log.Lbl("overlay_unknown_type"), w.Type)
	}
}
