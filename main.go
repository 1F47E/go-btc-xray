package main

import (
	"fmt"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/dns"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/logger"
	"math/rand"
	"sync"
	"time"
)

func main() {
	cfg := config.New()
	guiCh := make(chan gui.Data, 100) // do not block sending gui updated
	guiLogsCh := make(chan string, 100)
	// TODO: make it optional with env flag
	g := gui.New()
	go func() {
		g.Start()

		// fake data to init
		// data := gui.Data{
		// 	Connections: []float64{1, 2, 3, 4, 5},
		// }
		// g.Update(data)
	}()
	// ctx, cancel := context.WithCancel(context.Background())

	log := logger.New(guiLogsCh)

	// SCAN SEED NODES
	go func() {
		d := dns.New(log)
		addrs, err := d.Scan()
		if err != nil {
			log.Fatalf("failed to scan nodes: %v", err)
		}
		// connect to first node
		if len(addrs) == 0 {
			log.Fatalf("no node addresses found")
		}
		log.Debugf("dns scan found %d nodes", len(addrs))
	}()

	// addrs := make([]string, 0)
	// c := client.NewClient(log, addrs, guiCh)

	// get from file
	// cfg := config.New()
	// f := cfg.NodesDB
	// f := cfg.PeersDB
	// nodes, err := client.NodesRead(f)
	// if err != nil {
	// 	log.Fatalf("failed to read nodes: %v", err)
	// }

	// read and update gui data
	go func() {
		for d := range guiCh {
			g.Update(d)
		}
	}()
	// read logs chan
	go func() {
		for l := range guiLogsCh {
			g.Log(l)
		}
	}()

	// send fake logs to debug
	go func() {
		cnt := 0
		for {
			time.Sleep(1 * time.Second)
			for i := 0; i < 5; i++ {
				cnt = cnt + i
				guiLogsCh <- fmt.Sprintf("test log %d\n", cnt)
			}
		}
	}()

	// send fake conn data to debug
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			rInt := rand.Intn(cfg.ConnectionsLimit)
			guiCh <- gui.Data{
				Connections: rInt,
			}
		}
	}()
	// send fake nodes total to debug
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			rInt := rand.Intn(100)
			guiCh <- gui.Data{
				NodesTotal: 100 + rInt,
			}
		}
	}()
	// send good nodes to debug
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			rInt := rand.Intn(100)
			guiCh <- gui.Data{
				NodesGood: 100 + rInt,
			}
		}
	}()
	// send dead nodes to debug
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			rInt := rand.Intn(100)
			guiCh <- gui.Data{
				NodesDead: 100 + rInt,
			}
		}
	}()

	// debug
	// random cut first 10
	// rand shuffle nodes
	// rand.Seed(time.Now().UnixNano())
	// rand.Shuffle(len(nodes), func(i, j int) {
	// 	nodes[i], nodes[j] = nodes[j], nodes[i]
	// })
	// nodes = nodes[:5]

	// monitor new peers, report
	// go c.Start()

	wg := sync.WaitGroup{}

	// connect to random node (debug)
	// rand.Seed(time.Now().UnixNano())
	// randInt := rand.Intn(len(nodes))
	// node := nodes[randInt]
	// wg.Add(1)
	// go node.Connect()

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
