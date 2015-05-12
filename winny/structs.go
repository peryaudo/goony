package winny

import (
	"bytes"
	"crypto/rc4"
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
)

// Basic Winny data structures
// They might be going to be published externally.

type nodeAddr struct {
	IP   [4]byte
	Port int
}

func (n *nodeAddr) MarshalStream(w io.Writer) (err error) {
	writeLE(w, n.IP[:])
	writeLE(w, uint16(n.Port))
	return
}

func (n *nodeAddr) UnmarshalStream(r io.Reader) (err error) {
	err = readLE(r, n.IP[:])
	if err != nil {
		return
	}
	var port uint16
	err = readLE(r, &port)
	n.Port = int(port)
	return
}

type nodeInfo struct {
	Ver      int
	CertStr  string
	Ddns     string
	BbsPort  int
	Speed    int
	Clusters [3]string
	IsBbs    bool
}

type FileKey struct {
	Node      nodeAddr
	BbsNode   nodeAddr
	Size      uint32
	Hash      [16]byte
	FileName  string
	Trip      [11]byte
	BbsTrip   []byte
	Ttl       uint16
	RefCnt    uint32
	Timestamp uint32
	IsIgnored bool
	KeyVer    byte
}

func (k *FileKey) MarshalStream(w io.Writer) (err error) {
	k.Node.MarshalStream(w)
	k.BbsNode.MarshalStream(w)
	writeLE(w, k.Size)
	writeLE(w, k.Hash[:])

	fileName, err := toSjis(k.FileName)
	if err != nil {
		return
	}
	writeLE(w, byte(len(fileName)))

	var checksum uint32
	for _, by := range fileName {
		checksum += uint32(by)
	}
	writeLE(w, uint16(checksum&0xFFFF))

	cip, _ := rc4.NewCipher([]byte{byte(int(checksum) & 0xFF)})
	cip.XORKeyStream(fileName, fileName)

	writeLE(w, fileName)

	writeLE(w, k.Trip[:])
	writeLE(w, byte(len(k.BbsTrip)))
	writeLE(w, k.BbsTrip)
	writeLE(w, k.Ttl)
	writeLE(w, k.RefCnt)
	writeLE(w, k.Timestamp)
	writeLE(w, toByte(k.IsIgnored))
	writeLE(w, k.KeyVer)
	return
}

func (k *FileKey) UnmarshalStream(r io.Reader) (err error) {
	err = k.Node.UnmarshalStream(r)
	if err != nil {
		return
	}
	err = k.BbsNode.UnmarshalStream(r)
	if err != nil {
		return
	}
	err = readLE(r, &k.Size)
	if err != nil {
		return
	}
	err = readLE(r, k.Hash[:])
	if err != nil {
		return
	}

	var fileNameLen byte
	err = readLE(r, &fileNameLen)
	if err != nil {
		return
	}
	var checksum uint16
	err = readLE(r, &checksum)
	if err != nil {
		return
	}

	fileName := make([]byte, fileNameLen)
	err = readLE(r, fileName)
	if err != nil {
		return
	}

	rawFileName := make([]byte, fileNameLen)
	copy(rawFileName, fileName)

	cip, _ := rc4.NewCipher([]byte{byte(int(checksum) & 0xFF)})
	cip.XORKeyStream(fileName, fileName)

	var actual uint32
	for _, by := range fileName {
		actual += uint32(by)
	}
	if uint16(actual&0xFFFF) != checksum {
		return errors.New(
			fmt.Sprintf("invalid query file name checksum; checksum: %d rawFileName: %#v",
				checksum,
				rawFileName))
	}

	k.FileName, err = toUtf8(fileName)

	err = readLE(r, k.Trip[:])
	if err != nil {
		return
	}
	var bbsTripLen byte
	err = readLE(r, &bbsTripLen)
	if err != nil {
		return
	}
	k.BbsTrip = make([]byte, bbsTripLen)
	err = readLE(r, k.BbsTrip)
	if err != nil {
		return
	}
	err = readLE(r, &k.Ttl)
	if err != nil {
		return
	}
	err = readLE(r, &k.RefCnt)
	if err != nil {
		return
	}
	err = readLE(r, &k.Timestamp)
	if err != nil {
		return
	}
	var isIgnored byte
	err = readLE(r, &isIgnored)
	if err != nil {
		return
	}
	k.IsIgnored = isIgnored != 0
	err = readLE(r, &k.KeyVer)
	return
}

// Basic utility functions for marshaling and unmarshaling

func readLE(r io.Reader, data interface{}) error {
	return binary.Read(r, binary.LittleEndian, data)
}

func writeLE(w io.Writer, data interface{}) error {
	return binary.Write(w, binary.LittleEndian, data)
}

func toByte(bo bool) byte {
	if bo {
		return 1
	} else {
		return 0
	}
}

func toSjis(s string) (b []byte, err error) {
	var bs bytes.Buffer
	w := transform.NewWriter(&bs, japanese.ShiftJIS.NewEncoder())
	_, err = w.Write([]byte(s))
	b = bs.Bytes()
	return
}

func toUtf8(b []byte) (s string, err error) {
	r := transform.NewReader(bytes.NewReader(b), japanese.ShiftJIS.NewDecoder())
	raw, err := ioutil.ReadAll(r)
	s = string(raw)
	return
}
