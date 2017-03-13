package overlay

import (
	"crypto/rand"
	"github.com/dist-ribut-us/ipc"
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

func TestServer(t *testing.T) {
	ip := "127.0.0.1"
	proc1, err := ipc.New(2222)
	assert.NoError(t, err)
	s1, err := NewServer(proc1)
	assert.NoError(t, err)
	s1.setIP(t, ip)
	proc2, err := ipc.New(2223)
	assert.NoError(t, err)
	s2, err := NewServer(proc2)
	assert.NoError(t, err)
	s2.setIP(t, ip)

	s2Node := &Node{
		Pub:      s2.pub,
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
	assert.True(t, ok)

	if s1Node, ok := s2.NodeByID(s1.pub.GetID()); assert.True(t, ok) {
		msg := make([]byte, 1000)
		rand.Read(msg) // random data will not use compression
		s2.NetSend(msg, s1Node)

		select {
		case msgOut := <-s1.netChan:
			assert.NoError(t, msgOut.Err)
			assert.Equal(t, msg, msgOut.Body)
		case <-time.After(50 * time.Millisecond):
			t.Error("Timed out")
		}

		s2.NetSend([]byte(loremIpsum), s1Node) // loremIpusm will use compression
		select {
		case msgOut := <-s1.netChan:
			assert.NoError(t, msgOut.Err)
			assert.Equal(t, []byte(loremIpsum), msgOut.Body)
		case <-time.After(50 * time.Millisecond):
			t.Error("Timed out")
		}
	}
}

func TestCompress(t *testing.T) {
	// text should achieve a pretty high compression rate
	text := []byte(loremIpsum)
	gztxt := compress(text)
	assert.True(t, len(gztxt) < len(text))
	assert.Equal(t, gzipped, gztxt[0])

	txt, err := decompress(gztxt)
	assert.NoError(t, err)
	assert.Equal(t, text, txt)
}
