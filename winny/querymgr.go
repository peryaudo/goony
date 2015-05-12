package winny

import (
	"log"
	"time"
)

// queryMgr manages searching queries.
type queryMgr struct {
	servent *Servent

	AddQuery    chan *queryReq
	RemoveQuery chan chan *FileKey

	AddKeywordStream    chan chan string
	RemoveKeywordStream chan chan string

	RecvQuery chan *recvCmd // recvCmd.cmd.(type) == *cmdQuery

	keys         map[[16]byte]*FileKey
	queryChans   map[chan *FileKey]string
	keywordChans map[chan string]struct{}
}

func newQueryMgr(s *Servent) *queryMgr {
	return &queryMgr{
		servent:             s,
		AddQuery:            make(chan *queryReq),
		RemoveQuery:         make(chan chan *FileKey),
		AddKeywordStream:    make(chan chan string),
		RemoveKeywordStream: make(chan chan string),
		RecvQuery:           make(chan *recvCmd),
		keys:                make(map[[16]byte]*FileKey),
		queryChans:          make(map[chan *FileKey]string),
		keywordChans:        make(map[chan string]struct{})}
}

func (m *queryMgr) ListenAndServe() {
	// TODO(peryaudo): set proper timers and do query management
	// TODO(peryaudo): force key TTL 1500 secs

	spreadTick := time.Tick(30 * time.Second)

	for {
		select {
		case <-spreadTick:
			m.servent.nodeMgr.SendCmd <- &sendCmd{
				Direction: DirectionAll,
				cmd:       &cmdSpread{}}
			log.Printf("total keys: %d\n", len(m.keys))

		case q := <-m.AddQuery:
			m.queryChans[q.Results] = q.Keyword

		case ch := <-m.RemoveQuery:
			delete(m.queryChans, ch)

		case k := <-m.AddKeywordStream:
			m.keywordChans[k] = struct{}{}

		case ch := <-m.RemoveKeywordStream:
			delete(m.keywordChans, ch)

		case recvCmd := <-m.RecvQuery:
			m.dispatchQuery(recvCmd.cmd.(*cmdQuery))
		}
	}
}

func (m *queryMgr) dispatchQuery(query *cmdQuery) {
	// Dispatch to search result channels
	// TODO(peryaudo): conditionally dispatch to queryChans
	for _, key := range query.Keys {
		if m.keys[key.Hash] != nil {
			continue
		}

		for ch, _ := range m.queryChans {
			ch <- &key
		}
	}

	// Dispatch to keyword stream channels
	if len(query.Keyword) > 0 {
		for ch, _ := range m.keywordChans {
			ch <- query.Keyword
		}
	}

	// Save file keys
	for _, key := range query.Keys {
		m.keys[key.Hash] = &key
	}

	// Add the addrs in the query to the node list
	for _, addr := range query.Nodes {
		m.servent.nodeMgr.AddNodeAddr <- addr
	}
	for _, key := range query.Keys {
		m.servent.nodeMgr.AddNodeAddr <- key.Node
	}
}
