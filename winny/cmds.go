package winny

import (
	"bytes"
	"crypto/rc4"
	"errors"
)

// This file implements marshaling and unmarshaling of Winny commands.

type cmd interface {
	Idx() int
	Marshal() (b []byte, err error)
	Unmarshal(b []byte) (err error)
}

const (
	cmdIdxProtoHdr   = 0
	cmdIdxSpeed      = 1
	cmdIdxConnType   = 2
	cmdIdxSelfAddr   = 3
	cmdIdxAddr       = 4
	cmdIdxSpread     = 10
	cmdIdxCacheReq   = 11
	cmdIdxSpreadCond = 12
	cmdIdxQuery      = 13
	cmdIdxCacheRes   = 21

	cmdIdxClose           = 31
	cmdIdxCloseTransLimit = 32
	cmdIdxCloseBadPort0   = 33
	cmdIdxCloseIgnored    = 34
	cmdIdxCloseSlow       = 35
	cmdIdxCloseForgery    = 37

	cmdIdxCompat = 97
)

func (c *cmdProtoHdr) Idx() int   { return cmdIdxProtoHdr }
func (c *cmdSpeed) Idx() int      { return cmdIdxSpeed }
func (c *cmdConnType) Idx() int   { return cmdIdxConnType }
func (c *cmdSelfAddr) Idx() int   { return cmdIdxSelfAddr }
func (c *cmdAddr) Idx() int       { return cmdIdxAddr }
func (c *cmdSpread) Idx() int     { return cmdIdxSpread }
func (c *cmdCacheReq) Idx() int   { return cmdIdxCacheReq }
func (c *cmdSpreadCond) Idx() int { return cmdIdxSpreadCond }
func (c *cmdQuery) Idx() int      { return cmdIdxQuery }
func (c *cmdCacheRes) Idx() int   { return cmdIdxCacheRes }

func (c *cmdClose) Idx() int           { return cmdIdxClose }
func (c *cmdCloseTransLimit) Idx() int { return cmdIdxCloseTransLimit }
func (c *cmdCloseBadPort0) Idx() int   { return cmdIdxCloseBadPort0 }
func (c *cmdCloseIgnored) Idx() int    { return cmdIdxCloseIgnored }
func (c *cmdCloseSlow) Idx() int       { return cmdIdxCloseSlow }
func (c *cmdCloseForgery) Idx() int    { return cmdIdxCloseForgery }

func (c *cmdCompat) Idx() int { return cmdIdxCompat }

type cmdSpread struct{ cmdWithNoPayload }

type cmdClose struct{ cmdWithNoPayload }
type cmdCloseTransLimit struct{ cmdWithNoPayload }
type cmdCloseBadPort0 struct{ cmdWithNoPayload }
type cmdCloseIgnored struct{ cmdWithNoPayload }
type cmdCloseSlow struct{ cmdWithNoPayload }
type cmdCloseForgery struct{ cmdWithNoPayload }

type cmdCompat struct{ cmdWithNoPayload }

type cmdWithNoPayload struct {
}

func (c *cmdWithNoPayload) Marshal() (b []byte, err error) {
	b = []byte{}
	return
}

func (c *cmdWithNoPayload) Unmarshal(b []byte) (err error) {
	if len(b) > 0 {
		err = errors.New("payload not empty")
	}
	return
}

type cmdProtoHdr struct {
	Ver     int
	CertStr string
}

var protoHdrCertKey = []byte{0x39, 0x38, 0x37, 0x38, 0x39, 0x61, 0x73, 0x6a}

func (c *cmdProtoHdr) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, uint32(c.Ver))
	writeLE(&bs, []byte(c.CertStr))
	b = bs.Bytes()

	cip, _ := rc4.NewCipher(protoHdrCertKey)
	cip.XORKeyStream(b, b)
	return
}

func (c *cmdProtoHdr) Unmarshal(b []byte) (err error) {
	dec := make([]byte, len(b))
	cip, _ := rc4.NewCipher(protoHdrCertKey)
	cip.XORKeyStream(dec, b)

	bs := bytes.NewBuffer(dec[0:4])
	var ver uint32
	err = readLE(bs, &ver)
	if err != nil {
		return
	}
	c.Ver = int(ver)

	c.CertStr = string(dec[4:])
	return
}

