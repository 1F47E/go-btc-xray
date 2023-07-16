package node

import (
	"context"
	"fmt"
	"go-btc-downloader/pkg/cmd"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/logger"
	"net"
	"time"
)

var cfg = config.New()
var log *logger.Logger = logger.New(nil)

type Status int

const (
	New = iota
	Connected
	Disconnected
	Dead
)

type Node struct {
	ip        string
	conn      net.Conn
	pingNonce uint64
	pingCount uint8
	status    Status
	isGood    bool // handshake is OK
	version   int32
	newAddrCh chan []string
}

func NewNode(ip string, newAddrCh chan []string) *Node {
	return &Node{
		ip:        ip,
		newAddrCh: newAddrCh,
	}
}

// NOT USED
func (n *Node) Resolve() {
	_, err := net.ResolveTCPAddr("tcp", n.ip)
	if err != nil {
		n.status = Dead
	}
}

// returns true if connection was closed
func (n *Node) Disconnect() bool {
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
		n.status = Disconnected // TODO: remove?
		return true
	}
	return false
}

func (n *Node) IsNew() bool {
	return n.status == New
}

func (n *Node) IsDead() bool {
	return n.status == Dead
}

func (n *Node) IsConnected() bool {
	return n.status == Connected && n.conn != nil
}

// debug
func (n *Node) HasConnection() bool {
	return n.conn != nil
}

func (n *Node) IsGood() bool {
	return n.isGood
}

func (n *Node) Endpoint() string {
	return fmt.Sprintf("%s:%d", n.ip, cfg.NodesPort)
}

// wrapper with brackets for ipv6
func (n *Node) EndpointSafe() string {
	return fmt.Sprintf("[%s]:%d", n.ip, cfg.NodesPort)
}

func (n *Node) Connect(ctx context.Context) {
	a := fmt.Sprintf("▶︎ %s", n.ip)
	log.Debugf("%s connecting...\n", a)
	defer func() {
		n.conn = nil
		n.status = Disconnected
		log.Debugf("%s closed\n", a)
	}()
	conn, err := net.DialTimeout("tcp", n.EndpointSafe(), cfg.NodeTimeout)
	if err != nil {
		n.status = Dead
		log.Debugf("%s failed to connect: %v\n", a, err)
		return
	}
	log.Debugf("%s connected\n", a)
	n.conn = conn
	n.status = Connected
	// handle answers
	// exit on closed connection or context cancel
	go n.listen(ctx)

	// ===== NEGOTIATION
	// TODO: make it in a separate negotiation function
	// 1. sending version
	log.Debugf("%s sending version...\n", a)
	n.UpdateNonce()
	err = cmd.SendVersion(n.conn, n.pingNonce)
	if err != nil {
		log.Errorf("[%s]: failed to write version: %v", a, err)
		return
	}
	log.Debugf("%s OK\n", a)

	// 2. send addr v2
	log.Debugf("%s sending sendaddrv2...\n", a)
	err = cmd.SendAddrV2(n.conn)
	if err != nil {
		log.Errorf("%s failed to write sendaddrv2: %v\n", a, err)
		return
	}
	log.Debugf("%s OK\n", a)

	// 3. send verAck
	// TODO: read version first
	time.Sleep(2 * time.Second)
	log.Debugf("%s sending verack...\n", a)
	err = cmd.SendVerAck(n.conn)
	if err != nil {
		log.Errorf("%s failed to write verack: %v\n", a, err)
		return
	}
	log.Debugf("%s OK\n", a)
	n.isGood = true

	// ====== NEGOTIATION DONE
	time.Sleep(1 * time.Second)

	// ask for peers once
	log.Debugf("%s sending getaddr...\n", a)
	err = cmd.SendGetAddr(n.conn)
	if err != nil {
		log.Errorf("%s failed to write getaddr: %v\n", a, err)
		return
	}
	log.Debugf("%s OK\n", a)

	// send pings
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n.conn == nil {
				log.Debugf("%s disconnected\n", a)
				return
			}
			if n.pingCount >= 1 {
				log.Debugf("%s ping count reached\n", a)
				return
			}
			log.Debugf("%s sending ping...\n", a)
			n.UpdateNonce()
			err = cmd.SendPing(n.conn, n.pingNonce)
			if err != nil {
				log.Debugf("%s failed to write ping: %v\n", a, err)
				return
			}
			log.Debugf("%s OK\n", a)
		}
	}
}
