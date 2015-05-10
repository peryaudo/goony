package winny

import (
	"crypto/rc4"
	"encoding/hex"
	"errors"
	"strings"
)

var nodeStrKey = []byte{0x70, 0x69, 0x65, 0x77, 0x66, 0x36, 0x61, 0x73, 0x63, 0x78, 0x6c, 0x76}

func DecryptNodeString(s string) (addr string, err error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 || s[0] != '@' {
		err = errors.New("node string does not start with @")
		return
	}

	b, err := hex.DecodeString(s[1:])
	if err != nil {
		return
	}
	if len(b) == 0 {
		err = errors.New("invalid node string length")
		return
	}

	key := append([]byte{b[0]}, nodeStrKey...)
	cip, err := rc4.NewCipher(key)
	if err != nil {
		return
	}

	plain := make([]byte, len(b)-1)
	cip.XORKeyStream(plain, b[1:])

	var checksum byte
	for _, by := range plain {
		checksum += by
	}

	if checksum != b[0] {
		err = errors.New("invalid node string checksum")
		return
	}

	addr = strings.TrimSpace(string(plain))
	return
}

func EncryptNodeString(addr string) (s string, err error) {
	addr = strings.TrimSpace(addr)

	b := []byte(addr)
	if len(b) == 0 {
		err = errors.New("invalid err length")
		return
	}

	var checksum byte
	for _, by := range b {
		checksum += by
	}

	key := append([]byte{checksum}, nodeStrKey...)
	cip, err := rc4.NewCipher(key)
	if err != nil {
		return
	}

	cip.XORKeyStream(b, b)

	b = append([]byte{checksum}, b...)

	s = "@" + hex.EncodeToString(b)
	return
}
