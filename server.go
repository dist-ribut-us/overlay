package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/errors"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/merkle"
	"github.com/dist-ribut-us/natt/igdp"
	"github.com/dist-ribut-us/packeter"
	"github.com/dist-ribut-us/rnet"
	"sync"
)

// Server represents an overlay server.
type Server struct {
	net         *rnet.Server
	key         *crypto.SignPriv
	packeter    *packeter.Packeter
	ipc         *ipc.Proc
	nByID       map[string]*node
	nByAddr     map[string]*node
	beacons     []*node
	nodesMux    sync.RWMutex
	loss        float64
	reliability float64
	addr        *rnet.Addr
	services    *portmap
	callbacks   *portmap
	forest      *merkle.Forest
	xchgCache   *xchgPairs
	NodeTTL     uint32 // default TTL in seconds
}

// NewServer creates an Overlay Server. The server starts off running. An
// overlay server can route messages from the network to local programs and send
// messages from local programs to the network.
func NewServer(proc *ipc.Proc, netPort rnet.Port) (*Server, error) {
	s := &Server{
		packeter:    packeter.New(),
		ipc:         proc,
		nByID:       make(map[string]*node),
		nByAddr:     make(map[string]*node),
		loss:        0.01,
		reliability: 0.999,
		services:    newportmap(),
		callbacks:   newportmap(),
		xchgCache:   newxchgPairs(),
		NodeTTL:     60 * 60, // one hour
	}
	s.services.set(serviceID, proc.Port())
	s.packeter.Handler = s.handleNetMessage
	s.ipc.Handler(s.handleIPCMessage)
	var err error
	s.net, err = rnet.New(netPort, s)
	return s, err
}

// SetKey will set the key on the server. If a forest has been initilized, it
// will check if the server is configured to use a static key and use that if
// so. Otherwise it will set a random key.
func (s *Server) SetKey() error {
	if s.forest == nil {
		s.RandomKey()
		return nil
	}
	static, err := s.GetStaticKey()
	if err != nil {
		return err
	}
	if static {
		return s.LoadKey()
	}
	s.RandomKey()
	return nil
}

// RandomKey sets the servers signing key to a random key value
func (s *Server) RandomKey() {
	_, s.key = crypto.GenerateSignPair()
}

var configBkt = []byte("config")
var keykey = []byte("key")
var statickey = []byte("statickey")

// ErrNoForest is returned when attempting to perform Overlay operations that
// require a storage forest before one is initilized
const ErrNoForest = errors.String("Overlay does not have forest")

// SetStaticKey sets the static key config value. If set to true, the current
// key value is saved.
func (s *Server) SetStaticKey(val bool) error {
	if s.forest == nil {
		return ErrNoForest
	}
	var b byte
	if val {
		b = 1
	}
	err := s.forest.SetValue(configBkt, statickey, []byte{b})
	if err != nil {
		return err
	}
	if !val {
		return nil
	}
	if s.key == nil {
		s.RandomKey()
	}
	return s.forest.SetValue(configBkt, keykey, s.key.Slice())
}

// GetStaticKey returns the current config value of statickey
func (s *Server) GetStaticKey() (bool, error) {
	if s.forest == nil {
		return false, ErrNoForest
	}
	val, err := s.forest.GetValue(configBkt, statickey)
	if err != nil {
		return false, err
	}
	if len(val) < 1 {
		return false, nil
	}
	return val[0] == 1, nil
}

// LoadKey will load the signing key from the forest, if one exists, otherwise
// it will create a random key and save it.
func (s *Server) LoadKey() error {
	if s.forest == nil {
		return ErrNoForest
	}
	val, err := s.forest.GetValue(configBkt, keykey)
	if err != nil {
		return err
	}

	if val == nil {
		_, s.key = crypto.GenerateSignPair()
		return s.forest.SetValue(configBkt, keykey, s.key.Slice())
	}
	s.key = crypto.SignPrivFromSlice(val)
	return nil
}

// Run the overlay server
func (s *Server) Run() {
	go s.net.Run()
	s.ipc.Run()
}

// Forest opens the merkle forest for the overlay server.
func (s *Server) Forest(key *crypto.Symmetric, dir string) (err error) {
	s.forest, err = merkle.Open(dir, key)
	s.forest.MakeBuckets(configBkt)
	return
}

// SetupNetwork tries to connect to the network.
func (s *Server) SetupNetwork() {
	if err := igdp.Setup(); err == nil {
		_, err = igdp.AddPortMapping(s.net.Port(), s.net.Port())
		log.Error(err)
	}
	ip, err := igdp.GetExternalIP()
	log.Error(err)

	addr := s.net.Port().On(ip)
	log.Error(addr.Err)
	s.addr = addr

	log.Info(log.Lbl("IPC>"), s.ipc.Port().On("127.0.0.1"), log.Lbl("Net>"), addr, s.key.Pub())
}

func (s *Server) nodeByAddr(addr *rnet.Addr) (*node, bool) {
	s.nodesMux.RLock()
	n, ok := s.nByAddr[addr.String()]
	s.nodesMux.RUnlock()
	return n, ok
}

func (s *Server) nodeByID(id *crypto.ID) (*node, bool) {
	s.nodesMux.RLock()
	n, ok := s.nByID[id.String()]
	s.nodesMux.RUnlock()
	return n, ok
}

func (s *Server) addNode(n *node) *Server {
	id := n.Pub.ID().String()
	s.nodesMux.Lock()
	s.nByID[id] = n
	if n.FromAddr != nil {
		s.nByAddr[n.FromAddr.String()] = n
	}
	s.nodesMux.Unlock()
	return s
}

// Close stop all processes for the overlay server
func (s *Server) Close() {
	log.Info("closing_overlay_server")
	s.net.Close()
	s.ipc.Close()
}
