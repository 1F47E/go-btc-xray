package main

import (
	"go-btc-downloader/pkg/client"
	"go-btc-downloader/pkg/dns"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/logger"
	"os"
	"sync"
)

func main() {
	// cfg := config.New()
	guiCh := make(chan gui.IncomingData, 100) // do not block sending gui updated
	log := logger.New(guiCh)

	if os.Getenv("GUI") != "0" {
		g := gui.New(guiCh)
		go g.Start()
	}
	// TODO: make graceful shutdown
	// ctx, cancel := context.WithCancel(context.Background())

	// start client and listen for new nodes to connect
	c := client.NewClient(log, guiCh)
	go c.Start()
	log.Debugf("client started")

	if os.Getenv("GUI_SIM") != "1" {
		// scan seed nodes, add them to the client
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
