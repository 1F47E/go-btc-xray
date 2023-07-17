package client

import (
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/storage"
	"path/filepath"
	"runtime"
	"time"
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
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if len(c.nodesNew) == 0 {
				continue
			}
			// feed the first node from the new nodes
			n := c.nodesNew[0]
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
			err := n.Connect(c.ctx, c.nodeResCh)
			if err != nil {
				c.nodesDeadCnt++
			}
		}
	}
}

// Get stats of all the nodes, filter good ones, save them.
func (c *Client) wGuiUpdater() {
	c.log.Debug("[CLIENT]: STAT: worker started")
	defer c.log.Debug("[CLIENT]: STAT: worker exited")

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:

			// send new data to gui
			c.guiCh <- gui.IncomingData{
				Connections: len(c.queueCh),
				NodesTotal:  len(c.nodes),
				NodesQueued: len(c.nodesNew),
				NodesGood:   len(c.nodesGood),
				NodesDead:   c.nodesDeadCnt,
			}
			c.log.Debugf("[CLIENT]: STAT: total:%d, connected:%d/%d, good:%d, dead:%d", len(c.nodes), len(c.queueCh), cfg.ConnectionsLimit, len(c.nodesGood), c.nodesDeadCnt)
			// report G count and memory used
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			c.log.Debugf("[CLIENT]: STAT: G:%d, MEM:%dKb\n", runtime.NumGoroutine(), m.Alloc/1024)

			// save good to json file
			path := filepath.Join(cfg.DataDir, cfg.NodesFilename)
			if len(c.nodesGood) > 0 {
				err := storage.Save(path, c.nodesGood)
				if err != nil {
					c.log.Errorf("[CLIENT]: STAT: failed to save nodes: %v\n", err)
					continue
				}

				c.log.Debugf("[CLIENT] STAT: saved %d node to %v\n", len(c.nodesGood), path)
			}
		}
	}
}
