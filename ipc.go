package overlay

import (
	"github.com/dist-ribut-us/ipcrouter"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/dist-ribut-us/overlay/overlaymessages"
	"os"
)

// QueryHandler for ipc queries to Overlay service
func (s *Server) QueryHandler(q ipcrouter.Query) {
	switch t := q.GetType(); t {
	case message.Ping:
		q.Respond([]byte{q.GetBody()[0] + 1})
	case message.GetPubKey:
		q.Respond(s.key.Pub().Slice())
	case message.GetPort:
		q.Respond(uint32(s.net.Port()))
	case overlaymessages.GetID:
		q.Respond(
			(&overlaymessages.ID{
				Sign:  s.key.Pub(),
				Xchng: s.keyX.Pub(),
			}).Serialize(),
		)
	default:
		log.Info(log.Lbl("unknown_query_type"), t)
	}
}

// CommandHandler for ipc commands to Overlay service
func (s *Server) CommandHandler(c ipcrouter.Command) {
	switch t := c.GetType(); t {
	case message.RegisterService:
		s.handleRegisterService(c)
	case message.AddBeacon:
		s.handleAddBeacon(c)
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

func (s *Server) handleRegisterService(c ipcrouter.Command) {
	id := c.BodyToUint32()
	log.Info(log.Lbl("registered_service"), id, c.Port())
	s.services.set(id, c.Port())
}
