package client

import (
	"encoding/json"
	"fmt"
	"go-btc-downloader/pkg/config"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/miekg/dns"
)

var cfg = config.New()

type Node struct {
	Address net.IP    `json:"address"`
	Conn    *net.Conn `json:"-"`
}

func (n *Node) Connect() {
	fmt.Printf("connecting to %v\n", n.Address.String())
	conn, err := net.Dial("tcp", n.Address.String()+":"+strconv.Itoa(cfg.NodesPort))
	if err != nil {
		return
	}
	fmt.Printf("connected to %v\n", n.Address.String())
	n.Conn = &conn
	// log connection
	fmt.Printf("connection: %v", conn)
	// TODO: send version and ping
}

func NodesRead() ([]*Node, error) {
	ret := make([]*Node, 0)
	// read from json
	fData, err := os.ReadFile(cfg.NodesDB)
	if err != nil {
		return ret, err
	}
	var data []string
	err = json.Unmarshal(fData, &data)
	if err != nil {
		return ret, err
	}
	for _, addr := range data {
		ret = append(ret, &Node{Address: net.ParseIP(addr)})
	}

	return ret, nil
}

func NodesScan() ([]*Node, error) {
	nodes := make([]*Node, 0)
	fmt.Println("Getting nodes from dns seeds... via ", cfg.DnsAddress)
	now := time.Now()
	if cfg.DnsSeeds == nil {
		return nil, fmt.Errorf("no dns seeds")
	}
	for _, seed := range cfg.DnsSeeds {
		// fmt.Printf("Asking seed [%s] for nodes...", seed)
		// fmt.Printf("Dns timeout: %v dnsAddress: %s", cfg.DnsTimeout, cfg.DnsAddress)
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(seed), dns.TypeA)
		c := new(dns.Client)
		c.Net = "tcp"
		c.Timeout = cfg.DnsTimeout
		in, _, err := c.Exchange(m, cfg.DnsAddress)
		if err != nil {
			fmt.Printf("Failed to get nodes from %v: %v\n", seed, err)
			continue
		}
		fmt.Printf("Got %v nodes from %v\n", len(in.Answer), seed)
		// loop through dns records
		for _, ans := range in.Answer {
			// check that record is valid
			if _, ok := ans.(*dns.A); !ok {
				continue
			}
			record := ans.(*dns.A)
			// check if already exists
			for _, node := range nodes {
				if node.Address.Equal(record.A) {
					continue
				}
			}
			nodes = append(nodes, &Node{Address: record.A})
		}
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("No nodes found")
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
	err = os.WriteFile(cfg.NodesDB, fDataJson, 0644)
	if err != nil {
		log.Fatalf("failed to write nodes: %v", err)
	}
	log.Printf("saved %v nodes to %v\n", len(nodes), cfg.NodesDB)
	return nodes, nil
}
