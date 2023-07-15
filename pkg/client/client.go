package client

import (
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
	log   *logger.Logger
	nodes map[string]*node.Node
	guiCh chan gui.Data
}

// initial nodes from DNS
func NewClient(log *logger.Logger, guiCh chan gui.Data) *Client {
	c := Client{
		log:   log,
		nodes: make(map[string]*node.Node),
		guiCh: guiCh,
	}
	return &c
}

func (c *Client) AddNodes(ips []string) {
	c.log.Debugf("[CLIENT]: adding %d nodes\n", len(ips))
	nodes := make([]*node.Node, 0)
	for _, ip := range ips {
		n := node.NewNode(ip, newAddrCh)
		nodes = append(nodes, n)
	}
	for _, n := range nodes {
		c.nodes[n.Endpoint()] = n
	}
}

func (c *Client) Nodes() map[string]*node.Node {
	return c.nodes
}

func (c *Client) NodesCnt() int {
	return len(c.nodes)
}

// count connected nodes
func (c *Client) ConnectedNodesCnt() int {
	cnt := 0
	mu.Lock()
	for _, node := range c.nodes {
		if node.IsConnected() {
			cnt++
		}
	}
	mu.Unlock()
	return cnt
}

// ===== WORKERS

func (c *Client) Start() {
	go c.nodesListner() // read channel
	go c.nodesConnector()
	go c.nodesUpdater()
}

// listen for new nodes from the connected nodes
func (c *Client) nodesListner() {
	c.log.Debug("[CLIENT]: LISTEN: started")
	defer c.log.Debug("[CLIENT]: LISTEN: exited")
	for ips := range newAddrCh {
		c.log.Debugf("[CLIENT]: LISTEN: got batch %d new nodes\n", len(ips))
		mu.Lock()
		for _, ip := range ips {
			if _, ok := c.nodes[ip]; ok {
				continue
			}
			c.nodes[ip] = node.NewNode(ip, newAddrCh)
		}
		mu.Unlock()
	}
}

// Connect to the nodes with a limit of connection
func (c *Client) nodesConnector() {
	c.log.Debug("[CLIENT]: CONN: nodesConnector started")
	defer c.log.Debug("[CLIENT]: CONN: nodesConnector exited")
	limit := cfg.ConnectionsLimit // connection pool
	for {
		time.Sleep(1 * time.Second)
		connectedCnt := c.ConnectedNodesCnt()
		left := limit - connectedCnt
		// connect only if have slot
		if left <= 0 {
			continue
		}
		// select node to connect to
		waitlist := make([]*node.Node, 0)
		mu.Lock()
		for _, node := range c.nodes {
			if node.IsNew() {
				waitlist = append(waitlist, node)
			}
		}
		mu.Unlock()
		c.log.Infof("[CLIENT]: CONN: %d/%d nodes connected\n", connectedCnt, limit)
		c.log.Infof("[CLIENT]: CONN: %d nodes in waitlist\n", len(waitlist))
		c.log.Infof("[CLIENT]: CONN: connecting to %d nodes\n", left)
		if left > 0 && len(waitlist) > 0 {
			for i := 0; i <= left; i++ {
				if i >= len(waitlist) {
					break
				}
				go waitlist[i].Connect()
			}
		}
		// exit if done
		// if len(waitlist) == 0 {
		// 	c.log.Warn("[CLIENT]: CONN: no nodes to connect, exit")
		// 	for _, node := range c.nodes {
		// 		node.Disconnect()
		// 	}
		// 	os.Exit(0)
		// }
	}
}

// Get stats of all the nodes, filter good ones, save them.
func (c *Client) nodesUpdater() {
	c.log.Debug("[CLIENT]: STAT: nodesUpdater started")
	defer c.log.Debug("[CLIENT]: STAT: nodesUpdater exited")
	for {
		time.Sleep(1 * time.Second)
		if len(c.nodes) == 0 {
			c.log.Warn("[CLIENT] STAT no nodes, exit")
			continue
		}

		// filter good nodes
		good := make([]*node.Node, 0)
		var dead, connected, connections, queued int
		mu.Lock()
		for _, node := range c.nodes {
			if node.IsNew() {
				queued++
			}
			if node.IsGood() {
				good = append(good, node)
			}
			if node.IsDead() {
				dead++
			}
			if node.IsConnected() {
				connected++
			}
			if node.HasConnection() {
				connections++
			}
		}
		mu.Unlock()
		// updat edata to gui
		c.guiCh <- gui.Data{
			Connections: connections,
			NodesTotal:  len(c.nodes),
			NodesQueued: queued,
			NodesGood:   len(good),
			NodesDead:   dead,
		}
		c.log.Infof("[CLIENT]: STAT: total:%d, connected:%d(%d conn), dead:%d, good:%d\n", len(c.nodes), connected, connections, dead, len(good))
		// report G count and memory used
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		c.log.Debugf("[CLIENT]: STAT: G:%d, MEM:%dKb\n", runtime.NumGoroutine(), m.Alloc/1024)

		// save good to json file
		if len(good) > 0 {
			err := storage.Save(cfg.GoodNodesDB, good)
			if err != nil {
				c.log.Debugf("[CLIENT]: STAT: failed to save nodes: %v\n", err)
				continue
			}

			c.log.Infof("[CLIENT] STAT: saved %d node to %v\n", len(good), cfg.GoodNodesDB)
		}
	}
}
