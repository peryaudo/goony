package winny

import (
	"errors"
	"log"
	"math"
	"math/rand"
	"net"
	"runtime"
	"strconv"
	"time"
)

// nodeMgr manages node list and connections.
// nodeMgr listens on the servent port and tries connecting other nodes to satisfy the criteria.
// Each connection is a separated goroutine running the instance of nodeConn.
type nodeMgr struct {
	servent *Servent

	// Send a command to nodes with matching conditions.
	SendCmd chan *sendCmd

	// Add nodes to the list.
	AddNode     chan *cmdAddr
	AddNodeAddr chan nodeAddr
	AddNodeStr  chan string

	// Disconnect from the node.
	Disconnect chan nodeAddr

	// Returns complete node list in the encrypted form.
	GetNodeList chan chan []string

	established chan *establishedConn
	closed      chan *closedConn

	connNodes map[nodeAddr]*nodeConn
	nodes     map[nodeAddr]*nodeInfo

	addConnTrying chan struct{}
	subConnTrying chan struct{}
	connTrying    int
}

type establishedConn struct {
	*nodeConn

	Addr     nodeAddr
	PrevAddr nodeAddr

	ProtoHdr *cmdProtoHdr
	Speed    *cmdSpeed
	ConnType *cmdConnType
	SelfAddr *cmdSelfAddr
}

type closedConn struct {
	Addr nodeAddr

	Reason error
}

type recvCmd struct {
	From           nodeAddr
	FromDownstream bool

	cmd
}

type sendCmd struct {
	To        *nodeAddr // nil if the target is not specific
	Direction int       // Ignored if To is not nil

	cmd
}

const (
	directionAll = iota
	directionUp
	directionDown
)

func newNodeMgr(s *Servent) *nodeMgr {
	return &nodeMgr{
		servent:     s,
		SendCmd:     make(chan *sendCmd),
		AddNode:     make(chan *cmdAddr),
		AddNodeAddr: make(chan nodeAddr),
		AddNodeStr:  make(chan string),
		Disconnect:  make(chan nodeAddr),
		GetNodeList: make(chan chan []string),
		established: make(chan *establishedConn),
		closed:      make(chan *closedConn),
		connNodes:   make(map[nodeAddr]*nodeConn),
		nodes:       make(map[nodeAddr]*nodeInfo),

		// Simultaneous connection trial limit
		addConnTrying: make(chan struct{}),
		subConnTrying: make(chan struct{}),
		connTrying:    8}
}

func (m *nodeMgr) listen() {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(m.servent.Port))
	if err != nil {
		log.Println(err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		go nodeConnAccept(conn, m)
	}

}

func (m *nodeMgr) ListenAndServe() {
	go m.listen()

	manageTick := time.Tick(4 * time.Second)

	for {
		select {
		case <-manageTick:
			m.manageNodeConn()
			m.manageNodeList()

		case sendcmd := <-m.SendCmd:
			m.selectAndSend(sendcmd)

		case cmdaddr := <-m.AddNode:
			m.addNode(cmdaddr)

		case nodeaddr := <-m.AddNodeAddr:
			if !isPrivateIP(nodeaddr.IP[:]) && m.nodes[nodeaddr] == nil {
				m.nodes[nodeaddr] = &nodeInfo{}
			}

		case nodestr := <-m.AddNodeStr:
			m.addNodeStr(nodestr)

		case nodeaddr := <-m.Disconnect:
			if m.connNodes[nodeaddr] != nil {
				m.connNodes[nodeaddr].Close()
			}

		case listChan := <-m.GetNodeList:
			nodeStrs := make([]string, len(m.nodes))
			i := 0
			for addr, _ := range m.nodes {
				nodeStr, _ := EncryptNodeString(
					net.IP(addr.IP[:]).String() + ":" + strconv.Itoa(addr.Port))
				nodeStrs[i] = nodeStr
				i++
			}
			listChan <- nodeStrs

		case est := <-m.established:
			m.addEstablishedNode(est)

		case cls := <-m.closed:
			delete(m.connNodes, cls.Addr)

		case <-m.addConnTrying:
			m.connTrying++

		case <-m.subConnTrying:
			m.connTrying--
		}
	}
}

func (m *nodeMgr) selectAndSend(sendcmd *sendCmd) {
	if sendcmd.To != nil {
		conn := m.connNodes[*(sendcmd.To)]
		if conn == nil {
			return
		}
		conn.Send(sendcmd.cmd)

		return
	}

	sent := 0
	for _, conn := range m.connNodes {
		switch sendcmd.Direction {
		case directionAll:
			conn.Send(sendcmd.cmd)
			sent++
		case directionUp:
			if !conn.IsDownstream {
				conn.Send(sendcmd.cmd)
				sent++
			}
		case directionDown:
			if conn.IsDownstream {
				conn.Send(sendcmd.cmd)
				sent++
			}
		}
	}

	// log.Printf("sent command: %#v to %d nodes\n", sendcmd.cmd, sent)
}

