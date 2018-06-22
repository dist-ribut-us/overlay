package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/ipcrouter"
	"github.com/dist-ribut-us/ipcrouter/testservice"
	"github.com/dist-ribut-us/log"
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

	// setup service and overlay for A
	serviceA, err := testservice.New(314159, getPort.Next())
	assert.NoError(t, err)
	go serviceA.Run()
	defer serviceA.Close()
	overlayProcA, err := ipcrouter.New(getPort.Next())
	assert.NoError(t, err)
	overlaySrvA, err := NewServer(overlayProcA, getPort.Next())
	assert.NoError(t, err)
	overlaySrvA.setIP(t, ip)
	overlaySrvA.RandomKey()
	go overlaySrvA.Run()
	defer overlaySrvA.Close()
	serviceA.NetSenderPort = overlayProcA.Port()

	// setup service and overlay for B
	serviceB, err := testservice.New(265358, getPort.Next())
	assert.NoError(t, err)
	go serviceB.Run()
	defer serviceB.Close()
	overlayProcB, err := ipcrouter.New(getPort.Next())
	assert.NoError(t, err)
	overlaySrvB, err := NewServer(overlayProcB, getPort.Next())
	assert.NoError(t, err)
	overlaySrvB.setIP(t, ip)
	overlaySrvB.RandomKey()
	go overlaySrvB.Run()
	defer overlaySrvB.Close()

	log.Info(serviceA.Port(), overlayProcA.Port(), overlaySrvA.net.Port())
	log.Info(serviceB.Port(), overlayProcB.Port(), overlaySrvB.net.Port())

	// serviceA is going to make a request from serviceB, in order for overlayB
	// to know how to route the message, serviceB needs to register with overlayB.
	// RegisterWithOverlay is a helper method to do this.
	serviceB.RegisterWithOverlay(serviceB.ServiceID(), overlaySrvB.router.Port())

	// overlayA needs to know about nodeB before it can send the handshake
	nodeB := &node{
		Pub:      overlaySrvB.key.Pub(),
		FromAddr: overlaySrvB.addr,
		ToAddr:   overlaySrvB.addr,
	}
	overlaySrvA.addNode(nodeB)

	// Send the query from A to B
	serviceA.Router.
		Query(message.Test, "query_from_A").
		SetService(serviceB.ServiceID()).
		SendToNet(nodeB.ToAddr, serviceA.NetResponder)

	// check that B receives the query
	select {
	case nq := <-serviceB.Chan.NetQuery:
		assert.Equal(t, "query_from_A", nq.BodyString())
		nq.Respond("resp_from_B")
	case <-time.After(time.Millisecond * 10):
		t.Error("time out")
	}

	// check that A recieves the response
	// TODO: Fix this - need to wire up a NetResponse
	select {
	case r := <-serviceA.Chan.NetResponse:
		assert.Equal(t, "resp_from_B", r.BodyString())
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
