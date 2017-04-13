package overlay

import (
	"bytes"
	"compress/gzip"
	"github.com/dist-ribut-us/bufpool"
	"github.com/dist-ribut-us/errors"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/dist-ribut-us/packeter"
	"github.com/dist-ribut-us/rnet"
	"github.com/golang/protobuf/proto"
	"sync"
	"time"
)

// Compression tags
const (
	NoCompression = byte(iota)
	GZipped
)

func (s *Server) message(cPkt []byte, addr *rnet.Addr) {
	n, ok := s.nodeByAddr(addr)
	if !ok {
		log.Info(log.Lbl("unknown_node"), addr)
		return
	}

	pPkt, err := n.Shared.Open(cPkt[1:])
	if log.Error(errors.Wrap("decrypting overly message", err)) {
		return
	}
	s.packeter.Receive(pPkt, addr)
}

func (s *Server) handleNetMessage(msg *packeter.Package) {
	if log.Error(msg.Err) {
		return
	} else if len(msg.Body) == 0 {
		log.Info(log.Lbl("message_has_no_body"))
	}

	h, err := s.unmarshalNetMessage(msg)
	if log.Error(err) {
		return
	}

	var port rnet.Port
	originPort, ok := s.callbacks.get(h.Id)
	if ok {
		port = originPort
	} else {
		servicePort, ok := s.services.get(h.Service)
		if ok {
			port = servicePort
		} else {
			log.Info(log.Lbl("no_service_or_callback_for_msg"), h.Id, h.Service)
			return
		}
	}
	s.router.Send(port, h)
}

// ErrUnknonNode will occure if a message is received from an unknown address.
// This shouldn't happen because the packets need to know the node to be
// decrypted.
const ErrUnknonNode = errors.String("Unknown node by address")

func (s *Server) unmarshalNetMessage(msg *packeter.Package) (*message.Header, error) {
	if msg.Body[0] == GZipped {
		b, err := decompress(msg.Body[1:])
		if err != nil {
			return nil, err
		}
		msg.Body = append(msg.Body[0:0], b.Bytes()...)
		bufpool.Put(b)
	} else {
		msg.Body = msg.Body[1:]
	}

	h := &message.Header{}
	err := proto.Unmarshal(msg.Body, h)
	if err != nil {
		return nil, err
	}
	h.SetFlag(message.FromNet)
	n, ok := s.nodeByAddr(msg.Addr)
	if !ok {
		return nil, ErrUnknonNode
	}
	h.NodeID = n.id()[:]
	h.Id = msg.ID
	h.SetAddr(msg.Addr)
	if n.TTL > 0 {
		n.liveTil = time.Now().Add(n.TTL)
	}

	return h, nil
}

var encSymmetricTag = []byte{encSymmetric}

var noCompressionTag = []byte{NoCompression}
var gzTag = []byte{GZipped}

// ErrMsgIDZero is returned if there is an attempt to send a message with ID of
// 0 - this is probably a sign that something isn't correctly setting the ID
const ErrMsgIDZero = errors.String("Message ID cannot be 0")

func (s *Server) netSend(msg *message.Header, n *node, compression bool, origin rnet.Port) {
	s.addNode(n)
	if n.Shared == nil || !n.live() {
		log.Info(log.Lbl("delay_net_send_for_handshake"), n.Shared == nil, !n.live(), n.liveTil)
		s.sendHandshakeRequest(n, func() {
			log.Info(log.Lbl("handshake_complete:resuming"))
			s.netSend(msg, n, compression, origin)
		})
		return
	}

	var bts []byte
	var bb *bytes.Buffer

	id := msg.Id
	if id == 0 {
		log.Error(ErrMsgIDZero)
		return
	}
	msg.Id = 0

	pb := getPBuffer(noCompressionTag)
	if log.Error(pb.Marshal(msg)) {
		return
	}
	bts = pb.Bytes()

	if compression {
		bb = compress(gzTag, bts[1:])
		cpBts := bb.Bytes()
		if len(cpBts) < len(bts) {
			bts = cpBts
		} else {
			bufpool.Put(bb)
			bb = nil
		}
	}

	packets, err := s.packeter.Make(nil, bts, s.loss, s.reliability, id)
	if log.Error(err) {
		packets = [][]byte{bts}
	}

	packets = n.Shared.SealPackets(encSymmetricTag, packets, nil, 0)

	pbPool.Put(pb)
	if bb != nil {
		bufpool.Put(bb)
	}

	if msg.IsQuery() {
		s.callbacks.set(id, origin)
	}
	errs := s.net.SendAll(packets, n.ToAddr)
	for _, err := range errs {
		log.Error(err)
	}
}

var pbPool = sync.Pool{
	New: func() interface{} {
		return proto.NewBuffer(nil)
	},
}

func getPBuffer(tag []byte) *proto.Buffer {
	b := pbPool.Get().(*proto.Buffer)
	b.Reset()
	b.SetBuf(append(b.Bytes(), tag...))
	return b
}

func compress(tag, msg []byte) *bytes.Buffer {
	b := bufpool.Get()
	_, err := b.Write(tag)
	if log.Error(err) {
		return nil
	}
	w := gzip.NewWriter(b)
	_, err = w.Write(msg)
	if log.Error(err) {
		log.Error(w.Close())
		return nil
	}
	if log.Error(w.Close()) {
		return nil
	}
	return b
}

func decompress(zmsg []byte) (*bytes.Buffer, error) {
	br := bytes.NewBuffer(zmsg)
	r, err := gzip.NewReader(br)
	if err != nil {
		return nil, err
	}
	err = r.Close()
	if err != nil {
		return nil, err
	}
	b := bufpool.Get()
	_, err = b.ReadFrom(r)
	bufpool.Put(br)
	if err != nil {
		return nil, err
	}
	return b, nil
}
