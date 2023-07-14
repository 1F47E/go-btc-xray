package main

import (
	"go-btc-downloader/pkg/client"
	"go-btc-downloader/pkg/dns"
	"go-btc-downloader/pkg/logger"
	"sync"
)

func main() {
	// ctx, cancel := context.WithCancel(context.Background())

	log := logger.New()

	// get from file
	// cfg := config.New()
	// f := cfg.NodesDB
	// f := cfg.PeersDB
	// nodes, err := client.NodesRead(f)
	// if err != nil {
	// 	log.Fatalf("failed to read nodes: %v", err)
	// }

	// get from dns
	addrs, err := dns.Scan()
	if err != nil {
		log.Fatalf("failed to scan nodes: %v", err)
	}

	// connect to first node
	if len(addrs) == 0 {
		log.Fatalf("no node addresses found")
	}

	c := client.NewClient(addrs)
	// debug
	// random cut first 10
	// rand shuffle nodes
	// rand.Seed(time.Now().UnixNano())
	// rand.Shuffle(len(nodes), func(i, j int) {
	// 	nodes[i], nodes[j] = nodes[j], nodes[i]
	// })
	// nodes = nodes[:5]

	// monitor new peers, report
	go c.NodesUpdated()

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
	// exit := make(chan os.Signal, 1)
	// signal.Notify(exit, os.Interrupt)
	// <-exit
	// cancel()
	// log.Warn("exiting")
	wg.Add(1)
	wg.Wait()
}
