package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandshakeFormat(t *testing.T) {
	senderPub, senderPriv := crypto.GenerateKey()
	receiverPub, receiverPriv := crypto.GenerateKey()

	shared := receiverPub.Precompute(senderPriv)
	hs := buildHandshake(senderPub, shared)
	assert.Equal(t, hs[0], handshake)

	pub, sharedOut, ok := validateHandshake(hs[1:], receiverPriv)
	assert.True(t, ok)
	if pub == nil {
		t.Error("Should not be nil")
	} else {
		assert.Equal(t, senderPub.Slice(), pub.Slice())
		assert.Equal(t, sharedOut, shared)
	}
}
