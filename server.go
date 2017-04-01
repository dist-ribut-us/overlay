package overlay

import (
	"github.com/dist-ribut-us/crypto"
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
	nByID       map[string]*Node
	nByAddr     map[string]*Node
	beacons     []*beacon
	mtxNodes    sync.RWMutex
	loss        float64
	reliability float64
	addr        *rnet.Addr
	services    map[uint32]rnet.Port
	callbacks   map[uint32]rnet.Port
	forest      *merkle.Forest
	xchgCache   map[string]*crypto.XchgPair
	cacheMux    sync.RWMutex
}

// NewServer creates an Overlay Server. The server starts off running. An
// overlay server can route messages from the network to local programs and send
// messages from local programs to the network.
func NewServer(proc *ipc.Proc, netPort rnet.Port) (*Server, error) {
	_, key := crypto.GenerateSignPair()

	srv := &Server{
		key:         key,
		packeter:    packeter.New(),
		ipc:         proc,
		nByID:       make(map[string]*Node),
		nByAddr:     make(map[string]*Node),
		loss:        0.01,
		reliability: 0.999,
		services:    make(map[uint32]rnet.Port),
		callbacks:   make(map[uint32]rnet.Port),
		xchgCache:   make(map[string]*crypto.XchgPair),
	}

	var err error
	srv.net, err = rnet.RunNew(netPort, srv)
	go proc.Run()

	srv.packeter.Handler = srv.handleNetMessage
	srv.ipc.Handler(srv.handleIPCMessage)

	return srv, err
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

	log.Info(log.Lbl("IPC>"), s.ipc.Port().On("127.0.0.1"), log.Lbl("Net>"), addr, s.PubStr())
}

// NodeByAddr gets a node using an address
func (s *Server) NodeByAddr(addr *rnet.Addr) (*Node, bool) {
	s.mtxNodes.RLock()
	node, ok := s.nByAddr[addr.String()]
	s.mtxNodes.RUnlock()
	return node, ok
}

// NodeByID gets a node using a crypto.ID
func (s *Server) NodeByID(id *crypto.ID) (*Node, bool) {
	s.mtxNodes.RLock()
	node, ok := s.nByID[id.String()]
	s.mtxNodes.RUnlock()
	return node, ok
}

// AddNode will add a node to the server
func (s *Server) AddNode(node *Node) *Server {
	id := node.Pub.ID().String()
	s.mtxNodes.Lock()
	s.nByID[id] = node
	if node.FromAddr != nil {
		s.nByAddr[node.FromAddr.String()] = node
	}
	s.mtxNodes.Unlock()
	return s
}

// PubStr get the public key as as string
func (s *Server) PubStr() string {
	if s.key == nil {
		return ""
	}
	return s.key.Pub().String()
}

// NetPort gets the network facing port
func (s *Server) NetPort() rnet.Port { return s.net.Port() }

// IPCPort gets the port used to communicate with other processes
func (s *Server) IPCPort() rnet.Port { return s.ipc.Port() }

// Close stop all processes for the overlay server
func (s *Server) Close() {
	s.net.Close()
	s.ipc.Close()
}
