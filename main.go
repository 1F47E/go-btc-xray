package main

import (
	"fmt"
	"go-btc-downloader/pkg/client"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/dns"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/logger"
	"math/rand"
	"os"
	"sync"
	"time"
)

func main() {
	cfg := config.New()
	guiCh := make(chan gui.IncomingData, 100) // do not block sending gui updated
	guiLogsCh := make(chan string, 100)
	log := logger.New(guiLogsCh)

	if os.Getenv("GUI") != "0" {
		g := gui.New()
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
		go g.Start()
	}
	// TODO: make graceful shutdown
	// ctx, cancel := context.WithCancel(context.Background())

	// start client and listen for new nodes to connect
	c := client.NewClient(log, guiCh)
	go c.Start()
	log.Debugf("client started")

	// scan seed nodes, add them to the client
	if os.Getenv("GUI_SIM") != "1" {
		go func() {
			d := dns.New(log)
			addrs, err := d.Scan()
			if err != nil {
				log.Fatalf("failed to scan nodes: %v", err)
			}
			if len(addrs) == 0 {
				log.Fatalf("no node addresses found")
			}
			log.Debugf("dns scan found %d nodes", len(addrs))
			c.AddNodes(addrs)
		}()
	}

	// get from file
	// cfg := config.New()
	// f := cfg.NodesDB
	// f := cfg.PeersDB
	// nodes, err := client.NodesRead(f)
	// if err != nil {
	// 	log.Fatalf("failed to read nodes: %v", err)
	// }

	if os.Getenv("GUI_SIM") == "1" {
		// send fake logs to debug gui
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

		// send fake conn data to debug gui
		go func() {
			for {
				time.Sleep(100 * time.Millisecond)
				rInt := rand.Intn(cfg.ConnectionsLimit)
				guiCh <- gui.IncomingData{
					Connections: rInt,
				}
			}
		}()
		// send fake nodes total to debug
		go func() {
			for {
				time.Sleep(200 * time.Millisecond)
				rInt := rand.Intn(100)
				guiCh <- gui.IncomingData{
					NodesTotal: 100 + rInt,
				}
			}
		}()
		// // send queue
		go func() {
			for {
				time.Sleep(200 * time.Millisecond)
				rInt := rand.Intn(100)
				guiCh <- gui.IncomingData{
					NodesQueued: 20 + rInt,
				}
			}
		}()
		// // send good nodes to debug
		go func() {
			for {
				time.Sleep(300 * time.Millisecond)
				rInt := rand.Intn(100)
				guiCh <- gui.IncomingData{
					NodesGood: 100 + rInt,
				}
			}
		}()
		// // send dead nodes to debug
		go func() {
			for {
				time.Sleep(400 * time.Millisecond)
				rInt := rand.Intn(100)
				guiCh <- gui.IncomingData{
					NodesDead: 100 + rInt,
				}
			}
		}()
		// // send debug msg in and out
		go func() {
			for {
				time.Sleep(500 * time.Millisecond)
				rIntIn := rand.Intn(100)
				rIntOut := rand.Intn(100)
				guiCh <- gui.IncomingData{
					MsgIn:  rIntIn,
					MsgOut: rIntOut,
				}
			}
		}()
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
