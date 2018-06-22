package overlay

import (
	"github.com/dist-ribut-us/ipcrouter"
	"github.com/dist-ribut-us/overlay/overlaymessages"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetID(t *testing.T) {
	router, err := ipcrouter.New(getPort.Next())
	assert.NoError(t, err)
	s, err := NewServer(router, getPort.Next())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	s.RandomKey()
	go s.Run()

	router, err = ipcrouter.New(getPort.Next())
	assert.NoError(t, err)
	go router.Run()

	wait := make(chan bool)
	router.
		Query(overlaymessages.GetID, nil).
		To(s.router.Port()).
		SetService(overlaymessages.ServiceID).
		Send(func(r ipcrouter.Response) {
			id := overlaymessages.DeserializeID(r.GetBody())
			assert.Equal(t, s.key.Pub().Slice(), id.Sign.Slice())
			assert.Equal(t, s.keyX.Pub().Slice(), id.Xchng.Slice())
			wait <- true
		})

	select {
	case <-wait:
	case <-time.After(time.Millisecond * 100):
		t.Error("timeout")
	}
}
