package winny

import (
	"crypto/rand"
	"crypto/rc4"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

type nodeConn struct {
	mgr *nodeMgr

	nodeAddr nodeAddr

	conn *rc4Conn

	IsNat        bool
	IsDownstream bool
	cmdConnType

	Since time.Time
}

func nodeConnAccept(conn net.Conn, m *nodeMgr) {
	c := &nodeConn{
		mgr:          m,
		conn:         &rc4Conn{Raw: conn},
		IsDownstream: true}
	c.establish()
}

func nodeConnDial(nodeAddr nodeAddr, m *nodeMgr) {
	m.subConnTrying <- struct{}{}
	defer func() {
		m.addConnTrying <- struct{}{}
	}()

	ip := net.IP(nodeAddr.IP[:]).String()
	port := strconv.Itoa(nodeAddr.Port)

	conn, err := net.DialTimeout("tcp", ip+":"+port, 10*time.Second)
	if err != nil {
		m.closed <- &closedConn{Addr: nodeAddr, Reason: err}
		return
	}

	c := &nodeConn{
		mgr:          m,
		nodeAddr:     nodeAddr,
		conn:         &rc4Conn{Raw: conn},
		IsDownstream: false}
	c.establish()
}

func (c *nodeConn) Send(cmd cmd) (err error) {
	payload, err := cmd.Marshal()
	if err != nil {
		return
	}

	length := uint32(len(payload) + 1)
	idx := byte(cmd.Idx())

	err = writeLE(c.conn, length)
	if err != nil {
		return
	}
	err = writeLE(c.conn, idx)
	if err != nil {
		return
	}
	err = writeLE(c.conn, payload)
	return
}

func (c *nodeConn) Close() error {
	return c.conn.Raw.Close()
}

func (c *nodeConn) establish() {
	defer c.Close()
	var err error

	err = c.sendHandshake()
	if err != nil {
		c.mgr.closed <- &closedConn{
			Addr:   c.nodeAddr,
			Reason: errors.New(fmt.Sprintf("sendHandshake failed: %v", err))}
		return
	}
	e, err := c.recvHandshake()
	if err != nil {
		c.mgr.closed <- &closedConn{
			Addr:   c.nodeAddr,
			Reason: errors.New(fmt.Sprintf("recvHandshake failed: %v", err))}
		return
	}

	c.Since = time.Now()

	if e.Addr.IP != e.SelfAddr.IP {
		c.IsNat = true
	}

	// If the remote is within 20% faster or slower to the local,
	// acceptor is upstream and connector is downstream.
	// Otherwise, slower node is downstream while faster node is upstream.
	localSpeed := float32(c.mgr.servent.Speed)
	remoteSpeed := float32(e.Speed.Speed)

	if remoteSpeed < localSpeed*0.8 || localSpeed*1.2 < remoteSpeed {
		c.IsDownstream = remoteSpeed < localSpeed
	}

	c.mgr.established <- e

	for {
		cmd, err := c.recv()
		if err != nil {
			c.mgr.closed <- &closedConn{
				Addr:   c.nodeAddr,
				Reason: errors.New(fmt.Sprintf("recv failed: %v", err))}
			return
		}
		c.mgr.servent.recvCmd <- &recvCmd{
			FromDownstream: c.IsDownstream,
			From:           c.nodeAddr,
			cmd:            cmd}
	}
}

func (c *nodeConn) sendHandshake() (err error) {
	var rnd [6]byte
	rand.Read(rnd[:])

	err = writeLE(c.conn, rnd[:])
	if err != nil {
		return err
	}

	// Ignore the top two bytes of the generated random bytes as RC4 key
	key := rnd[2:]
	c.conn.WCip, _ = rc4.NewCipher(strlenWorkaround(key))

	err = c.Send(&cmdCompat{})
	if err != nil {
		return
	}

	err = c.Send(&cmdProtoHdr{
		Ver:     12710,
		CertStr: "Winny Ver2.0b1 (goony)"})
	if err != nil {
		return
	}

	// Shuffle the key
	for i := 1; i < 4; i++ {
		key[i] ^= 0x39
	}
	c.conn.WCip, _ = rc4.NewCipher(strlenWorkaround(key))

	err = c.Send(c.mgr.cmdSpeed())
	if err != nil {
		return
	}
	err = c.Send(c.mgr.cmdConnType())
	if err != nil {
		return
	}
	err = c.Send(c.mgr.cmdSelfAddr(c.localIP()))
	return
}

func (c *nodeConn) recvHandshake() (e *establishedConn, err error) {
	e = &establishedConn{nodeConn: c, PrevAddr: c.nodeAddr}

	var rnd [6]byte
	err = readLE(c.conn, rnd[:])
	if err != nil {
		return
	}

	key := rnd[2:]
	c.conn.RCip, _ = rc4.NewCipher(strlenWorkaround(key))

	var cmd cmd

	cmd, err = c.recv()
	if err != nil {
		return
	}
	_, ok := cmd.(*cmdCompat)
	if !ok {
		err = errors.New("receiving cmdCompat failed")
		return
	}

	cmd, err = c.recv()
	if err != nil {
		return
	}
	e.ProtoHdr, ok = cmd.(*cmdProtoHdr)
	if !ok {
		err = errors.New("receiving cmdProtoHdr failed")
		return
	}

	// Shuffle the key
	for i := 1; i < 4; i++ {
		key[i] ^= 0x39
	}
	c.conn.RCip, _ = rc4.NewCipher(strlenWorkaround(key))

	cmd, err = c.recv()
	if err != nil {
		return
	}
	e.Speed, ok = cmd.(*cmdSpeed)
	if !ok {
		err = errors.New("receiving cmdSpeed failed")
		return
	}

	cmd, err = c.recv()
	if err != nil {
		return
	}
	e.ConnType, ok = cmd.(*cmdConnType)
	if !ok {
		err = errors.New("receiving cmdConnType failed")
		return
	}
	c.cmdConnType = *e.ConnType

	cmd, err = c.recv()
	if err != nil {
		return
	}
	e.SelfAddr, ok = cmd.(*cmdSelfAddr)
	if !ok {
		err = errors.New("receiving cmdSelfAddr failed")
		return
	}

	// Update port of the Addr
	copy(e.Addr.IP[:], c.remoteIP())
	e.Addr.Port = e.SelfAddr.Port

	return
}

// TODO(peryaudo): instead of directly forwarding cmdQuery, replace private IP with node's global IP.
// TODO(peryaudo): set deadline to all the recvs
func (c *nodeConn) recv() (cmd cmd, err error) {
	var length uint32
	var idx byte

	err = readLE(c.conn, &length)
	if err != nil {
		return
	}
	err = readLE(c.conn, &idx)
	if err != nil {
		return
	}

	switch idx {
	case cmdIdxProtoHdr:
		cmd = &cmdProtoHdr{}
	case cmdIdxSpeed:
		cmd = &cmdSpeed{}
	case cmdIdxConnType:
		cmd = &cmdConnType{}
	case cmdIdxSelfAddr:
		cmd = &cmdSelfAddr{}
	case cmdIdxAddr:
		cmd = &cmdAddr{}
	case cmdIdxSpread:
		cmd = &cmdSpread{}
	case cmdIdxCacheReq:
		cmd = &cmdCacheReq{}
	case cmdIdxSpreadCond:
		cmd = &cmdSpreadCond{}
	case cmdIdxQuery:
		cmd = &cmdQuery{}
	case cmdIdxCacheRes:
		cmd = &cmdCacheRes{}

	case cmdIdxClose:
		cmd = &cmdClose{}
	case cmdIdxCloseTransLimit:
		cmd = &cmdCloseTransLimit{}
	case cmdIdxCloseBadPort0:
		cmd = &cmdCloseBadPort0{}
	case cmdIdxCloseIgnored:
		cmd = &cmdCloseIgnored{}
	case cmdIdxCloseSlow:
		cmd = &cmdCloseSlow{}
	case cmdIdxCloseForgery:
		cmd = &cmdCloseForgery{}

	case cmdIdxCompat:
		cmd = &cmdCompat{}

	default:
		err = errors.New("Invalid command index")
		return
	}

	if cmd.Idx() != int(idx) {
		panic("command index mismatching!")
	}

	if idx == cmdIdxCacheRes && length > 70*1024*1024 {
		err = errors.New("payload too long")
	}
	if idx != cmdIdxCacheRes && length > 1*1024*1024 {
		err = errors.New("payload too long")
	}
	if err != nil {
		return
	}

	payload := make([]byte, length-1)
	err = readLE(c.conn, payload)
	if err != nil {
		return
	}

	err = cmd.Unmarshal(payload)

	if err != nil {
		if cmd.Idx() == cmdIdxQuery {
			err = errors.New(fmt.Sprintf("command parsing error: %v type: %T", err, cmd))
		} else {
			err = errors.New(fmt.Sprintf("command parsing error: %v type: %T payload: %#v", err, cmd, payload))
		}
	}

	// FIXME: DBG
	if err == nil && cmd.Idx() != cmdIdxQuery {
		marshaled, err := cmd.Marshal()
		if err != nil || len(marshaled) != len(payload) {
			log.Printf("failed recv payload: %#v", payload)
			panic(fmt.Sprintf("remarshaling failed %#v", cmd))
		}
		for i := 0; i < len(marshaled); i++ {
			if marshaled[i] != payload[i] {
				panic(fmt.Sprintf("remarshaling failed %#v", cmd))
			}
		}
	}
	// FIXME: DBG END

	return
}

func (c *nodeConn) localIP() []byte {
	str := c.conn.Raw.LocalAddr().String()
	host, _, _ := net.SplitHostPort(str)
	return net.ParseIP(host).To4()
}

func (c *nodeConn) remoteIP() []byte {
	str := c.conn.Raw.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(str)
	return net.ParseIP(host).To4()
}

// Imitating Winny's infamous usage of strlen
func strlenWorkaround(b []byte) []byte {
	if b[0] == 0 {
		return b[0:1]
	}
	for i := 1; i < len(b); i++ {
		if b[i] == 0 {
			return b[0:i]
		}
	}
	return b
}

type rc4Conn struct {
	RCip *rc4.Cipher
	WCip *rc4.Cipher
	Raw  net.Conn
}

func (c *rc4Conn) Read(b []byte) (n int, err error) {
	n, err = c.Raw.Read(b)
	if c.RCip != nil {
		c.RCip.XORKeyStream(b[:n], b[:n])
	}
	return
}

func (c *rc4Conn) Write(b []byte) (n int, err error) {
	enc := make([]byte, len(b))
	if c.WCip != nil {
		c.WCip.XORKeyStream(enc, b)
	}
	n, err = c.Raw.Write(enc)
	return
}
