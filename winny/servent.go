package winny

import (
	"errors"
	"log"
)

type Servent struct {
	Speed    int
	Port     int
	Ddns     string
	Clusters [3]string

	recvCmd  chan *recvCmd
	nodeMgr  *nodeMgr
	queryMgr *queryMgr
}

func (s *Servent) ListenAndServe() (err error) {
	s.init()

	if s.Speed == 0 {
		err = errors.New("you must specify the node speed")
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

func (s *Servent) AddNode(node string) {
	s.init()

	go func() {
		s.nodeMgr.AddNodeStr <- node
	}()
}

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

func (s *Servent) KeyStream() <-chan *QueryResult {
	panic("unimplemented")
	return nil
}

func (s *Servent) NodeList() []string {
	ch := make(chan []string)
	s.nodeMgr.GetNodeList <- ch
	return <-ch
}
