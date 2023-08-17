package client

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/1F47E/go-btc-xray/internal/gui"
	"github.com/1F47E/go-btc-xray/internal/storage"
)

// listen for new nodes from the connected nodes
func (c *Client) wNewAddrListner() {
	c.log.Debug("[CLIENT]: LISTENER worker started")
	defer c.log.Debug("[CLIENT]: LISTNER worker exited")

	for {
		select {
		case <-c.ctx.Done():
			return
		case ips := <-c.newAddrCh:
			c.AddNodes(ips)
		}
	}
}

// feed the queue with new nodes
func (c *Client) wNodesFeeder() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if len(c.nodesNew) == 0 {
				// do not overload the cpu by spinning to fast
				time.Sleep(time.Millisecond * 100)
				continue
			}
			n := c.nodesNew[0]
			// feed the first node from the new nodes
			// pop it from the new slice for garbage collection
			// will block if queue is full
			c.nodesNew = c.nodesNew[1:]
			c.queueCh <- n
		}
	}
}

// get errors from the nodes connections
func (c *Client) wNodeResultsHandler() {
	c.log.Debug("[CLIENT]: ERRORS worker started")
	defer c.log.Debug("[CLIENT]: ERRORS worker exited")
	for {
		select {
		case <-c.ctx.Done():
			return
		case n := <-c.nodeResCh:
			c.nodesGood = append(c.nodesGood, n)
		}
	}
}

// save good nodes to a file
func (c *Client) wNodeSaver() {
	c.log.Debug("[CLIENT]: SAVER worker started")
	ticker := time.NewTicker(time.Second * 1)
	cnt := 0
	defer func() {
		c.log.Debug("[CLIENT]: SAVER worker exited")
		ticker.Stop()
	}()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if len(c.nodesGood) == cnt {
				continue
			}
			// save good nodes to a file
			err := storage.Save(c.nodesGood)
			if err != nil {
				c.log.Errorf("[CLIENT]: STAT: failed to save nodes: %v\n", err)
				continue
			}
			c.log.Infof("[CLIENT]: saved %d nodes", len(c.nodesGood))
			cnt = len(c.nodesGood)
		}
	}
}

// Connect to the nodes with a limit of connection
// Number of workers = connections limit
func (c *Client) wNodesConnector(n int) {
	c.log.Debugf("[CLIENT]: CONN_%d worker started", n)
	defer func() {
		c.log.Debugf("[CLIENT]: CONN_%d worker exited", n)
	}()
	for {
		select {
		case <-c.ctx.Done():
			return
		case n := <-c.queueCh:
			atomic.AddInt32(&c.activeConns, 1)
			err := n.Connect(c.ctx, c.nodeResCh)
			if err != nil {
				atomic.AddInt32(&c.nodesDeadCnt, 1)

			}
			atomic.AddInt32(&c.activeConns, -1)
		}
	}
}

// Get stats of all the nodes, filter good ones, save them.
func (c *Client) wGuiUpdater() {
	c.log.Debug("[CLIENT]: STAT: worker started")
	defer c.log.Debug("[CLIENT]: STAT: worker exited")

	// gui data update rate
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:

			// send new data to gui
			connCnt := c.ActiveConns()
			deadCnt := atomic.LoadInt32(&c.nodesDeadCnt)
			c.guiCh <- gui.IncomingData{
				Connections: connCnt,
				NodesTotal:  len(c.nodes),
				NodesQueued: len(c.nodesNew),
				NodesGood:   len(c.nodesGood),
				NodesDead:   deadCnt,
			}
			c.log.Debugf("[CLIENT]: STAT: total:%d, connected:%d/%d, good:%d, dead:%d", len(c.nodes), connCnt, cfg.ConnectionsLimit, len(c.nodesGood), c.nodesDeadCnt)

			// report G count and memory used
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			c.log.Debugf("[CLIENT]: STAT: G:%d, MEM:%dKb\n", runtime.NumGoroutine(), m.Alloc/1024)

		}
	}
}
