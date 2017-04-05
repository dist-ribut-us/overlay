package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/rnet"
	"sync"
)

type portmap struct {
	Map map[uint32]rnet.Port
	sync.RWMutex
}

func newportmap() *portmap {
	return &portmap{
		Map: make(map[uint32]rnet.Port),
	}
}

func (t *portmap) get(key uint32) (rnet.Port, bool) {
	t.RLock()
	k, b := t.Map[key]
	t.RUnlock()
	return k, b
}

func (t *portmap) set(key uint32, val rnet.Port) {
	t.Lock()
	t.Map[key] = val
	t.Unlock()
}

func (t *portmap) delete(keys ...uint32) {
	t.Lock()
	for _, key := range keys {
		delete(t.Map, key)
	}
	t.Unlock()
}

type xchgPairs struct {
	Map map[string]*crypto.XchgPair
	sync.RWMutex
}

func newxchgPairs() *xchgPairs {
	return &xchgPairs{
		Map: make(map[string]*crypto.XchgPair),
	}
}

func (t *xchgPairs) get(key string) (*crypto.XchgPair, bool) {
	t.RLock()
	k, b := t.Map[key]
	t.RUnlock()
	return k, b
}

func (t *xchgPairs) set(key string, val *crypto.XchgPair) {
	t.Lock()
	t.Map[key] = val
	t.Unlock()
}

func (t *xchgPairs) delete(keys ...string) {
	t.Lock()
	for _, key := range keys {
		delete(t.Map, key)
	}
	t.Unlock()
}
