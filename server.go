package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/packeter"
	"github.com/dist-ribut-us/rnet"
	"sync"
)

type Server struct {
	*rnet.Server
	pub         *crypto.Pub
	priv        *crypto.Priv
	id          *crypto.ID
	packeter    *packeter.Packeter
	ipc         *ipc.Proc
	nById       map[string]*Node
	nByAddr     map[string]*Node
	mtxNodes    sync.RWMutex
	loss        float64
	reliability float64
	addr        *rnet.Addr
}

// NewServer creates an Overlay Server. The server starts off running. An
// overlay server can route messages from the network to local programs and send
// messages from local programs to the network.
func NewServer(proc *ipc.Proc, ip string) (*Server, error) {
	pub, priv := crypto.GenerateKey()

	port := rnet.RandomPort()
	addr := port.On(ip)
	if addr.Err != nil {
		return nil, addr.Err
	}

	srv := &Server{
		pub:         pub,
		priv:        priv,
		id:          pub.GetID(),
		packeter:    packeter.New(),
		ipc:         proc,
		nById:       make(map[string]*Node),
		nByAddr:     make(map[string]*Node),
		loss:        0.01,
		reliability: 0.999,
		addr:        addr,
	}

	var err error
	srv.Server, err = rnet.RunNew(port, srv)
	go proc.Run()

	return srv, err
}

func (s *Server) NodeByAddr(addr *rnet.Addr) (*Node, bool) {
	s.mtxNodes.RLock()
	node, ok := s.nByAddr[addr.String()]
	s.mtxNodes.RUnlock()
	return node, ok
}

func (s *Server) NodeById(id *crypto.ID) (*Node, bool) {
	s.mtxNodes.RLock()
	node, ok := s.nById[id.String()]
	s.mtxNodes.RUnlock()
	return node, ok
}

func (s *Server) AddNode(node *Node) {
	id := node.Pub.GetID().String()
	s.mtxNodes.Lock()
	s.nById[id] = node
	if node.FromAddr != nil {
		s.nByAddr[node.FromAddr.String()] = node
	}
	s.mtxNodes.Unlock()
}

func (s *Server) Chan() <-chan *packeter.Message { return s.packeter.Chan() }
func (s *Server) PubStr() string {
	if s.pub == nil {
		return ""
	}
	return s.pub.String()
}

func (s *Server) IPCChan() <-chan *ipc.Message { return s.ipc.Chan() }
