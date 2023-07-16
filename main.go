package main

import (
	"context"
	"go-btc-downloader/pkg/client"
	"go-btc-downloader/pkg/dns"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// do not block, sending gui updates
	guiCh := make(chan gui.IncomingData, 42)
	log := logger.New(guiCh)

	// context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// GUI
	var ui *gui.GUI
	if os.Getenv("GUI") != "0" {
		ui = gui.New(ctx, guiCh)
		go ui.Start()
	}

	// CLIENT
	// start client and listen for new nodes to connect
	c := client.NewClient(ctx, log, guiCh)
	go c.Start()

	// DNS SCAN
	// if not debugging gui
	if os.Getenv("GUI_DEBUG") != "1" {
		// scan seed nodes, add them to the client
		go func() {
			d := dns.New(log)
			addrs := d.Scan()
			if len(addrs) == 0 {
				log.Fatalf("no seed nodes found")
			}
			c.AddNodes(addrs)
		}()
	}

	// GRACEFUL SHUTDOWN

	// block and wait for the OS signal to exit
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Debug("received exit signal, canceling ctx")
		cancel()
	}()
	// blocking, waiting for all the goroutines to exit
	<-ctx.Done()
	log.Debug("context canceled, exiting")
	log.ResetToStdout()
	// exit from GUI
	if ui != nil {
		go ui.Stop()
	}
	// Closing all the connections
	c.Stop()
}
