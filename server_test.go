package overlay

import (
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/message"
	"github.com/dist-ribut-us/rnet"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"testing"
)

var portInc uint16 = 5555

func getPort() rnet.Port {
	portInc++
	return rnet.Port(portInc)
}

const loremIpsum = `Lorem ipsum dolor sit amet, consectetur adipiscing elit.
Nullam eu interdum nibh, vel malesuada nunc. Morbi sit amet augue finibus magna
interdum dictum. Donec tincidunt consectetur hendrerit. Praesent hendrerit
mauris vel erat accumsan, eu posuere augue interdum. Sed semper ut magna nec
molestie. Nullam accumsan metus vel arcu sodales rutrum. Duis nec malesuada ex,
nec tempor ante. Praesent pellentesque maximus turpis quis vulputate. Cras quis
tincidunt leo, in dapibus urna. Donec consectetur, erat nec eleifend accumsan,
risus mi egestas est, quis facilisis augue lacus a metus. Aliquam tincidunt sit
amet dui pellentesque suscipit. Aenean quis enim purus. Aliquam orci augue,
blandit eu convallis nec, laoreet vitae sapien. Donec metus tellus, placerat at
tempor in, posuere sit amet enim. Curabitur rhoncus mollis massa, vitae finibus
velit ultrices sit amet.`

func (s *Server) setIP(t *testing.T, ip string) {
	addr := s.net.Port().On(ip)
	if addr.Err != nil {
		t.Error(addr.Err)
	}
	s.addr = addr
}

func init() {
	log.Mute()
}

func unmute() {
	log.ToStdOut()
	log.SetDebug(true)
}

func mute() {
	log.Mute()
}

func TestCompress(t *testing.T) {
	// text should achieve a pretty high compression rate
	text := []byte(loremIpsum)
	gztxt := compress(gzTag, text).Bytes()
	assert.True(t, len(gztxt) < len(text))
	assert.Equal(t, GZipped, gztxt[0])

	bb, err := decompress(gztxt[1:])
	assert.NoError(t, err)
	txt := bb.Bytes()
	assert.Equal(t, text, txt)
}

func TestGetPBuffer(t *testing.T) {
	pb := getPBuffer([]byte{111})
	h := message.NewHeader(message.Test, "this is a test")
	pb.Marshal(h)
	b := pb.Bytes()
	assert.Equal(t, byte(111), b[0])
	h = &message.Header{}
	assert.NoError(t, proto.Unmarshal(b[1:], h))
	assert.Equal(t, message.Test, h.GetType())
	assert.Equal(t, "this is a test", string(h.Body))
}
