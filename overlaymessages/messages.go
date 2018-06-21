package overlaymessages

import (
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/message"
)

const (
	GetID = message.Type(iota + message.ServiceTypeOffset)
)

const (
	ServiceID uint32 = 2864974
)

type ID struct {
	Sign  *crypto.SignPub
	Xchng *crypto.XchgPub
}

func (i *ID) Serialize() []byte {
	return append(i.Sign.Slice(), i.Xchng.Slice()...)
}

func DeserializeID(b []byte) *ID {
	return &ID{
		Sign:  crypto.SignPubFromSlice(b[:crypto.KeyLength]),
		Xchng: crypto.XchgPubFromSlice(b[crypto.KeyLength:]),
	}
}
