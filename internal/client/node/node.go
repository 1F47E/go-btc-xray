package node

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"net"
	"time"

	"github.com/1F47E/go-btc-xray/internal/cmd"
	"github.com/1F47E/go-btc-xray/internal/config"
	"github.com/1F47E/go-btc-xray/internal/logger"
)

var cfg = config.New()

type status int

const (
	new = iota
	connecting
	connected
	disconnected
	dead
)

type Result struct {
	Node  *Node
	Error error
}

type Node struct {
	log       *logger.Logger
	ip        string
	conn      net.Conn
	pingNonce uint64
	pongCount uint8
	status    status
	version   int32
	newAddrCh chan []string
}

func NewNode(log *logger.Logger, ip string, newAddrCh chan []string) *Node {
	n := Node{
		log:       log,
		ip:        ip,
		newAddrCh: newAddrCh,
	}
	n.UpdatePingNonce()
	return &n
}

func (n *Node) Disconnect() bool {
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
		n.status = disconnected
		return true
	}
	return false
}

func (n *Node) UpdatePingNonce() {
	nonceBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(2, 62))))
	n.pingNonce = nonceBig.Uint64()
}

func (n *Node) IsNew() bool {
	return n.status == new
}

func (n *Node) IsDead() bool {
	return n.status == dead
}

func (n *Node) IsConnecting() bool {
	return n.status == connecting
}
func (n *Node) IsConnected() bool {
	return n.status == connected && n.conn != nil
}

func (n *Node) Endpoint() string {
	return fmt.Sprintf("%s:%d", n.ip, cfg.NodesPort)
}

// wrapper with brackets for ipv6 needed for net.Dial
func (n *Node) EndpointSafe() string {
	return fmt.Sprintf("[%s]:%d", n.ip, cfg.NodesPort)
}

// returning error here will consider the node as dead
func (n *Node) Connect(ctx context.Context, resCh chan *Node) error {
	n.status = connecting
	a := fmt.Sprintf("▶︎ %s", n.ip)
	n.log.Debugf("%s connecting...\n", a)
	defer func() {
		n.conn = nil
		n.log.Debugf("%s closed\n", a)
	}()
	conn, err := net.DialTimeout("tcp", n.EndpointSafe(), cfg.NodeTimeout)
	if err != nil {
		n.status = dead
		return fmt.Errorf("%s failed to connect: %w", a, err)
	}
	n.log.Debugf("%s connected\n", a)
	n.conn = conn
	n.status = connected
	// handle answers
	// exit on closed connection or context cancel
	go n.listen(ctx)

	// ===== NEGOTIATION
	// TODO: make it in a separate negotiation function
	// 1. sending version
	n.log.Debugf("%s sending version...\n", a)
	err = cmd.SendVersion(n.conn, n.pingNonce)
	if err != nil {
		return fmt.Errorf("%s failed to write version: %v", a, err)
	}
	n.log.Debugf("%s OK\n", a)

	// 2. send addr v2
	n.log.Debugf("%s sending sendaddrv2...\n", a)
	err = cmd.SendAddrV2(n.conn)
	if err != nil {
		return fmt.Errorf("%s failed to write sendaddrv2: %v", a, err)
	}
	n.log.Debugf("%s OK\n", a)

	// 3. send verAck
	// TODO: read version first
	time.Sleep(2 * time.Second)
	n.log.Debugf("%s sending verack...\n", a)
	err = cmd.SendVerAck(n.conn)
	if err != nil {
		return fmt.Errorf("%s failed to write verack: %v", a, err)
	}
	n.log.Debugf("%s OK\n", a)

	// send results but continue working,
	// asking for peers and sending a few pings
	resCh <- n

	// ====== NEGOTIATION DONE
	time.Sleep(1 * time.Second)

	// ask for peers once
	n.log.Debugf("%s sending getaddr...\n", a)
	err = cmd.SendGetAddr(n.conn)
	if err != nil {
		n.log.Errorf("%s failed to write getaddr: %v", a, err)
		return nil
	}
	n.log.Debugf("%s OK\n", a)

	// Sending a ping to keep a connection while waiting for peers from get addr command
	// Waiting for the pong in the listen goroutine and increment ping count
	// Every ping should have a nonce different from the previous one
	// Disconnect if ping count reached or no pong received
	timeout, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	pingCount := 0
	for {
		select {
		case <-timeout.Done():
			n.log.Warnf("%s ping timeout\n", a)
			return nil
		case <-ctx.Done():
			n.log.Warnf("%s context done, disconnecting\n", a)
			return nil
		case <-ticker.C:
			if n.conn == nil {
				n.log.Debugf("%s disconnected\n", a)
				return nil
			}
			if n.pongCount >= 1 {
				n.log.Debugf("%s pong count reached\n", a)
				return nil
			}
			if pingCount >= cfg.PingRetrys {
				n.log.Debugf("%s ping retry count reached\n", a)
				return nil
			}
			n.log.Debugf("%s sending ping...\n", a)
			err = cmd.SendPing(n.conn, n.pingNonce)
			if err != nil {
				n.log.Errorf("%s failed to write ping: %v", a, err)
				return nil
			}
			pingCount++
			n.log.Debugf("%s OK\n", a)
		}
	}
}
