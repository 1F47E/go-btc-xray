package client

import (
	"go-btc-downloader/pkg/client/node"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/storage"
	"runtime"
	"time"
)

// get errors from the nodes connections
func (c *Client) wErrorsHandler() {
	c.log.Debug("[CLIENT]: ERRORS worker started")
	defer c.log.Debug("[CLIENT]: ERRORS worker exited")
	for {
		select {
		case <-c.ctx.Done():
			return
		case err := <-c.errCh:
			if err != nil {
				c.CntDead++
				c.log.Errorf("[CLIENT]: %s\n", err)
			}
		}
	}
}

// listen for new nodes from the connected nodes
func (c *Client) wNodesListner() {
	c.log.Debug("[CLIENT]: LISTENER worker started")
	defer c.log.Debug("[CLIENT]: LISTNER worker exited")

	for {
		select {
		case <-c.ctx.Done():
			return
		case ips := <-newAddrCh:
			c.AddNodes(ips)
		}
	}
}

// Connect to the nodes with a limit of connection
func (c *Client) wNodesConnector() {
	c.log.Debug("[CLIENT]: CONN worker started")
	defer c.log.Debug("[CLIENT]: CONN worker exited")

	// check if we have enough connections and connect to the nodes
	limit := cfg.ConnectionsLimit // connection pool
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// get connecting and connected nodes
			connectedCnt := c.NodesBusyCnt()
			left := limit - connectedCnt
			// connect only if have slot
			if left <= 0 {
				continue
			}
			c.log.Infof("[CLIENT]: CONN: total nodes %d", len(c.nodes))
			c.log.Infof("[CLIENT]: CONN: %d/%d nodes connected", connectedCnt, limit)
			// get nodes to connect to
			pool := c.GetFromQueue(left)

			// exit client logic
			// idea is to stop the client and close all the connections but keep the gui running
			// if len(c.nodes) > 50 { // BUG: debug, force exit
			if len(pool) == 0 && connectedCnt == 0 {
				c.log.Warn("[CLIENT]: CONN: no nodes to connect, exit")
				c.exit()
			}
			if len(pool) == 0 {
				c.log.Debugf("[CLIENT]: CONN: no nodes to connect")
				continue
			}
			c.log.Infof("[CLIENT]: CONN: connecting to %d nodes", len(pool))
			for _, n := range pool {
				go func(n *node.Node) {
					err := n.Connect(c.ctx)
					if err != nil {
						c.errCh <- err
					}
				}(n)
			}
		}
	}
}

// Get stats of all the nodes, filter good ones, save them.
func (c *Client) wNodesUpdater() {
	c.log.Debug("[CLIENT]: STAT: worker started")
	defer c.log.Debug("[CLIENT]: STAT: worker exited")

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:

			queue := c.NodesQueue()
			good := c.NodesGood()
			deadCnt := c.CntDead
			cntBusy := c.NodesBusyCnt() // busy nodes - connecting and connected status

			// send new data to gui
			c.guiCh <- gui.IncomingData{
				Connections: cntBusy,
				NodesTotal:  len(c.nodes),
				NodesQueued: len(queue),
				NodesGood:   len(good),
				NodesDead:   deadCnt,
			}
			c.log.Debugf("[CLIENT]: STAT: total:%d, connected:%d/%d, dead:%d, good:%d\n", len(c.nodes), cntBusy, cfg.ConnectionsLimit, deadCnt, len(good))
			// report G count and memory used
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			c.log.Debugf("[CLIENT]: STAT: G:%d, MEM:%dKb\n", runtime.NumGoroutine(), m.Alloc/1024)

			// save good to json file
			if len(good) > 0 {
				err := storage.Save(cfg.GoodNodesDB, good)
				if err != nil {
					c.log.Errorf("[CLIENT]: STAT: failed to save nodes: %v\n", err)
					continue
				}

				c.log.Debugf("[CLIENT] STAT: saved %d node to %v\n", len(good), cfg.GoodNodesDB)
			}
		}
	}
}
