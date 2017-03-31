package overlay

import (
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestQueryResponse(t *testing.T) {
	serviceA, err := ipc.RunNew(1234)
	assert.NoError(t, err)
	defer serviceA.Close()
	serviceB, err := ipc.RunNew(1235)
	assert.NoError(t, err)
	defer serviceB.Close()

	out := make(chan string)
	serviceB.Handler(func(msg *ipc.Base) {
		log.Info("msg_B_id", msg.Id)
		out <- msg.BodyString() + ":to_B"
		msg.Respond("resp_from_B")
	})

	ip := "127.0.0.1"
	var serviceBID uint32 = 31415926

	overlayProcA, err := ipc.New(2234)
	assert.NoError(t, err)
	overlaySrvA, err := NewServer(overlayProcA, 3234)
	assert.NoError(t, err)
	overlaySrvA.setIP(t, ip)
	defer overlaySrvA.Close()

	overlayProcB, err := ipc.New(2235)
	assert.NoError(t, err)
	overlaySrvB, err := NewServer(overlayProcB, 3235)
	assert.NoError(t, err)
	overlaySrvB.setIP(t, ip)
	defer overlaySrvB.Close()

	serviceB.RegisterWithOverlay(serviceBID, overlaySrvB.IPCPort())

	nodeB := &Node{
		Pub:      overlaySrvB.key.Pub(),
		FromAddr: overlaySrvB.addr,
		ToAddr:   overlaySrvB.addr,
	}
	overlaySrvA.AddNode(nodeB)
	overlaySrvA.Handshake(nodeB)

	ok := false
	for i := 0; i < 10 && !ok; i++ {
		time.Sleep(time.Millisecond)
		ok = len(overlaySrvB.nByID) == 1
	}
	assert.True(t, ok)

	nodeA, ok := overlaySrvB.NodeByID(overlaySrvA.key.Pub().ID())
	if !assert.True(t, ok) {
		return
	}
	assert.NotNil(t, nodeA)

	serviceA.
		Query(message.Test, []byte("query_from_A")).
		ToNet(overlaySrvA.IPCPort(), nodeB.ToAddr, serviceBID).
		Send(func(r *ipc.Base) {
			assert.Equal(t, message.Test, r.GetType())
			assert.True(t, r.IsResponse())
			out <- string(r.Body) + ":to_A"
		})

	select {
	case s := <-out:
		assert.Equal(t, "query_from_A:to_B", s)
	case <-time.After(time.Millisecond * 10):
		t.Error("time out")
	}

	select {
	case s := <-out:
		assert.Equal(t, "resp_from_B:to_A", s)
	case <-time.After(time.Millisecond * 10):
		t.Error("time out")
	}
}
