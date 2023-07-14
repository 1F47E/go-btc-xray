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

// TODO: run in batches + connect to the new nodes
func (c *Client) AddrListner() {
	for addrs := range newAddrCh {
		log.Debugf("[NODES]: got batch %d new nodes\n", len(addrs))
		mu.Lock()
		for _, addr := range addrs {
			tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				log.Debugf("[NODES]: failed to resolve addr %s: %v\n", addr, err)
				continue
			}
			// n := node.Node{Addr: *tcpAddr}
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

func (c *Client) Connector() {
	limit := 5
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
			if node.IsConnected() {
				continue
			}
			waitlist = append(waitlist, node)
		}
		mu.Unlock()
		log.Infof("[NODES]: %d/%d nodes connected\n", connectedCnt, limit)
		if left > 0 && len(waitlist) > 0 {
			log.Infof("[NODES]: %d nodes in waitlist\n", len(waitlist))
			log.Infof("[NODES]: connecting to %d nodes\n", left)
			for i := 0; i <= left; i++ {
				if i >= len(waitlist) {
					break
				}
				go waitlist[i].Connect()
			}
		}
	}
}

func (c *Client) NodesUpdated() {
	log.Debug("[NODES]: NodesUpdated started")
	defer log.Debug("[NODES]: NodesUpdated exited")
	for {
		time.Sleep(1 * time.Second)
		if len(c.nodes) == 0 {
			log.Fatal("[NODES]: no nodes, exit")
		}
		log.Infof("[NODES]: update nodes. total now %d\n", len(c.nodes))

		// filter good nodes
		good := make([]string, 0)
		mu.Lock()
		toDelete := make([]string, 0)
		for addr, node := range c.nodes {
			if node.IsDead() {
				log.Warnf("[NODES]: node %s is dead\n", addr)
				toDelete = append(toDelete, addr)
				continue
			}
			good = append(good, addr)
		}
		for _, addr := range toDelete {
			delete(c.nodes, addr)
		}
		mu.Unlock()
		perc := float64(len(good)) / float64(len(c.nodes)) * 100
		log.Infof("[NODES]: good nodes %d/%d (%.2f%%)\n", len(good), len(c.nodes), perc)
		if len(good) == 0 {
			continue
		}
		if len(good) == len(c.nodes) {
			log.Warn("[NODES]: all nodes are good, exit")
			log.Fatal("exit") // TODO: graceful exit
			return
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
	}
}

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
