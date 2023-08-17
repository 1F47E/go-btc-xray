package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
)

type Config struct {
	Network          Network
	NodesFilename    string
	NodesPort        uint16
	NodeTimeout      time.Duration
	PingInterval     time.Duration
	PingTimeout      time.Duration
	PingRetrys       int
	ListenInterval   time.Duration
	ConnectionsLimit int
	LogsDir          string
	LogsFilename     string
	DataDir          string

	DnsAddress string
	DnsTimeout time.Duration
	DnsSeeds   []string

	Gui bool

	// Wire
	Pver uint32

	// var btcnet = wire.MainNet
	Btcnet wire.BitcoinNet
}

func New() *Config {
	cfg := &Config{
		// var dnsAddress = "1.1.1.1:53" // cloudflare dns, 2x slower
		// google dns
		// DnsAddress:     "8.8.8.8:53",
		// cloudflare dns
		DnsAddress: "1.1.1.1:53",
		// quad dns
		// DnsAddress:     "9.9.9.9:53",

		Pver:           wire.ProtocolVersion, // 70016
		NodeTimeout:    5 * time.Second,
		PingInterval:   1 * time.Minute,
		PingTimeout:    15 * time.Second,
		PingRetrys:     3,
		ListenInterval: 1 * time.Second,
		LogsDir:        "logs",
		LogsFilename:   fmt.Sprintf("logs_%s.log", time.Now().Format("2006-01-02_15-04-05")),
		DataDir:        "data",
		Gui:            os.Getenv("GUI") != "0", // enabled by default
		// Pver: 70013,
	}
	if os.Getenv("DEBUG") == "1" {
		cfg.ConnectionsLimit = 10
	} else {
		cfg.ConnectionsLimit = 50
	}
	// override connections limit
	if os.Getenv("CONN") != "" {
		conn, err := strconv.Atoi(os.Getenv("CONN"))
		if err != nil {
			log.Fatalf("error converting CONN env variable to int: %v", err)
		}
		cfg.ConnectionsLimit = conn
	}
	if os.Getenv("TESTNET") == "1" {
		cfg.Network = NetworkTestnet
		cfg.Btcnet = wire.TestNet3
		cfg.DnsTimeout = 10 * time.Second
		cfg.NodesFilename = "testnet.json"
		cfg.NodesPort = 18333
		cfg.DnsSeeds = []string{
			"testnet-seed.bitcoin.jonasschnelli.ch",
			"seed.tbtc.petertodd.org",
			"seed.testnet.bitcoin.sprovoost.nl",
			"testnet-seed.bluematt.me",
		}
	} else {
		cfg.Network = NetworkMainnet
		cfg.Btcnet = wire.MainNet

		cfg.DnsTimeout = 5 * time.Second
		cfg.NodesFilename = "mainnet.json"
		cfg.NodesPort = 8333
		cfg.DnsSeeds = []string{
			"dnsseed.emzy.de",
			"dnsseed.bluematt.me",
			"dnsseed.bitcoin.dashjr.org",
			"seed.bitcoin.sipa.be",
			"seed.bitcoinstats.com",
			"seed.bitcoin.jonasschnelli.ch",
			"seed.btc.petertodd.org",
			"seed.bitcoin.sprovoost.nl",
			"seed.bitcoin.wiz.biz",
			"seed.bitnodes.io",
		}
	}
	return cfg
}
