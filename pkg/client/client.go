package client

import (
	"context"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/logger"
	"go-btc-downloader/pkg/node"
	"go-btc-downloader/pkg/storage"
	"runtime"
	"sync"
	"time"
)

var mu = sync.Mutex{}

var cfg = config.New()

// batch of new addresses. not block the sented (listner) goroutine
var newAddrCh = make(chan []string, 100)

type Client struct {
	ctx   context.Context
	exit  context.CancelFunc
	log   *logger.Logger
	nodes map[string]*node.Node
	guiCh chan gui.IncomingData
}

func NewClient(ctx context.Context, log *logger.Logger, guiCh chan gui.IncomingData) *Client {
	// client context to stop the client but not the app
	cliCtx, cancel := context.WithCancel(ctx)
	c := Client{
		ctx:   cliCtx,
		exit:  cancel,
		log:   log,
		nodes: make(map[string]*node.Node),
		guiCh: guiCh,
	}
	return &c
}

func (c *Client) AddNodes(ips []string) {
	c.log.Debugf("[CLIENT]: got batch of %d nodes\n", len(ips))
	cnt := 1
	for _, ip := range ips {
		if _, ok := c.nodes[ip]; ok {
			continue
		}
		mu.Lock()
		c.nodes[ip] = node.NewNode(c.log, ip, newAddrCh)
		mu.Unlock()
		cnt++
	}
	c.log.Debugf("[CLIENT]: got %d nodes from %d batch\n", cnt, len(ips))
}

func (c *Client) Nodes() map[string]*node.Node {
	return c.nodes
}

func (c *Client) NodesQueue() []*node.Node {
	nodes := make([]*node.Node, 0)
	mu.Lock()
	for _, n := range c.nodes {
		if n.IsNew() {
			nodes = append(nodes, n)
		}
	}
	mu.Unlock()
	return nodes
}

func (c *Client) NodesGood() []*node.Node {
	nodes := make([]*node.Node, 0)
	mu.Lock()
	for _, n := range c.nodes {
		if n.IsGood() {
			nodes = append(nodes, n)
		}
	}
	mu.Unlock()
	return nodes
}

func (c *Client) NodesGoodCnt() int {
	mu.Lock()
	cnt := 0
	for _, n := range c.nodes {
		if n.IsGood() {
			cnt++
		}
	}
	mu.Unlock()
	return cnt
}

func (c *Client) NodesDeadCnt() int {
	mu.Lock()
	cnt := 0
	for _, n := range c.nodes {
		if n.IsDead() {
			cnt++
		}
	}
	mu.Unlock()
	return cnt
}

func (c *Client) GetFromQueue(num int) []*node.Node {
	ret := make([]*node.Node, 0)
	queue := c.NodesQueue()
	if len(queue) == 0 {
		return ret
	}
	for i, n := range queue {
		if len(ret) == num || i == len(queue)-1 {
			break
		}
		ret = append(ret, n)
	}
	return ret
}

func (c *Client) NodesCnt() int {
	return len(c.nodes)
}

// count busy nodes - connecting and connected status
func (c *Client) NodesBusyCnt() int {
	cnt := 0
	mu.Lock()
	for _, node := range c.nodes {
		if node.IsConnected() || node.IsConnecting() {
			cnt++
		}
	}
	mu.Unlock()
	return cnt
}

func (c *Client) Start() {
	go c.wNodesConnector() // connect to the nodes with a queue
	go c.wNodesListner()   // read channel with new nodes
	go c.wNodesUpdater()   // save nodes info
}

func (c *Client) Stop() {
	c.log.Debug("[CLIENT]: disconnecting...")
	defer c.log.Debug("[CLIENT]: exited")
	cnt := 0
	for _, n := range c.nodes {
		if n.Disconnect() {
			cnt++
		}
	}
	c.log.Debugf("[CLIENT]: disconnected %d nodes\n", cnt)
}

// ===== WORKERS

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
				go n.Connect(c.ctx)
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
			deadCnt := c.NodesDeadCnt()
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
