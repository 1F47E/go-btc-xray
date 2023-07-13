package main

import (
	"go-btc-downloader/pkg/client"
	"log"
	"math/rand"
	"sync"
	"time"
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
	// get random node
	rand.Seed(time.Now().UnixNano())
	randInt := rand.Intn(len(nodes))
	node := nodes[randInt]
	wg := sync.WaitGroup{}
	wg.Add(1)
	node.Connect()
	wg.Wait()
}
