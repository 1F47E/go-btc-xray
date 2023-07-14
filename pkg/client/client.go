package client

import (
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/logger"
	"net"
	"sync"
	"time"
)

var mu = sync.Mutex{}
var cfg = config.New()
var log *logger.Logger = logger.New()

// batch of new addresses. not block the sented (listner) goroutine
var newNodesCh = make(chan []string, 100)

type Client struct {
	nodes map[string]*Node
}

func NewClient(nodes []*Node) *Client {
	c := Client{
		nodes: make(map[string]*Node),
	}
	for _, n := range nodes {
		c.nodes[n.Endpoint()] = n
	}
	return &c
}

// TODO: run in batches + connect to the new nodes
func (c *Client) NewNodesListner() {
	for addrs := range newNodesCh {
		log.Debugf("[NODES]: got batch %d new nodes\n", len(addrs))
		mu.Lock()
		for _, addr := range addrs {
			tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				log.Debugf("[NODES]: failed to resolve addr %s: %v\n", addr, err)
				continue
			}
			n := Node{Addr: *tcpAddr}
			// add only if new
			addr := n.EndpointSafe()
			if _, ok := c.nodes[addr]; !ok {
				c.nodes[addr] = &n
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
		if node.IsDead {
			continue
		}
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
		waitlist := make([]*Node, 0)
		mu.Lock()
		for _, node := range c.nodes {
			if node.IsConnected() {
				continue
			}
			if node.PingCount > 0 {
				continue
			}
			if node.IsDead {
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

func (c *Client) UpdateNodes() {
	for {
		time.Sleep(10 * time.Second)
		log.Debugf("[NODES]: update nodes. total now %d\n", len(c.nodes))

		// filter good nodes
		good := make([]string, 0)
		mu.Lock()
		toDelete := make([]string, 0)
		for addr, node := range c.nodes {
			if node.IsGood() {
				good = append(good, addr)
			}

			if node.IsDead {
				log.Debugf("[NODES]: node %s is dead\n", addr)
				toDelete = append(toDelete, addr)
			}
		}
		for _, addr := range toDelete {
			delete(c.nodes, addr)
		}
		mu.Unlock()
		perc := float64(len(good)) / float64(len(c.nodes)) * 100
		log.Debugf("[NODES]: good nodes %d/%d (%.2f%%)\n", len(good), len(c.nodes), perc)
		if len(good) == 0 {
			continue
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
