// Package winny provides Winny servent implementation.
package winny

import (
	"errors"
	"log"
)

// A Servent is an Winny servent.
// You should specify Speed and Port while others are optional.
type Servent struct {
	Speed    int
	Port     int
	Ddns     string
	Clusters [3]string

	recvCmd  chan *recvCmd
	nodeMgr  *nodeMgr
	queryMgr *queryMgr
}

// ListenAndServe starts Winny servent.
// It listens on the specified port while trying connecting other nodes.
// You can add other nodes explicitly by using AddNode().
func (s *Servent) ListenAndServe() (err error) {
	s.init()

	if s.Speed == 0 {
		err = errors.New("you must specify the node speed")
		return
	}

	if s.Port == 0 {
		err = errors.New("you must specify the listen port")
		return
	}

	go s.nodeMgr.ListenAndServe()
	go s.queryMgr.ListenAndServe()

	for recvCmd := range s.recvCmd {
		closeConn := false

		switch cmd := recvCmd.cmd.(type) {
		case *cmdAddr:
			s.nodeMgr.AddNode <- cmd
		case *cmdQuery:
			s.queryMgr.RecvQuery <- recvCmd

		case *cmdClose:
			closeConn = true
		case *cmdCloseTransLimit:
			closeConn = true
		case *cmdCloseBadPort0:
			closeConn = true
		case *cmdCloseIgnored:
			closeConn = true
		case *cmdCloseSlow:
			closeConn = true
		case *cmdCloseForgery:
			closeConn = true

		default:
			log.Printf("warning: unexpected command type %T\n", cmd)
		}

		if closeConn {
			s.nodeMgr.Disconnect <- recvCmd.From
		}
	}

	return
}

func (s *Servent) init() {
	if s.recvCmd != nil {
		return
	}

	s.recvCmd = make(chan *recvCmd)
	s.nodeMgr = newNodeMgr(s)
	s.queryMgr = newQueryMgr(s)
}

// AddNode adds other Winny nodes to the node list.
// The node string must be in the encrypted form (e.g. @fc259bdf....).
// Nodes with the private IP addresses are ignored.
// There's no guarantee that the servent will connect to the given nodes.
func (s *Servent) AddNode(node string) {
	s.init()

	go func() {
		s.nodeMgr.AddNodeStr <- node
	}()
}

//
func (s *Servent) Query(keyword string) <-chan *QueryResult {
	s.init()

	query := &queryReq{
		Keyword: keyword,
		Result:  make(chan *QueryResult)}

	go func() {
		s.queryMgr.AddQuery <- query
	}()

	return query.Result
}

// NodeList returns the complete node list the servent has.
// The returned strings are in the encrypted form.
func (s *Servent) NodeList() []string {
	ch := make(chan []string)
	s.nodeMgr.GetNodeList <- ch
	return <-ch
}
