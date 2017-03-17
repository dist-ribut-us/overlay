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

func (s *Server) handleNetMessage(msg *packeter.Message) {
	if log.Error(msg.Err) {
		return
	} else if len(msg.Body) == 0 {
		log.Info(log.Lbl("message_has_no_body"))
	}

	h, err := s.unmarshalNetMessage(msg)
	if log.Error(err) {
		return
	}
	servicePort, ok := s.services[h.Service]
	if !ok {
		log.Info(log.Lbl("no_service_registered_for"), h.Service)
		return
	}
	s.IPCSend(msg.Body, servicePort)
}

func (s *Server) unmarshalNetMessage(msg *packeter.Message) (*message.Header, error) {
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
	h.SetType(message.NetReceive)
	h.SetAddr(msg.Addr)
	h.Id = msg.ID

	return h, nil
}

var encSharedTag = []byte{encShared}

var noCompressionTag = []byte{NoCompression}
var gzTag = []byte{GZipped}

// NetSend sends a message over the network
func (s *Server) NetSend(msg *message.Header, node *Node, compression bool) {
	msg.Service = msg.Type32
	msg.SetType(message.NetSend)

	var bts []byte
	var bb *bytes.Buffer

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

	packets, err := s.packeter.Make(nil, bts, s.loss, s.reliability)
	if log.Error(err) {
		packets = [][]byte{bts}
	}

	packets = node.Shared(s.priv).SealPackets(encSharedTag, packets, nil, 0)

	pbPool.Put(pb)
	if bb != nil {
		bufpool.Put(bb)
	}

	s.net.SendAll(packets, node.ToAddr)
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
