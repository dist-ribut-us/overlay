package overlay

import (
	"crypto/rand"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const loremIpsum = `Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Nullam eu interdum nibh, vel malesuada nunc. Morbi sit amet augue finibus magna
interdum dictum. Donec tincidunt consectetur hendrerit. Praesent hendrerit
mauris vel erat accumsan, eu posuere augue interdum. Sed semper ut magna nec
molestie. Nullam accumsan metus vel arcu sodales rutrum. Duis nec malesuada ex,
nec tempor ante. Praesent pellentesque maximus turpis quis vulputate. Cras quis
tincidunt leo, in dapibus urna. Donec consectetur, erat nec eleifend accumsan,
risus mi egestas est, quis facilisis augue lacus a metus. Aliquam tincidunt sit
amet dui pellentesque suscipit. Aenean quis enim purus. Aliquam orci augue,
blandit eu convallis nec, laoreet vitae sapien. Donec metus tellus, placerat at
tempor in, posuere sit amet enim. Curabitur rhoncus mollis massa, vitae finibus
velit ultrices sit amet.`

func (s *Server) setIP(t *testing.T, ip string) {
	addr := s.net.Port().On(ip)
	if addr.Err != nil {
		t.Error(addr.Err)
	}
	s.addr = addr
}

func init() {
	log.To(nil)
}

func TestServer(t *testing.T) {
	ip := "127.0.0.1"
	proc1, err := ipc.New(2222)
	assert.NoError(t, err)
	s1, err := NewServer(proc1, 3222)
	assert.NoError(t, err)
	s1.setIP(t, ip)
	s1.packeter.Handler = nil
	defer s1.Close()
	proc2, err := ipc.New(2223)
	assert.NoError(t, err)
	s2, err := NewServer(proc2, 3223)
	assert.NoError(t, err)
	s2.setIP(t, ip)
	s2.packeter.Handler = nil
	defer s2.Close()

	s2Node := &Node{
		Pub:      s2.key.Pub(),
		FromAddr: s2.addr,
		ToAddr:   s2.addr,
	}
	s1.AddNode(s2Node)
	s1.Handshake(s2Node)

	ok := false
	for i := 0; i < 10 && !ok; i++ {
		time.Sleep(time.Millisecond)
		ok = len(s2.nByID) == 1
	}
	if !assert.True(t, ok) {
		return
	}

	s1Node, ok := s2.NodeByID(s1.key.Pub().ID())
	if !assert.True(t, ok) {
		return
	}
	msg := message.NewHeader(message.Test, make([]byte, 1000))
	rand.Read(msg.Body) // random data will not use compression
	s2.NetSend(msg, s1Node, true, s2.NetPort())

	select {
	case msgOut := <-s1.packeter.Chan():
		assert.NoError(t, msgOut.Err)
		h, err := s1.unmarshalNetMessage(msgOut)
		assert.NoError(t, err)
		assert.Equal(t, msg.Body, h.Body)
		assert.Equal(t, message.Test, h.GetType())
	case <-time.After(50 * time.Millisecond):
		t.Error("Timed out")
	}

	msg = message.NewHeader(message.Test, loremIpsum)

	s2.NetSend(msg, s1Node, true, s2.NetPort()) // loremIpusm will use compression
	select {
	case msgOut := <-s1.packeter.Chan():
		assert.NoError(t, msgOut.Err)
		h, err := s1.unmarshalNetMessage(msgOut)
		assert.NoError(t, err)
		assert.Equal(t, loremIpsum, string(h.Body))
		assert.Equal(t, message.Test, h.GetType())
	case <-time.After(50 * time.Millisecond):
		t.Error("Timed out")
	}
}

func TestCompress(t *testing.T) {
	// text should achieve a pretty high compression rate
	text := []byte(loremIpsum)
	gztxt := compress(gzTag, text).Bytes()
	assert.True(t, len(gztxt) < len(text))
	assert.Equal(t, GZipped, gztxt[0])

	bb, err := decompress(gztxt[1:])
	assert.NoError(t, err)
	txt := bb.Bytes()
	assert.Equal(t, text, txt)
}

func TestGetPBuffer(t *testing.T) {
	pb := getPBuffer([]byte{111})
	h := message.NewHeader(message.Test, "this is a test")
	pb.Marshal(h)
	b := pb.Bytes()
	assert.Equal(t, byte(111), b[0])
	h = &message.Header{}
	assert.NoError(t, proto.Unmarshal(b[1:], h))
	assert.Equal(t, message.Test, h.GetType())
	assert.Equal(t, "this is a test", string(h.Body))
}
