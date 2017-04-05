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
	services    map[uint32]rnet.Port
	servicesMux sync.RWMutex
	callbacks   map[uint32]rnet.Port
	callbackMux sync.RWMutex
	forest      *merkle.Forest
	xchgCache   map[string]*crypto.XchgPair
	cacheMux    sync.RWMutex
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
		services:    map[uint32]rnet.Port{serviceID: proc.Port()},
		callbacks:   make(map[uint32]rnet.Port),
		xchgCache:   make(map[string]*crypto.XchgPair),
		NodeTTL:     60 * 60, // one hour
	}
	s.packeter.Handler = s.handleNetMessage
	s.ipc.Handler(s.handleIPCMessage)
	var err error
	s.net, err = rnet.New(netPort, s)
	return s, err
}

func (s *Server) RandomKey() {
	_, s.key = crypto.GenerateSignPair()
}

var configBkt = []byte("config")
var keykey = []byte("key_______")

const ErrNoForest = errors.String("Overlay does not have forest")

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
		s.forest.SetValue(configBkt, keykey, s.key.Slice())
	} else {
		s.key = crypto.SignPrivFromSlice(val)
	}
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