type cmdSpeed struct {
	Speed int
}

func (c *cmdSpeed) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, float32(c.Speed))
	b = bs.Bytes()
	return
}

func (c *cmdSpeed) Unmarshal(b []byte) (err error) {
	bs := bytes.NewBuffer(b)
	var speed float32
	err = readLE(bs, &speed)
	c.Speed = int(speed)
	return
}

type cmdConnType struct {
	Type       int
	IsPort0    bool
	IsBadPort0 bool
	IsBbs      bool
}

const (
	connTypeSearch   = 0
	connTypeTransfer = 1
	connTypeBbs      = 2
)

func (c *cmdConnType) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, byte(c.Type))
	writeLE(&bs, toByte(c.IsPort0))
	writeLE(&bs, toByte(c.IsBadPort0))
	writeLE(&bs, toByte(c.IsBbs))
	b = bs.Bytes()
	return
}

func (c *cmdConnType) Unmarshal(b []byte) (err error) {
	var linkType, port0, badPort0, bbs byte

	bs := bytes.NewBuffer(b)
	err = readLE(bs, &linkType)
	if err != nil {
		return
	}
	err = readLE(bs, &port0)
	if err != nil {
		return
	}
	err = readLE(bs, &badPort0)
	if err != nil {
		return
	}
	err = readLE(bs, &bbs)
	if err != nil {
		return
	}

	c.Type = int(linkType)
	c.IsPort0 = port0 != 0
	c.IsBadPort0 = badPort0 != 0
	c.IsBbs = bbs != 0
	return
}

type cmdSelfAddr struct {
	IP       [4]byte
	Port     int
	Ddns     string
	Clusters [3]string
}

func (c *cmdSelfAddr) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, c.IP)
	writeLE(&bs, uint32(c.Port))

	// Convert DDNS and cluster strings to Shift-JIS
	ddns, err := toSjis(c.Ddns)
	if err != nil {
		return
	}

	var clusters [3][]byte
	for i := 0; i < 3; i++ {
		clusters[i], err = toSjis(c.Clusters[i])
		if err != nil {
			return
		}
	}

	// Write their lengths
	writeLE(&bs, byte(len(ddns)))
	for i := 0; i < 3; i++ {
		writeLE(&bs, byte(len(clusters[i])))
	}

	// Write their bytes
	writeLE(&bs, ddns)
	for i := 0; i < 3; i++ {
		writeLE(&bs, clusters[i])
	}

	b = bs.Bytes()
	return
}

func (c *cmdSelfAddr) Unmarshal(b []byte) (err error) {
	bs := bytes.NewBuffer(b)

	err = readLE(bs, c.IP[:])
	if err != nil {
		return
	}

	var port uint32
	err = readLE(bs, &port)
	if err != nil {
		return
	}
	c.Port = int(port)

	// Read lengths of DDNS and cluster strings

	var ddnsLen byte
	err = readLE(bs, &ddnsLen)
	if err != nil {
		return
	}

	var clusterLens [3]byte
	for i := 0; i < 3; i++ {
		err = readLE(bs, &clusterLens[i])
		if err != nil {
			return
		}
	}

	// Allocate byte arrays with the lengths

	ddns := make([]byte, ddnsLen)
	var clusters [3][]byte
	for i := 0; i < 3; i++ {
		clusters[i] = make([]byte, clusterLens[i])
	}

	// Read the strings

	err = readLE(bs, ddns)
	if err != nil {
		return
	}
	for i := 0; i < 3; i++ {
		err = readLE(bs, clusters[i])
		if err != nil {
			return
		}
	}

	// Convert them into UTF-8

	c.Ddns, err = toUtf8(ddns)
	if err != nil {
		return
	}

	for i := 0; i < 3; i++ {
		c.Clusters[i], err = toUtf8(clusters[i])
		if err != nil {
			return
		}
	}

	return
}

