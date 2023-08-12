//
//  __  __     ______     ______     __  __
// /\_\_\_\   /\  == \   /\  __ \   /\ \_\ \
// \/_/\_\/_  \ \  __<   \ \  __ \  \ \____ \
//   /\_\/\_\  \ \_\ \_\  \ \_\ \_\  \/\_____\
//   \/_/\/_/   \/_/ /_/   \/_/\/_/   \/_____/
// bitcoin network scanner
//

package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/1F47E/go-btc-xray/pkg/client"
	"github.com/1F47E/go-btc-xray/pkg/config"
	"github.com/1F47E/go-btc-xray/pkg/dns"
	"github.com/1F47E/go-btc-xray/pkg/gui"
	"github.com/1F47E/go-btc-xray/pkg/logger"
	"github.com/1F47E/go-btc-xray/pkg/printer"
	"github.com/1F47E/go-btc-xray/pkg/storage"
)

func main() {
	printer.Banner()

	var err error
	cfg := config.New()

	guiCh := make(chan gui.IncomingData, 42)
	log := logger.New(guiCh)

	// create temp folders
	err = storage.Bootstrap()
	if err != nil {
		log.Fatalf("failed to bootstrap the storage: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// TUI
	var ui *gui.GUI
	if cfg.Gui {
		ui = gui.New(ctx, guiCh)
		go ui.Start()
	}

	// RPC CLIENT
	c := client.NewClient(ctx, log, guiCh)

	if os.Getenv("DRY_RUN") != "1" {
		// DNS SCAN
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

	log.Debug("waiting for the context to be canceled")
	// blocking, waiting for all the goroutines to exit
	<-ctx.Done()
	log.Debug("context canceled, exiting")
	log.ResetToStdout()
	// exit from GUI
	if ui != nil {
		go ui.Stop()
	}
	// RPC disconnect from all the nodes
	c.Disconnect()
}
