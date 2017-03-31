package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandshakeFormat(t *testing.T) {
	senderXPub, _ := crypto.GenerateXchgKeypair()
	senderSPub, senderSPriv := crypto.GenerateSignKeypair()

	hs := buildHandshake(handshakeRequest, senderXPub, senderSPriv)
	assert.Equal(t, hs[0], handshakeRequest)

	spub, xpub, ok := validateHandshake(hs, nil)
	assert.True(t, ok)
	if assert.NotNil(t, spub) && assert.NotNil(t, xpub) {
		assert.Equal(t, spub, senderSPub)
		assert.Equal(t, xpub, senderXPub)
	}
}
