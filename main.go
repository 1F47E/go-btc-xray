package main

import (
	"go-btc-downloader/pkg/client"
	"log"
	"sync"
)

func main() {
	// get from file
	// cfg := config.New()
	// f := cfg.NodesDB
	// f := cfg.PeersDB
	// nodes, err := client.NodesRead(f)
	// if err != nil {
	// 	log.Fatalf("failed to read nodes: %v", err)
	// }

	// get from dns
	nodes, err := client.SeedScan()
	if err != nil {
		log.Fatalf("failed to scan nodes: %v", err)
	}

	// connect to first node
	if len(nodes) == 0 {
		log.Fatalf("no nodes found")
	}

	// debug
	// random cut first 10
	// rand shuffle nodes
	// rand.Seed(time.Now().UnixNano())
	// rand.Shuffle(len(nodes), func(i, j int) {
	// 	nodes[i], nodes[j] = nodes[j], nodes[i]
	// })
	// nodes = nodes[:5]

	// monitor new peers, report
	c := client.NewClient(nodes)
	go c.UpdateNodes()

	wg := sync.WaitGroup{}

	// connect to random node (debug)
	// rand.Seed(time.Now().UnixNano())
	// randInt := rand.Intn(len(nodes))
	// node := nodes[randInt]
	// wg.Add(1)
	// go node.Connect()

	// connect to all nodes
	go c.Connector()
	// go c.NewNodesListner()
	// always block for now
	wg.Add(1)
	wg.Wait()
}
