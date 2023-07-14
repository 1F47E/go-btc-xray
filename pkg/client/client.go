package client

import (
	"go-btc-downloader/pkg/logger"
	"go-btc-downloader/pkg/node"
	"net"
	"sync"
	"time"
)

var mu = sync.Mutex{}
var log *logger.Logger = logger.New()

// batch of new addresses. not block the sented (listner) goroutine
var newAddrCh = make(chan []string, 100)

type Client struct {
	nodes map[string]*node.Node
}

// initial nodes from DNS
func NewClient(addrs []string) *Client {
	nodes := make([]*node.Node, 0)
	for _, addr := range addrs {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			log.Debugf("failed to resolve addr %s: %v\n", addr, err)
			continue
		}
		n := node.NewNode(*tcpAddr, newAddrCh)
		nodes = append(nodes, n)
	}

	c := Client{
		nodes: make(map[string]*node.Node),
	}
	for _, n := range nodes {
		c.nodes[n.Endpoint()] = n
	}
	return &c
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
	go c.nodesListner()
	go c.nodesConnector()
	go c.nodesUpdater()
}

// listen for new nodes from the connected nodes
func (c *Client) nodesListner() {
	log.Debug("[NODES]: newNodesListner started")
	defer log.Debug("[NODES]: newNodesListner exited")
	for addrs := range newAddrCh {
		log.Debugf("[NODES]: got batch %d new nodes\n", len(addrs))
		mu.Lock()
		for _, addr := range addrs {
			// TODO: check for ip key first before resolving
			tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				log.Debugf("[NODES]: failed to resolve addr %s: %v\n", addr, err)
				continue
			}
			n := node.NewNode(*tcpAddr, newAddrCh)
			// add only if new
			addr := n.EndpointSafe()
			if _, ok := c.nodes[addr]; !ok {
				c.nodes[addr] = n
			}
		}
		mu.Unlock()
	}
}

// Connect to the nodes with a limit of connection
func (c *Client) nodesConnector() {
	log.Debug("[NODES CONNECTOR]: nodesConnector started")
	defer log.Debug("[NODES CONNECTOR]: nodesConnector exited")
	limit := 5 // connection pool
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
		log.Infof("[NODES CONNECTOR]: %d/%d nodes connected\n", connectedCnt, limit)
		log.Infof("[NODES CONNECTOR]: %d nodes in waitlist\n", len(waitlist))
		log.Infof("[NODES CONNECTOR]: connecting to %d nodes\n", left)
		if left > 0 && len(waitlist) > 0 {
			for i := 0; i <= left; i++ {
				if i >= len(waitlist) {
					break
				}
				go waitlist[i].Connect()
			}
		}
	}
}

// Get stats of all the nodes, filter good ones, save them.
func (c *Client) nodesUpdater() {
	log.Debug("[NODES STAT]: nodesUpdater started")
	defer log.Debug("[NODES STAT]: nodesUpdater exited")
	for {
		time.Sleep(1 * time.Second)
		if len(c.nodes) == 0 {
			log.Warn("[NODES STAT]: no nodes, exit")
			return
		}

		// filter good nodes
		good := make([]string, 0)
		dead := 0
		connected := 0
		connections := 0
		mu.Lock()
		for addr, node := range c.nodes {
			if node.IsHandshaked() {
				good = append(good, addr)
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
		log.Infof("[NODES STAT]: total:%d, connected:%d(%d conn), dead:%d, good:%d\n", len(c.nodes), connected, connections, dead, len(good))
	}
}

// save to json file
// j, err := json.MarshalIndent(good, "", "  ")
// if err != nil {
// 	log.Debugf("[NODES]: failed to marshal nodes: %v\n", err)
// 	continue
// }
// err = os.WriteFile(cfg.NodesDB, j, 0644)
// if err != nil {
// 	log.Debugf("[NODES]: failed to write nodes: %v\n", err)
// 	continue
// }
// log.Debugf("[NODES]: saved %d node to %v\n", len(good), cfg.PeersDB)

// func SeedScan() ([]*node.Node, error) {
// 	nodes := make([]*node.Node, 0)
// 	log.Debug("Getting nodes from dns seeds... via ", cfg.DnsAddress)
// 	now := time.Now()
// 	if cfg.DnsSeeds == nil {
// 		return nil, fmt.Errorf("no dns seeds")
// 	}
// 	for _, seed := range cfg.DnsSeeds {
// 		m := new(dns.Msg)
// 		m.SetQuestion(dns.Fqdn(seed), dns.TypeA)
// 		c := new(dns.Client)
// 		c.Net = "tcp"
// 		c.Timeout = cfg.DnsTimeout
// 		in, _, err := c.Exchange(m, cfg.DnsAddress)
// 		if err != nil {
// 			log.Errorf("Failed to get nodes from %v: %v\n", seed, err)
// 			continue
// 		}
// 		if len(in.Answer) == 0 {
// 			log.Errorf("No nodes found from %v\n", seed)
// 		} else {
// 			log.Infof("Got %v nodes from %v\n", len(in.Answer), seed)
// 		}
// 		// loop through dns records
// 		for _, ans := range in.Answer {
// 			// check that record is valid
// 			if _, ok := ans.(*dns.A); !ok {
// 				continue
// 			}
// 			record := ans.(*dns.A)
// 			// check if already exists
// 			for _, node := range nodes {
// 				if node.IP().Equal(record.A) {
// 					continue
// 				}
// 			}
// 			a := net.TCPAddr{IP: record.A, Port: int(cfg.NodesPort)}
// 			n := node.NewNode(a, newAddrCh)
// 			nodes = append(nodes, n)
// 		}
// 	}
// 	if len(nodes) == 0 {
// 		return nil, fmt.Errorf("No nodes found")
// 	}
// 	log.Infof("Got %v nodes in %v\n", len(nodes), time.Since(now))

// 	// save nodes as json
// 	fData := make([]string, len(nodes))
// 	for i, n := range nodes {
// 		fData[i] = n.EndpointSafe() // [addr]:port for ipv6
// 	}
// 	fDataJson, err := json.MarshalIndent(fData, "", "  ")
// 	if err != nil {
// 		log.Fatalf("failed to marshal nodes: %v", err)
// 	}
// 	err = os.WriteFile(cfg.NodesDB, fDataJson, 0644)
// 	if err != nil {
// 		log.Fatalf("failed to write nodes: %v", err)
// 	}
// 	log.Infof("saved %v nodes to %v\n", len(nodes), cfg.NodesDB)
// 	return nodes, nil
// }
