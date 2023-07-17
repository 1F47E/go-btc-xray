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
var murw = sync.RWMutex{}

var cfg = config.New()

// batch of new addresses. not block the sented (listner) goroutine
var newAddrCh = make(chan []string, 100)

type Client struct {
	ctx  context.Context
	exit context.CancelFunc
	log  *logger.Logger

	nodes        map[string]*node.Node
	nodesGood    []*node.Node
	nodesDeadCnt int
	// send data to the gui
	guiCh chan gui.IncomingData
	// get data from the connected node
	// nodeErrCh chan error
	nodeResCh chan *node.Node
}

func NewClient(ctx context.Context, log *logger.Logger, guiCh chan gui.IncomingData) *Client {
	// client context to stop the client but not the gui
	// TODO: exit if no gui
	cliCtx, cancel := context.WithCancel(ctx)
	c := Client{
		ctx:       cliCtx,
		exit:      cancel,
		log:       log,
		nodes:     make(map[string]*node.Node),
		nodesGood: make([]*node.Node, 0),
		// nodeErrCh: make(chan error),
		nodeResCh: make(chan *node.Node),
		guiCh:     guiCh,
	}
	return &c
}

func (c *Client) Start() {
	go c.wNodesConnector() // connect to the nodes with a queue
	go c.wNewAddrListner() // read channel with new nodes
	go c.wGuiUpdater()     // save nodes info
	go c.wNodeResultsHandler()
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

// TODO: refactor this to a proper queue
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

// count busy nodes - connecting and connected status
func (c *Client) NodesBusyCnt() int {
	cnt := 0
	murw.RLock()
	for _, node := range c.nodes {
		if node.IsConnected() || node.IsConnecting() {
			cnt++
		}
	}
	murw.RUnlock()
	return cnt
}
