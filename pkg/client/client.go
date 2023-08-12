// Client managing and connecting to a list of Bitcoin nodes.
// It maintains a queue of nodes to be connected to and uses a worker pool to establish connections.
// Each worker runs in a separate goroutine and can connect to one node at a time.
// When a connection is established (or fails), the result is sent to a results handler.
// New nodes can be added to the client at any time from another nodes.
// They added to the nodes map for quick check for duplicates
// New nodes also added to the new nodes slices and then feeded to the queue to connect.
package client

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/1F47E/go-btc-xray/pkg/client/node"
	"github.com/1F47E/go-btc-xray/pkg/config"
	"github.com/1F47E/go-btc-xray/pkg/gui"
	"github.com/1F47E/go-btc-xray/pkg/logger"
)

var cfg = config.New()

type Client struct {
	mu   sync.Mutex
	ctx  context.Context
	exit context.CancelFunc
	log  *logger.Logger

	// nodes storage
	nodes     map[string]*node.Node
	nodesNew  []*node.Node
	nodesGood []*node.Node

	// atomic counters
	nodesDeadCnt int32
	activeConns  int32

	// channels
	queueCh   chan *node.Node
	guiCh     chan gui.IncomingData
	nodeResCh chan *node.Node
	newAddrCh chan []string
}

func NewClient(ctx context.Context, log *logger.Logger, guiCh chan gui.IncomingData) *Client {
	// client context to stop the client but not the gui
	// TODO: exit if no gui
	cliCtx, cancel := context.WithCancel(ctx)
	c := Client{
		mu:  sync.Mutex{},
		ctx: cliCtx,

		// called when the is no new nodes anymore to stop all the client workers
		exit: cancel,
		log:  log,

		// keeping all the nodes in a map for quick check for duplicates
		nodes: make(map[string]*node.Node),

		// all new nodes also added to the list
		// then feeder will put them to the queue
		nodesNew: make([]*node.Node, 0, 1000),

		// node considered good after successful connection and handshake
		nodesGood: make([]*node.Node, 0),

		// feeder will put new nodes to the queue
		queueCh: make(chan *node.Node, cfg.ConnectionsLimit),

		// results from the successfull node connection and handshake
		nodeResCh: make(chan *node.Node),

		// used to send updates to the gui
		guiCh: guiCh,

		// connected nodes will send batch of addresses, usually 1000
		// then they will be proccessed by the worker wNewAddrListner
		newAddrCh: make(chan []string, cfg.ConnectionsLimit),
	}
	return &c
}

func (c *Client) Start() {
	// collect and send data to the gui via channel
	go c.wGuiUpdater()

	// proccess good nodes that comes from the connector workers
	go c.wNodeResultsHandler()

	// save good nodes to a file periodically
	go c.wNodeSaver()

	// read channel with new nodes
	go c.wNewAddrListner()

	// feed the queue with new nodes
	go c.wNodesFeeder()

	// start a worker pool to connect to the nodes
	for i := 0; i < cfg.ConnectionsLimit; i++ {
		i := i
		go c.wNodesConnector(i)
	}
}

// TODO: refactor this to know what nodes are now connected
func (c *Client) Disconnect() {
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
	c.mu.Lock()
	for _, ip := range ips {
		if _, ok := c.nodes[ip]; ok {
			continue
		}
		n := node.NewNode(c.log, ip, c.newAddrCh)
		// add new nodes to the all nodes map but also to the queue
		c.nodes[ip] = n
		c.nodesNew = append(c.nodesNew, n)
		cnt++
	}
	// shuffle new nodes
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd.Shuffle(len(c.nodesNew), func(i, j int) {
		c.nodesNew[i], c.nodesNew[j] = c.nodesNew[j], c.nodesNew[i]
	})
	c.mu.Unlock()
	c.log.Debugf("[CLIENT]: got %d nodes from %d batch\n", cnt, len(ips))
}

func (c *Client) ActiveConns() int {
	return int(atomic.LoadInt32(&c.activeConns))
}
