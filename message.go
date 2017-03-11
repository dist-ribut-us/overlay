package overlay

import (
	"bytes"
	"compress/gzip"
	"github.com/dist-ribut-us/errors"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/rnet"
)

const (
	gzipped = byte(1 << iota)
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

// NetSend sends a message over the network
func (s *Server) NetSend(msg []byte, node *Node) {
	msg = compress(msg)
	pkts, err := s.packeter.Make(msg, s.loss, s.reliability)
	if log.Error(errors.Wrap("while sending message", err)) {
		return
	}
	shared := node.Shared(s.priv)

	pkts = shared.SealPackets([]byte{message}, pkts, nil)

	s.net.SendAll(pkts, node.ToAddr)
}

func compress(msg []byte) []byte {
	b := bytes.NewBuffer([]byte{gzipped})
	w := gzip.NewWriter(b)
	_, err := w.Write(msg)
	if log.Error(err) {
		w.Close()
		return append([]byte{0}, msg...)
	}
	if log.Error(w.Close()) {
		return append([]byte{0}, msg...)
	}
	zmsg := b.Bytes()
	if len(zmsg) >= len(msg) {
		return append([]byte{0}, msg...)
	}
	return zmsg
}

func decompress(zmsg []byte) ([]byte, error) {
	if zmsg[0]&gzipped != gzipped {
		return zmsg[1:], nil
	}
	r, err := gzip.NewReader(bytes.NewBuffer(zmsg[1:]))
	if err != nil {
		return nil, err
	}
	err = r.Close()
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	_, err = b.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (s *Server) unzip() {
	for msg := range s.packeter.Chan() {
		if msg.Err == nil {
			msg.Body, msg.Err = decompress(msg.Body)
		}
		s.netChan <- msg
	}
}
