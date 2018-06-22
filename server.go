package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/errors"
	"github.com/dist-ribut-us/ipcrouter"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/merkle"
	"github.com/dist-ribut-us/natt/igdp"
	"github.com/dist-ribut-us/overlay/overlaymessages"
	"github.com/dist-ribut-us/packeter"
	"github.com/dist-ribut-us/rnet"
)

// Server represents an overlay server.
type Server struct {
	*nodes
	net         *rnet.Server
	key         *crypto.SignPriv
	keyX        *crypto.XchgPair // Temporary until github.com/golang/go/issues/20504
	packeter    *packeter.Packeter
	router      *ipcrouter.Router
	loss        float64
	reliability float64
	addr        *rnet.Addr
	services    *portmap
	callbacks   *portmap
	forest      *merkle.Forest
	xchgCache   *xchgPairs
	NodeTTL     uint32 // default TTL in seconds
}

// NewServer initilizes part of the Overlay Server.
func NewServer(router *ipcrouter.Router, netPort rnet.Port) (*Server, error) {
	// The server does not have a key when it starts. The relationship with pool
	// is setup so that pool should send a message telling it how to load a key
	// before any network communication starts.
	s := &Server{
		nodes:       newNodes(),
		packeter:    packeter.New(),
		router:      router,
		loss:        0.01,
		reliability: 0.999,
		services:    newportmap(),
		callbacks:   newportmap(),
		xchgCache:   newxchgPairs(),
		NodeTTL:     60 * 60, // one hour
	}
	s.services.set(overlaymessages.ServiceID, router.Port())
	s.packeter.Handler = s.handleNetMessage
	s.router.Register(s)
	var err error
	s.net, err = rnet.New(netPort, s)
	return s, err
}

// ServiceID for Overlay service
func (*Server) ServiceID() uint32 {
	return overlaymessages.ServiceID
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
	s.keyX = crypto.GenerateXchgPair()
}

// Port returns the ipc router port for Overlay
func (s *Server) Port() rnet.Port {
	return s.router.Port()
}

var (
	configBkt = []byte("config")
	keykey    = []byte("key")
	keyXkey   = []byte("keyX")
	statickey = []byte("statickey")
)

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
	return s.forest.SetValue(configBkt, keyXkey, s.keyX.Slice())
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
	keyB, err := s.forest.GetValue(configBkt, keykey)
	if err != nil {
		return err
	}
	keyXB, err := s.forest.GetValue(configBkt, keyXkey)
	if err != nil {
		return err
	}

	if keyB == nil || keyXB == nil {
		s.SetStaticKey(true)
		return nil
	}

	s.key = crypto.SignPrivFromSlice(keyB)
	s.keyX = crypto.XchgPairFromSlice(keyXB)

	return nil
}

// Run the overlay server
func (s *Server) Run() {
	go s.net.Run()
	s.router.Run()
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

	log.Info(log.Lbl("IPC>"), s.router.Port().On("127.0.0.1"), log.Lbl("Net>"), addr, s.key.Pub())
}

// Close stop all processes for the overlay server
func (s *Server) Close() {
	log.Info(log.Lbl("closing_overlay_server"), log.KV{"net", s.net.Port()}, log.KV{"local", s.router.Port()})
	s.net.Close()
	s.router.Close()
}
