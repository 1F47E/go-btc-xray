package main

import (
	"context"
	"fmt"
	"go-btc-downloader/pkg/client"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/dns"
	"go-btc-downloader/pkg/gui"
	"go-btc-downloader/pkg/logger"
	"go-btc-downloader/pkg/storage"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var err error
	cfg := config.New()

	guiCh := make(chan gui.IncomingData, 42)

	// TODO: refactor this to storage 1 func
	err = storage.CreateDir(cfg.LogsDir)
	if err != nil {
		panic(fmt.Sprintf("failed to create logs dir: %v", err))
	}
	err = storage.CreateDir(cfg.DataDir)
	if err != nil {
		panic(fmt.Sprintf("failed to create data dir: %v", err))
	}

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
			// start the client after seed nodes are added
			go c.Start()
		}()
	}

	// PROFILING
	if os.Getenv("PPROF") == "1" {
		go func() {
			_ = http.ListenAndServe("localhost:6060", nil)
		}()
	}

	// GRACEFUL SHUTDOWN
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
