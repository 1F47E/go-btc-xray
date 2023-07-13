package client

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

var mu = sync.Mutex{}

type Client struct {
	nodes []*Node
	peers map[string]*Peer
}

func NewClient(nodes []*Node) *Client {
	return &Client{
		nodes: nodes,
		peers: make(map[string]*Peer),
	}
}
func (c *Client) UpdatePeers() {
	for {
		time.Sleep(5 * time.Second)
		mu.Lock()
		for _, node := range c.nodes {
			for _, peer := range node.peers {
				if _, ok := c.peers[peer.Addr]; !ok {
					c.peers[peer.Addr] = &peer
				}
			}
		}
		log.Printf("[PEERS]: total peers found %d\n", len(c.peers))

		// update json
		// TODO: merge peers and nodes
		// save only good peers with ping cnt > 3
		goodPeers := make([]string, 0)
		for addr := range c.peers {
			// if peer.IsAlive {
			// }
			// if
			goodPeers = append(goodPeers, addr)
		}
		mu.Unlock()

		j, err := json.MarshalIndent(goodPeers, "", "  ")
		if err != nil {
			log.Printf("[PEERS]: failed to marshal peers: %v\n", err)
			continue
		}
		// create new file, overwrite
		// uppend to a file
		// f, err := os.OpenFile(cfg.PeersDB, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		// if err != nil {
		// 	log.Printf("[PEERS]: failed to open peers: %v\n", err)
		// 	continue
		// }
		// write to the end
		// _, err = f.Write(j)
		err = os.WriteFile(cfg.PeersDB, j, 0644)
		if err != nil {
			log.Printf("[PEERS]: failed to write peers: %v\n", err)
			continue
		}
		log.Printf("[PEERS]: saved %v peers to %v\n", len(c.peers), cfg.PeersDB)
	}
}
