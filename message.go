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
)

// Compression tags
const (
	NoCompression = byte(iota)
	GZipped
)

func (s *Server) message(cPkt []byte, addr *rnet.Addr) {
	node, ok := s.NodeByAddr(addr)
	if !ok {
		log.Info(log.Lbl("unknown_node"), addr)
		return
	}

	pPkt, err := node.Shared(s.priv).Open(cPkt[1:])
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
	if originPort, ok := s.callbacks[h.Id]; ok {
		port = originPort
	} else if servicePort, ok := s.services[h.Service]; ok {
		port = servicePort
	} else {
		log.Info(log.Lbl("no_service_or_callback_for_msg"))
		return
	}
	id := h.Id
	h.Id = 0
	s.ipc.Send(id, h.Marshal(), port)
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
	node, ok := s.NodeByAddr(msg.Addr)
	if !ok {
		return nil, ErrUnknonNode
	}
	h.NodeID = node.GetID()[:]
	h.Id = msg.ID
	h.SetAddr(msg.Addr)

	return h, nil
}

var encSharedTag = []byte{encShared}

var noCompressionTag = []byte{NoCompression}
var gzTag = []byte{GZipped}

// NetSend sends a message over the network
func (s *Server) NetSend(msg *message.Header, node *Node, compression bool, origin rnet.Port) {
	var bts []byte
	var bb *bytes.Buffer

	id := msg.Id
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

	packets = node.Shared(s.priv).SealPackets(encSharedTag, packets, nil, 0)

	pbPool.Put(pb)
	if bb != nil {
		bufpool.Put(bb)
	}

	if msg.IsQuery() {
		s.callbacks[id] = origin
	}
	errs := s.net.SendAll(packets, node.ToAddr)
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