type cmdAddr struct {
	IP       [4]byte
	Port     int
	BbsPort  int
	IsBbs    bool
	Speed    int
	Clusters [3]string
}

func (c *cmdAddr) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, c.IP)
	writeLE(&bs, uint32(c.Port))
	writeLE(&bs, uint32(c.BbsPort))
	writeLE(&bs, toByte(c.IsBbs))
	writeLE(&bs, uint32(c.Speed))

	// Convert cluster strings to Shift-JIS
	var clusters [3][]byte
	for i := 0; i < 3; i++ {
		clusters[i], err = toSjis(c.Clusters[i])
		if err != nil {
			return
		}
	}

	// Write their lengths
	for i := 0; i < 3; i++ {
		writeLE(&bs, byte(len(clusters[i])))
	}

	// Write their bytes
	for i := 0; i < 3; i++ {
		writeLE(&bs, clusters[i])
	}

	b = bs.Bytes()
	return
}

func (c *cmdAddr) Unmarshal(b []byte) (err error) {
	bs := bytes.NewBuffer(b)

	var port, bbsPort, speed uint32

	err = readLE(bs, c.IP[:])
	if err != nil {
		return
	}
	err = readLE(bs, &port)
	if err != nil {
		return
	}
	c.Port = int(port)
	err = readLE(bs, &bbsPort)
	if err != nil {
		return
	}
	c.BbsPort = int(bbsPort)
	var bbs byte
	err = readLE(bs, &bbs)
	if err != nil {
		return
	}
	c.IsBbs = bbs != 0
	err = readLE(bs, &speed)
	if err != nil {
		return
	}
	c.Speed = int(speed)

	var clusterLens [3]byte
	for i := 0; i < 3; i++ {
		err = readLE(bs, &clusterLens[i])
		if err != nil {
			return
		}
	}

	var clusters [3][]byte
	for i := 0; i < 3; i++ {
		clusters[i] = make([]byte, clusterLens[i])
	}
	for i := 0; i < 3; i++ {
		err = readLE(bs, clusters[i])
	}

	for i := 0; i < 3; i++ {
		c.Clusters[i], err = toUtf8(clusters[i])
		if err != nil {
			return
		}
	}
	return
}

type cmdCacheReq struct {
	Id       uint32
	BeginIdx uint32
	Num      uint32
	Hash     [16]byte
	Size     uint32
}

func (c *cmdCacheReq) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, c.Id)
	writeLE(&bs, c.BeginIdx)
	writeLE(&bs, c.Num)
	writeLE(&bs, c.Hash[:])
	writeLE(&bs, c.Size)
	b = bs.Bytes()
	return
}

func (c *cmdCacheReq) Unmarshal(b []byte) (err error) {
	bs := bytes.NewBuffer(b)
	err = readLE(bs, &c.Id)
	if err != nil {
		return
	}
	err = readLE(bs, &c.BeginIdx)
	if err != nil {
		return
	}
	err = readLE(bs, &c.Num)
	if err != nil {
		return
	}
	err = readLE(bs, c.Hash[:])
	if err != nil {
		return
	}
	err = readLE(bs, &c.Size)
	return
}

// ??? It seems both pyny and poeny are not taking things seriously for the implementations of cmdSpreadCond
type cmdSpreadCond struct {
	// Keyword string
	Keyword [256]byte
	Trip    [16]byte
	Id      uint32
}

func (c *cmdSpreadCond) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	/*
		keyword, err := toSjis(c.Keyword)
		var expanded [256]byte
		copy(expanded[:], keyword)

		writeLE(&bs, expanded)
	*/
	writeLE(&bs, c.Keyword)
	writeLE(&bs, c.Trip)
	writeLE(&bs, c.Id)
	b = bs.Bytes()
	return
}

func (c *cmdSpreadCond) Unmarshal(b []byte) (err error) {
	bs := bytes.NewBuffer(b)

	/*
		var keyword [256]byte
		err = readLE(bs, keyword[:])
		if err != nil {
			return
		}

		keywordLen := 0
		for i := 0; i < len(keyword); i++ {
			if keyword[i] == 0 {
				break
			}
			keywordLen++
		}
		c.Keyword, err = toUtf8(keyword[0:keywordLen])
	*/
	err = readLE(bs, c.Keyword[:])
	if err != nil {
		return
	}

	err = readLE(bs, c.Trip[:])
	if err != nil {
		return
	}
	err = readLE(bs, &c.Id)
	return
}

