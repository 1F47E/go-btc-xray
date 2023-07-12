package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/miekg/dns"
)

var dnsAddress = "8.8.8.8:53" // google dns
// var dnsAddress = "1.1.1.1:53" // cloudflare dns, 2x slower

var dnsTimeout = 3 * time.Second
var dnsSeeds = []string{
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

var nodesFile = "data/nodes.json"

// var nodePort = 8333

type node struct {
	Address net.IP `json:"address"`
}

var nodes []*node

func main() {
	fmt.Println("Getting nodes from dns seeds... via ", dnsAddress)
	now := time.Now()
	for _, seed := range dnsSeeds {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(seed), dns.TypeA)
		c := new(dns.Client)
		c.Net = "tcp"
		c.Timeout = dnsTimeout
		in, _, err := c.Exchange(m, dnsAddress)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("Got %v nodes from %v\n", len(in.Answer), seed)
		// loop through dns records
		for _, ans := range in.Answer {
			// check that record is valid
			if record, ok := ans.(*dns.A); ok {
				nodes = append(nodes, &node{Address: record.A})
			}
		}
	}
	fmt.Printf("Got %v nodes in %v\n", len(nodes), time.Since(now))

	// save nodes as json
	fData := make([]string, len(nodes))
	for i, node := range nodes {
		fData[i] = node.Address.String()
	}
	fDataJson, err := json.MarshalIndent(fData, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal nodes: %v", err)
	}
	err = os.WriteFile(nodesFile, fDataJson, 0644)
	if err != nil {
		log.Fatalf("failed to write nodes: %v", err)
	}
	log.Printf("saved %v nodes to %v\n", len(nodes), nodesFile)
}
