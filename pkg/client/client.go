package client

import (
	"log"
	"net"
	"sync"
	"time"
)

var mu = sync.Mutex{}

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
		log.Printf("[NODES]: got batch %d new nodes\n", len(addrs))
		mu.Lock()
		for _, addr := range addrs {
			tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				log.Printf("[NODES]: failed to resolve addr %s: %v\n", addr, err)
				continue
			}
			n := Node{Addr: *tcpAddr}
			c.nodes[n.Endpoint()] = &n
		}
		mu.Unlock()
	}
}

func (c *Client) UpdateNodes() {
	for {
		time.Sleep(10 * time.Second)
		log.Printf("[NODES]: update nodes. total now %d\n", len(c.nodes))

		// filter good nodes
		good := make([]string, 0)
		mu.Lock()
		for addr, node := range c.nodes {
			if node.IsGood() {
				good = append(good, addr)
			}

			if !node.IsAlive {
				log.Printf("[NODES]: node %s is dead\n", addr)
				delete(c.nodes, addr)
			}
		}
		mu.Unlock()
		perc := float64(len(good)) / float64(len(c.nodes)) * 100
		log.Printf("[NODES]: good nodes %d/%d (%.2f%%)\n", len(good), len(c.nodes), perc)
		if len(good) == 0 {
			continue
		}

		// save to json file
		// j, err := json.MarshalIndent(good, "", "  ")
		// if err != nil {
		// 	log.Printf("[NODES]: failed to marshal nodes: %v\n", err)
		// 	continue
		// }
		// err = os.WriteFile(cfg.NodesDB, j, 0644)
		// if err != nil {
		// 	log.Printf("[NODES]: failed to write nodes: %v\n", err)
		// 	continue
		// }
		// log.Printf("[NODES]: saved %d node to %v\n", len(good), cfg.PeersDB)
	}
}
