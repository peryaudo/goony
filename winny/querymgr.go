package winny

import (
	"log"
	"math/rand"
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

	keys map[[16]byte]*FileKey

	queries            map[chan *FileKey]string
	keywordStreamChans map[chan string]struct{}

	queryIdCnt uint32
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
		queries:             make(map[chan *FileKey]string),
		keywordStreamChans:  make(map[chan string]struct{})}
}

func (m *queryMgr) ListenAndServe() {
	// TODO(peryaudo): set proper timers and do query management
	// TODO(peryaudo): force key TTL 1500 secs
	// TODO(peryaudo): send query at regular interval
	// TODO(peryaudo): process received search query

	spreadTimeout := time.After(30 * time.Second)
	searchTimeout := time.After(10 * time.Second)

	for {
		select {
		case recvCmd := <-m.RecvQuery:
			m.dispatchQuery(recvCmd.cmd.(*cmdQuery))

		case <-spreadTimeout:
			// SendCmd sends the command to a random single node every time.
			m.servent.nodeMgr.SendCmd <- &sendCmd{
				Direction: directionAll,
				cmd:       &cmdSpread{}}

			// Adjust cmdSpread interval by connected node count.
			// By diving the interval by the node count,
			// we can accomplish the same effect as sending the command
			// to all the nodes simultaneously by the interval.
			interval := 30 * time.Second
			nodeCnt := m.servent.connNodeCnt()
			if nodeCnt > 0 {
				interval /= time.Duration(nodeCnt)
			}

			log.Printf("total keys: %d current interval: %d\n", len(m.keys), interval/time.Second)

			spreadTimeout = time.After(interval)

		case <-searchTimeout:
			// Temporarily disabled because it does not work
			/*
				total := m.pickAndSearch()

				interval := 10 * time.Second
				if total > 0 {
					interval /= time.Duration(total)
				}

				log.Printf("query interval: %d\n", interval)

				searchTimeout = time.After(interval)
			*/

		case q := <-m.AddQuery:
			m.queries[q.Results] = q.Keyword
			for _, key := range m.keys {
				if len(q.Keyword) == 0 || key.Match(q.Keyword) {
					k := *key
					q.Results <- &k
				}
			}

		case ch := <-m.RemoveQuery:
			delete(m.queries, ch)

		case k := <-m.AddKeywordStream:
			m.keywordStreamChans[k] = struct{}{}

		case ch := <-m.RemoveKeywordStream:
			delete(m.keywordStreamChans, ch)
		}
	}
}

func (m *queryMgr) dispatchQuery(query *cmdQuery) {
	if query.Keyword == ".jpg" {
		panic("did it!")
	}

	// Dispatch to search result channels
	for _, key := range query.Keys {
		if m.keys[key.Hash] != nil {
			continue
		}

		for ch, keyword := range m.queries {
			if len(keyword) == 0 || key.Match(keyword) {
				k := key
				ch <- &k
			}
		}
	}

	// Dispatch to keyword stream channels
	if len(query.Keyword) > 0 {
		for ch, _ := range m.keywordStreamChans {
			ch <- query.Keyword
		}
	}

	// Save file keys
	for _, key := range query.Keys {
		k := key
		m.keys[key.Hash] = &k
	}

	// Add the addrs in the query to the node list
	for _, addr := range query.Nodes {
		m.servent.nodeMgr.AddNodeAddr <- addr
	}
	for _, key := range query.Keys {
		m.servent.nodeMgr.AddNodeAddr <- key.Node
	}
}

// pickAndSearch picks a genuine search query and sends cmdQuery for that.
// It returns total number of the genuine queries.
func (m *queryMgr) pickAndSearch() (total int) {
	total = 0
	for _, keyword := range m.queries {
		if len(keyword) > 0 {
			total++
		}
	}

	if total == 0 {
		return
	}

	picked := rand.Intn(total)

	// Counter for genuine queries (empty queries are ignored and not counted)
	cnt := 0
	for _, keyword := range m.queries {
		if len(keyword) == 0 {
			continue
		}

		if cnt == picked {
			m.servent.nodeMgr.SendCmd <- &sendCmd{
				Direction: directionRoughlyUp,
				cmd: &cmdQuery{
					Id:      m.queryIdCnt,
					Keyword: keyword,
					// When node list is empty, nodeConn will add own IP to it.
					Nodes: make([]nodeAddr, 0),
					Keys:  make([]FileKey, 0)}}
			m.queryIdCnt++
			return
		}
		cnt++
	}

	return
}
