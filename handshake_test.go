package overlay

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandshakeFormat(t *testing.T) {
	ax := crypto.GenerateXchgPair()
	_, as := crypto.GenerateSignPair()

	hs := buildHandshake(handshakeRequest, ax.Pub(), as)
	assert.Equal(t, hs[0], handshakeRequest)

	s, x, ok := validateHandshake(hs, nil)
	assert.True(t, ok)
	if assert.NotNil(t, s) && assert.NotNil(t, x) {
		assert.Equal(t, as.Pub(), s)
		assert.Equal(t, ax.Pub(), x)
	}

	s, x, ok = validateHandshake(hs, as.Pub())
	assert.True(t, ok)
	if assert.NotNil(t, s) && assert.NotNil(t, x) {
		assert.Equal(t, as.Pub(), s)
		assert.Equal(t, ax.Pub(), x)
	}

	_, other := crypto.GenerateSignPair()
	_, _, ok = validateHandshake(hs, other.Pub())
	assert.False(t, ok)
}
