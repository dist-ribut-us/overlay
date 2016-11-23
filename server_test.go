package overlay

import (
	"github.com/dist-ribut-us/rnet"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s1, err := NewServer(":2222")
	assert.NoError(t, err)
	s2, err := NewServer(":2223")
	assert.NoError(t, err)

	s2Addr, err := rnet.ResolveAddr("127.0.0.1:2223")
	assert.NoError(t, err)
	s2Node := &Node{
		Pub:      s2.pub,
		FromAddr: s2Addr,
		ToAddr:   s2Addr,
	}
	s1.AddNode(s2Node)
	s1.Handshake(s2Node)

	time.Sleep(time.Millisecond * 5)

	assert.Equal(t, 1, len(s2.nById))

	msg := []byte("Hello, server 1")
	s1Node, ok := s2.NodeById(s1.pub.GetID())
	if assert.True(t, ok) {
		s1Addr, err := rnet.ResolveAddr("127.0.0.1:2222")
		assert.NoError(t, err)
		s1Node.ToAddr = s1Addr
		s2.Send(msg, s1Node)

		select {
		case msgOut := <-s1.packeter.Chan():
			assert.NoError(t, msgOut.Err)
			assert.Equal(t, msg, msgOut.Body)
		case <-time.After(5 * time.Millisecond * 5):
			t.Error("Timed out")
		}
	}
}
