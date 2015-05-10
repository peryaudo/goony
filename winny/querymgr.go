package winny

import (
	"log"
	"time"
)

type queryReq struct {
	Keyword string
	Result  chan *QueryResult
}

type QueryResult struct {
}

type queryMgr struct {
	servent *Servent

	AddQuery chan *queryReq

	RecvQuery chan *recvCmd // recvCmd.cmd.(type) == *cmdQuery
}

func newQueryMgr(s *Servent) *queryMgr {
	return &queryMgr{
		servent:   s,
		AddQuery:  make(chan *queryReq),
		RecvQuery: make(chan *recvCmd)}
}

func (m *queryMgr) ListenAndServe() {
	// TODO(peryaudo): set proper timers and do query management
	// TODO(peryaudo): force key TTL 1500 secs

	spreadTick := time.Tick(30 * time.Second)

	for {
		select {
		case <-spreadTick:
			m.servent.nodeMgr.SendCmd <- &sendCmd{Direction: DirectionAll, cmd: &cmdSpread{}}
		case recvCmd := <-m.RecvQuery:
			query := recvCmd.cmd.(*cmdQuery)

			// DBG
			if len(query.Keyword) > 0 {
				log.Printf("Search: %s\n", query.Keyword)
				// log.Printf("Search: %d\n", len(query.Keyword))
			}
			for _, key := range query.Keys {
				log.Printf("File: %s\n", key.FileName)
				// log.Printf("File: %d\n", len(key.FileName))
			}
			// DBG

			for _, addr := range query.Nodes {
				m.servent.nodeMgr.AddNodeAddr <- addr
			}
			for _, key := range query.Keys {
				m.servent.nodeMgr.AddNodeAddr <- key.Node
			}

		case queryReq := <-m.AddQuery:
			// TODO(peryaudo): add to query request list
			queryReq = queryReq
		}
	}
}
