package client

import (
	"log"
	"time"
)

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
		for _, node := range c.nodes {
			for _, peer := range node.peers {
				if _, ok := c.peers[peer.Addr]; !ok {
					c.peers[peer.Addr] = &peer
				}
			}
		}
		log.Printf("[PEERS]: total peers found %d\n", len(c.peers))
	}
}
