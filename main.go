package main

import (
	"context"
	"fmt"
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
	"github.com/1F47E/go-btc-xray/pkg/storage"
)

const (
	banner = `
   	 __  __     ______     ______     __  __    
	/\_\_\_\   /\  == \   /\  __ \   /\ \_\ \   
	\/_/\_\/_  \ \  __<   \ \  __ \  \ \____ \  
	  /\_\/\_\  \ \_\ \_\  \ \_\ \_\  \/\_____\ 
	  \/_/\/_/   \/_/ /_/   \/_/\/_/   \/_____/ 

	`
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
)

func main() {
	fmt.Println(Green, banner, Reset)

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
	if cfg.Gui {
		ui = gui.New(ctx, guiCh)
		go ui.Start()
	}

	// RPC CLIENT
	c := client.NewClient(ctx, log, guiCh)

	if os.Getenv("GUI_DEBUG") != "1" {
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
