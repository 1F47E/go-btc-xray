package client

import (
	"context"
	"go-btc-downloader/pkg/client/node"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/logger"
	"sync"
)

var mu = sync.Mutex{}

var cfg = config.New()

// batch of new addresses. not block the sented (listner) goroutine
var newAddrCh = make(chan []string, 100)

type Client struct {
	ctx     context.Context
	exit    context.CancelFunc
	log     *logger.Logger
	nodes   map[string]*node.Node
	guiCh   chan gui.IncomingData
	errCh   chan error
	CntDead int
}

func NewClient(ctx context.Context, log *logger.Logger, guiCh chan gui.IncomingData) *Client {
	// client context to stop the client but not the app
	cliCtx, cancel := context.WithCancel(ctx)
	c := Client{
		ctx:   cliCtx,
		exit:  cancel,
		log:   log,
		nodes: make(map[string]*node.Node),
		errCh: make(chan error),
		guiCh: guiCh,
	}
	return &c
}

func (c *Client) Start() {
	go c.wNodesConnector() // connect to the nodes with a queue
	go c.wNodesListner()   // read channel with new nodes
	go c.wNodesUpdater()   // save nodes info
	go c.wErrorsHandler()
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

// func (c *Client) NodesByStatus(status node.Status) []*node.Node {
// 	nodes := make([]*node.Node, 0)
// 	mu.Lock()
// 	for _, n := range c.nodes {
// 		if n.Status() == status {
// 			nodes = append(nodes, n)
// 		}
// 	}
// 	mu.Unlock()
// 	return nodes
// }

// func (c *Client) NodesCountByStatus(status node.Status) int {
// 	mu.Lock()
// 	cnt := 0
// 	for _, n := range c.nodes {
// 		if n.Status() == status {
// 			cnt++
// 		}
// 	}
// 	mu.Unlock()
// 	return cnt
// }

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