func (m *nodeMgr) addNode(c *cmdAddr) {
	key := nodeAddr{IP: c.IP, Port: c.Port}

	if m.nodes[key] == nil {
		m.nodes[key] = &nodeInfo{}
	}
	m.nodes[key].BbsPort = c.BbsPort
	m.nodes[key].IsBbs = c.IsBbs
	m.nodes[key].Speed = c.Speed
	m.nodes[key].Clusters = c.Clusters
}

func (m *nodeMgr) addNodeStr(nodestr string) {
	addr, err := DecryptNodeString(nodestr)
	if err != nil {
		log.Println(err)
		return
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Println(err)
		return
	}

	ip := net.ParseIP(host).To4()
	if len(ip) != 4 {
		log.Println("invalid IP address (DDNS not supported): " + host)
		return
	}

	if isPrivateIP(ip) {
		return
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		log.Println(err)
		return
	}

	var key nodeAddr
	copy(key.IP[:], ip)
	key.Port = portInt

	m.nodes[key] = &nodeInfo{}
}

func (m *nodeMgr) addEstablishedNode(est *establishedConn) {
	m.connNodes[est.Addr] = est.nodeConn

	prevInfo := m.nodes[est.PrevAddr]
	delete(m.nodes, est.PrevAddr)
	if m.nodes[est.Addr] == nil {
		m.nodes[est.Addr] = prevInfo
	}

	if m.nodes[est.Addr] == nil {
		m.nodes[est.Addr] = &nodeInfo{}
	}
	m.nodes[est.Addr].Ver = est.ProtoHdr.Ver
	m.nodes[est.Addr].CertStr = est.ProtoHdr.CertStr
	m.nodes[est.Addr].Speed = est.Speed.Speed
	m.nodes[est.Addr].Ddns = est.SelfAddr.Ddns
	m.nodes[est.Addr].Clusters = est.SelfAddr.Clusters

	// log.Printf("established connection: %#v\n", m.nodes[est.Addr])
}

func (m *nodeMgr) cmdSpeed() *cmdSpeed {
	return &cmdSpeed{Speed: m.servent.Speed}
}

func (m *nodeMgr) cmdConnType() *cmdConnType {
	return &cmdConnType{
		Type:       connTypeSearch,
		IsPort0:    false,
		IsBadPort0: false,
		IsBbs:      false}
}

func (m *nodeMgr) cmdSelfAddr(IP []byte) (c *cmdSelfAddr) {
	c = &cmdSelfAddr{
		Port: m.servent.Port,
		Ddns: m.servent.Ddns}
	copy(c.IP[:], IP)
	copy(c.Clusters[:], m.servent.Clusters[:])
	return
}

func (m *nodeMgr) manageNodeConn() {
	upstream := 0
	downstream := 0
	for _, conn := range m.connNodes {
		if conn.Type != connTypeSearch {
			continue
		}
		if conn.IsDownstream {
			downstream++
		} else {
			upstream++
		}
	}

	if rand.Intn(8) == 0 {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		log.Printf("up: %d + down: %d = %d, known: %d trial remain: %d goroutines: %d total/used = %f\n",
			upstream,
			downstream,
			len(m.connNodes),
			len(m.nodes),
			m.connTrying,
			runtime.NumGoroutine(),
			float32(ms.TotalAlloc)/float32(ms.Alloc))
	}

	if upstream < 2 {
		// Try connecting upstream nodes.
		for i := 0; i < m.connTrying; i++ {
			k, err := m.selectNewNode()
			if err == nil {
				go nodeConnDial(k, m)
			} else {
				log.Println(err)
				break
			}
		}
	}

	if upstream > 3 {
		// Disconnect the one with shortest connecting time
		m.disconnShortest( /*isDownstream = */ false)
	}

	if downstream > 5 {
		// Disconnect the one with shortest connecting time
		m.disconnShortest( /*isDownstream = */ true)
	}
}

func (m *nodeMgr) disconnShortest(isDownstream bool) {
	var dur time.Duration = math.MaxInt64
	var cand *nodeConn
	for _, conn := range m.connNodes {
		if conn.Type != connTypeSearch {
			continue
		}
		if conn.IsDownstream == isDownstream {
			cur := time.Since(conn.Since)
			if cur < dur {
				dur = cur
				cand = conn
			}
		}
	}
	if cand != nil {
		cand.Close()
	}
}

func (m *nodeMgr) selectNewNode() (k nodeAddr, err error) {
	// TODO(peryaudo): evaluate nodes by clustering words and try from the top

	for key, info := range m.nodes {
		if _, ok := m.connNodes[key]; ok {
			continue
		}

		if info.Speed != 0 {
			localSpeed := float32(m.servent.Speed)
			remoteSpeed := float32(info.Speed)
			if localSpeed/remoteSpeed > 20.0 || remoteSpeed/localSpeed > 20.0 {
				continue
			}
		}

		// TODO(peryaudo): delay delete after connection failure
		delete(m.nodes, key)

		k = key
		return
	}

	err = errors.New("no selectable new node candidate")
	return
}

func (m *nodeMgr) manageNodeList() {
	// TODO(peryaudo): limit the number of entries in Nodes to 600 and remove far nodes
	return
}

func isPrivateIP(ip []byte) bool {
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}
	return false
}
