package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/message"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

// TestQueryResponse is the main test. It simulates two services communicating
// through overlay nodes.
func TestQueryResponse(t *testing.T) {
	ip := "127.0.0.1"
	var serviceBID uint32 = 31415926

	// setup service and overlay for A
	serviceA, err := ipc.RunNew(getPort())
	assert.NoError(t, err)
	defer serviceA.Close()
	overlayProcA, err := ipc.New(getPort())
	assert.NoError(t, err)
	overlaySrvA, err := NewServer(overlayProcA, getPort())
	assert.NoError(t, err)
	overlaySrvA.setIP(t, ip)
	overlaySrvA.RandomKey()
	go overlaySrvA.Run()
	defer overlaySrvA.Close()

	// setup service and overlay for B
	serviceB, err := ipc.RunNew(getPort())
	assert.NoError(t, err)
	defer serviceB.Close()
	overlayProcB, err := ipc.New(getPort())
	assert.NoError(t, err)
	overlaySrvB, err := NewServer(overlayProcB, getPort())
	assert.NoError(t, err)
	overlaySrvB.setIP(t, ip)
	overlaySrvB.RandomKey()
	go overlaySrvB.Run()
	defer overlaySrvB.Close()

	// serviceA is going to make a request from serviceB, in order for overlayB
	// to know how to route the message, serviceB needs to register with overlayB.
	// RegisterWithOverlay is a helper method to do this.
	serviceB.RegisterWithOverlay(serviceBID, overlaySrvB.ipc.Port())

	// overlayA needs to know about nodeB before it can send the handshake
	nodeB := &node{
		Pub:      overlaySrvB.key.Pub(),
		FromAddr: overlaySrvB.addr,
		ToAddr:   overlaySrvB.addr,
	}
	overlaySrvA.addNode(nodeB)

	// Before sending the request from A, setup the handler in B
	out := make(chan string)
	serviceB.Handler(func(msg *ipc.Base) {
		out <- msg.BodyString() + ":to_B"
		msg.Respond("resp_from_B")
	})

	// Send the query from A to B
	serviceA.
		Query(message.Test, []byte("query_from_A")).
		ToNet(overlaySrvA.ipc.Port(), nodeB.ToAddr, serviceBID).
		Send(func(r *ipc.Base) {
			assert.Equal(t, message.Test, r.GetType())
			assert.True(t, r.IsResponse())
			out <- string(r.Body) + ":to_A"
		})

	// check that B receives the query
	select {
	case s := <-out:
		assert.Equal(t, "query_from_A:to_B", s)
	case <-time.After(time.Millisecond * 10):
		t.Error("time out")
	}

	// check that A recieves the response
	select {
	case s := <-out:
		assert.Equal(t, "resp_from_B:to_A", s)
	case <-time.After(time.Millisecond * 10):
		t.Error("time out")
	}

	// check that both TTL values were set after the handshake
	b, ok := overlaySrvA.nodeByID(overlaySrvB.key.Pub().ID())
	assert.True(t, ok)
	var i int
	for i, ok = 0, b.TTL > 0; !ok && i < 10; i, ok = i+1, b.TTL > 0 {
		// Check for valid TTL value once per millisecond up to 10ms
		time.Sleep(time.Millisecond)
	}
	assert.True(t, ok)

	a, ok := overlaySrvB.nodeByID(overlaySrvA.key.Pub().ID())
	assert.True(t, ok)
	assert.True(t, a.TTL > 0)

	dirStr := "testDir"
	os.RemoveAll(dirStr)
	defer os.RemoveAll(dirStr)
	key := crypto.RandomSymmetric()

	err = overlaySrvA.Forest(key, dirStr)
	assert.NoError(t, err)

	static, err := overlaySrvA.GetStaticKey()
	assert.NoError(t, err)
	assert.False(t, static)

	oldkey := overlaySrvA.key
	overlaySrvA.key = nil
	overlaySrvA.SetStaticKey(true)
	static, err = overlaySrvA.GetStaticKey()
	assert.NoError(t, err)
	assert.True(t, static)
	overlaySrvA.SetKey()
	assert.NotEqual(t, oldkey, overlaySrvA.key)
	oldkey = overlaySrvA.key
	overlaySrvA.SetKey()
	assert.Equal(t, oldkey, overlaySrvA.key)

	overlaySrvA.SetStaticKey(false)
	static, err = overlaySrvA.GetStaticKey()
	assert.NoError(t, err)
	assert.False(t, static)
	overlaySrvA.SetKey()
	assert.NotEqual(t, oldkey, overlaySrvA.key)
}
