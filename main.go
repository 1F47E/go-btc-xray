package main

import (
	"go-btc-downloader/pkg/client"
	"go-btc-downloader/pkg/config"
	"log"
	"math/rand"
	"sync"
	"time"
)

func main() {
	cfg := config.New()

	// get from file
	f := cfg.NodesDB
	// f := cfg.PeersDB
	nodes, err := client.NodesRead(f)
	if err != nil {
		log.Fatalf("failed to read nodes: %v", err)
	}

	// get from dns
	// nodes, err := client.NodesScan()
	// if err != nil {
	// 	log.Fatalf("failed to scan nodes: %v", err)
	// }

	// connect to first node
	if len(nodes) == 0 {
		log.Fatalf("no nodes found")
	}

	// debug
	// cut first 10
	// rand shuffle nodes
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(nodes), func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})

	nodes = nodes[:5]

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