type cmdQuery struct {
	IsReply      bool
	IsSpread     bool
	IsDownstream bool
	IsBbs        bool
	Id           uint32
	Keyword      string
	Trip         [11]byte
	Nodes        []nodeAddr
	Keys         []fileKey
}

func (c *cmdQuery) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, toByte(c.IsReply))
	writeLE(&bs, toByte(c.IsSpread))
	writeLE(&bs, toByte(c.IsDownstream))
	writeLE(&bs, toByte(c.IsBbs))
	writeLE(&bs, c.Id)

	keyword, err := toSjis(c.Keyword)
	if err != nil {
		return
	}
	writeLE(&bs, byte(len(keyword)))
	writeLE(&bs, keyword)
	writeLE(&bs, c.Trip[:])

	writeLE(&bs, byte(len(c.Nodes)))
	for _, n := range c.Nodes {
		n.MarshalStream(&bs)
	}

	writeLE(&bs, uint16(len(c.Keys)))
	for _, k := range c.Keys {
		k.MarshalStream(&bs)
	}

	b = bs.Bytes()
	return
}

func (c *cmdQuery) Unmarshal(b []byte) (err error) {
	var reply, spread, downstream, bbs byte

	bs := bytes.NewBuffer(b)
	err = readLE(bs, &reply)
	if err != nil {
		return
	}
	err = readLE(bs, &spread)
	if err != nil {
		return
	}
	err = readLE(bs, &downstream)
	if err != nil {
		return
	}
	err = readLE(bs, &bbs)
	if err != nil {
		return
	}

	c.IsReply = reply != 0
	c.IsSpread = spread != 0
	c.IsDownstream = downstream != 0
	c.IsBbs = bbs != 0

	err = readLE(bs, &c.Id)
	if err != nil {
		return
	}

	var keywordLen byte
	err = readLE(bs, &keywordLen)
	if err != nil {
		return
	}
	keyword := make([]byte, keywordLen)
	err = readLE(bs, keyword[:])
	if err != nil {
		return
	}
	c.Keyword, err = toUtf8(keyword)
	if err != nil {
		return
	}

	err = readLE(bs, c.Trip[:])
	if err != nil {
		return
	}

	var nodesLen byte
	err = readLE(bs, &nodesLen)
	if err != nil {
		return
	}

	c.Nodes = make([]nodeAddr, nodesLen)
	for i := 0; i < int(nodesLen); i++ {
		err = c.Nodes[i].UnmarshalStream(bs)
		if err != nil {
			return
		}
	}

	var keysLen uint16
	err = readLE(bs, &keysLen)
	if err != nil {
		return
	}
	c.Keys = make([]fileKey, keysLen)
	for i := 0; i < int(keysLen); i++ {
		err = c.Keys[i].UnmarshalStream(bs)
		if err != nil {
			return
		}
	}

	return
}

type cmdCacheRes struct {
	Id       uint32
	BeginIdx uint32
	Hash     [16]byte
	Data     [0x10000]byte
}

func (c *cmdCacheRes) Marshal() (b []byte, err error) {
	var bs bytes.Buffer
	writeLE(&bs, c.Id)
	writeLE(&bs, c.BeginIdx)
	writeLE(&bs, c.Hash[:])
	writeLE(&bs, c.Data[:])
	b = bs.Bytes()
	return
}

func (c *cmdCacheRes) Unmarshal(b []byte) (err error) {
	bs := bytes.NewBuffer(b)
	err = readLE(bs, &c.Id)
	if err != nil {
		return
	}
	err = readLE(bs, &c.BeginIdx)
	if err != nil {
		return
	}
	err = readLE(bs, c.Hash[:])
	if err != nil {
		return
	}
	err = readLE(bs, c.Data[:])
	return
}
