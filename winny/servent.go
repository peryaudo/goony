// Package winny provides Winny servent implementation.
package winny

import (
	"errors"
	"log"
)

// An Winny servent.
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

// Starts Winny servent.
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

	// Start node and query managers in other goroutines
	go s.nodeMgr.ListenAndServe()
	go s.queryMgr.ListenAndServe()

	// Dispatches a received command to corresponding modules
	for recvCmd := range s.recvCmd {
		// true if the received command indicates disconnection request
		closeConn := false

		switch cmd := recvCmd.cmd.(type) {
		case *cmdAddr:
			s.nodeMgr.AddNode <- cmd
		case *cmdQuery:
			s.queryMgr.RecvQuery <- recvCmd

			// All the commands below indicates disconnection request
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
			// Request node manager to disconnect the sender node
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

// Adds other Winny nodes to the node list.
// The node string must be in the encrypted form (e.g. @fc259bdf....).
// Nodes with the private IP addresses are ignored.
// There's no guarantee that the servent will connect to the given nodes.
func (s *Servent) AddNode(node string) {
	s.init()

	// The function is unblocking because there's no guarantee that the servent is already started.
	go func() {
		s.nodeMgr.AddNodeStr <- node
	}()
}

type queryReq struct {
	Keyword string
	Results chan *FileKey
}

// Returns a channel that sends the file keys that match the keyword.
// If the keyword is an empty string, the channel streams all the file keys.
// Sending something to the quit channel stops the result stream.
func (s *Servent) Search(keyword string) (results <-chan *FileKey, quit chan<- struct{}) {
	s.init()

	q := &queryReq{
		Keyword: keyword,
		Results: make(chan *FileKey)}

	// The function is unblocking because there's no guarantee that the servent is already started.
	go func() {
		s.queryMgr.AddQuery <- q
	}()

	qu := make(chan struct{})

	go func() {
		// Wait until quit channel receives something and remove the query
		<-qu
		s.queryMgr.RemoveQuery <- q.Results
	}()

	results = q.Results
	quit = qu
	return
}

// Returns a channel that streams all the searching keywords flowing through the network.
// Sending something to the quit channel stops the result stream.
func (s *Servent) KeywordStream() (keywords <-chan string, quit chan<- struct{}) {
	s.init()

	k := make(chan string)
	qu := make(chan struct{})

	// The function is unblocking because there's no guarantee that the servent is already started.
	go func() {
		s.queryMgr.AddKeywordStream <- k
	}()

	go func() {
		// Wait until quit channel receives something and remove the keyword stream
		<-qu
		s.queryMgr.RemoveKeywordStream <- k
	}()

	keywords = k
	quit = qu
	return
}

// Returns the complete node list the servent has.
// The returned strings are in the encrypted form.
func (s *Servent) NodeList() []string {
	ch := make(chan []string)
	s.nodeMgr.GetNodeList <- ch
	return <-ch
}

// Returns the number of connected nodes.
// Used by the query manager to adjust request intervals.
func (s *Servent) connNodeCnt() int {
	ch := make(chan int)
	s.nodeMgr.getConnNodeCnt <- ch
	return <-ch
}
