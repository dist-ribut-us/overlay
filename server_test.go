package overlay

import (
	"github.com/dist-ribut-us/ipc"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	proc1, err := ipc.New(2222)
	assert.NoError(t, err)
	s1, err := NewServer(proc1, "127.0.0.1")
	assert.NoError(t, err)
	proc2, err := ipc.New(2223)
	assert.NoError(t, err)
	s2, err := NewServer(proc2, "127.0.0.1")
	assert.NoError(t, err)

	s2Addr := s2.Server.Port().On("127.0.0.1")
	assert.NoError(t, s2Addr.Err)
	s2Node := &Node{
		Pub:      s2.pub,
		FromAddr: s2Addr,
		ToAddr:   s2Addr,
	}
	s1.AddNode(s2Node)
	s1.Handshake(s2Node)

	time.Sleep(time.Millisecond * 5)
	assert.True(t, s2.IsRunning())

	assert.Equal(t, 1, len(s2.nById))

	msg := []byte("Hello, server 1")
	s1Node, ok := s2.NodeById(s1.pub.GetID())
	if assert.True(t, ok) {
		s1Addr := s1.Server.Port().On("127.0.0.1")
		assert.NoError(t, s1Addr.Err)
		s1Node.ToAddr = s1Addr
		s2.Send(msg, s1Node)

		select {
		case msgOut := <-s1.packeter.Chan():
			assert.NoError(t, msgOut.Err)
			assert.Equal(t, msg, msgOut.Body)
		case <-time.After(50 * time.Millisecond):
			t.Error("Timed out")
		}
	}
}
