package main

import (
	"go-btc-downloader/pkg/client"
	"log"
	"sync"
)

func main() {
	// cfg := config.New()
	nodes, err := client.NodesRead()
	// nodes, err := client.NodesScan()
	if err != nil {
		log.Fatalf("failed to read nodes: %v", err)
	}

	// connect to first node
	if len(nodes) == 0 {
		log.Fatalf("no nodes found")
	}

	// monitor new peers, report
	c := client.NewClient(nodes)
	go c.UpdatePeers()

	wg := sync.WaitGroup{}

	// connect to random node (debug)
	// rand.Seed(time.Now().UnixNano())
	// randInt := rand.Intn(len(nodes))
	// node := nodes[randInt]
	// wg.Add(1)
	// go node.Connect()

	// connect to all nodes
	for _, node := range nodes {
		go node.Connect()
	}
	// always block for now
	wg.Add(1)
	wg.Wait()
}
