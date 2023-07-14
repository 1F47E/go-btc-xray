package node

import (
	"crypto/rand"
	"fmt"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/logger"
	"math"
	"math/big"
	"net"
	"time"

	wire "github.com/btcsuite/btcd/wire"
)

var cfg = config.New()
var log *logger.Logger = logger.New()

type Node struct {
	addr      net.TCPAddr
	conn      net.Conn
	pingNonce uint64
	pingCount uint8
	isDead    bool // mark to delete
	newAddrCh chan []string
}

func NewNode(addr net.TCPAddr, newAddrCh chan []string) *Node {
	return &Node{
		addr:      addr,
		newAddrCh: newAddrCh,
	}
}

func (n *Node) IP() net.IP {
	return n.addr.IP
}

func (n *Node) IsDead() bool {
	return n.isDead
}

func (n *Node) Endpoint() string {
	return fmt.Sprintf("%s:%d", n.addr.IP.String(), n.addr.Port)
}

func (n *Node) EndpointSafe() string {
	return fmt.Sprintf("[%s]:%d", n.addr.IP.String(), n.addr.Port)
}

func (n *Node) IsGood() bool {
	return (!n.isDead && n.pingCount > 0)
}

func (n *Node) IsConnected() bool {
	return n.conn != nil
}

func (n *Node) Connect() {
	a := fmt.Sprintf("▶︎ %s:", n.Endpoint())
	log.Debugf("%s connecting...\n", a)
	defer func() {
		n.conn = nil
		log.Warnf("%s closed\n", a)
	}()
	timeout := time.Duration(5 * time.Second)
	conn, err := net.DialTimeout("tcp", n.Endpoint(), timeout)
	if err != nil {
		n.isDead = true
		log.Debugf("%s failed to connect: %v\n", a, err)
		return
	}
	log.Debugf("%s connected\n", a)
	n.conn = conn
	// handle answers
	// exit on closed connection
	go n.listen()

	// ===== NEGOTIATION
	// 1. sending version
	if n.conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	msg, err := n.localVersionMsg()
	if err != nil {
		log.Debugf("%s failed to create version: %v", a, err)
		return
	}
	log.Debugf("%s sending version...\n", a)
	cnt, err := wire.WriteMessageN(n.conn, msg, cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Debugf("[%s]: failed to write version: %v", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// 2. send addr v2
	// if pver < wire.AddrV2Version {
	// 	return nil
	// }
	if n.conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	log.Debugf("%s sending sendaddrv2...\n", a)
	sendAddrMsg := wire.NewMsgSendAddrV2()
	cnt, err = wire.WriteMessageN(n.conn, sendAddrMsg, cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Debugf("%s failed to write sendaddrv2: %v\n", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// 3. send verAck
	if n.conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	log.Debugf("%s sending verack...\n", a)
	cnt, err = wire.WriteMessageN(n.conn, wire.NewMsgVerAck(), cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Debugf("%s failed to write verack: %v\n", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// ====== NEGOTIATION DONE
	time.Sleep(1 * time.Second)

	// ask for peers once
	if n.conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	log.Debugf("%s sending getaddr...\n", a)
	msgAddr := wire.NewMsgGetAddr()
	cnt, err = wire.WriteMessageN(n.conn, msgAddr, cfg.Pver, cfg.Btcnet)
	if err != nil {
		// Log and handle the error
		log.Debugf("%s failed to write getaddr: %v\n", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// send pings
	time.Sleep(1 * time.Second)
	for {
		if n.conn == nil {
			log.Debugf("%s disconnected\n", a)
			return
		}
		if n.pingCount >= 1 {
			log.Debugf("%s ping count reached\n", a)
			return
		}
		log.Debugf("%s sending ping...\n", a)
		nonceBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(2, 62))))
		n.pingNonce = nonceBig.Uint64()
		msgPing := wire.NewMsgPing(n.pingNonce)
		cnt, err = wire.WriteMessageN(n.conn, msgPing, cfg.Pver, cfg.Btcnet)
		if err != nil {
			log.Debugf("%s failed to write ping: %v\n", a, err)
			return
		}
		log.Debugf("%s OK. sent %d bytes\n", a, cnt)
		time.Sleep(1 * time.Minute)
	}

	// TODO: send genesis block
	// blocks := make([][]byte, 0)
	// genesisHash, err := hex.DecodeString("00000000ad3d3d6aa486313522fdd4328509feefe8c37ead2a609884c6cbab92")
	// if err != nil {
	// 	log.Fatalf("failed to decode genesis hash: %v", err)
	// }
	// blocks = append(blocks, genesisHash)

	// var inventoryBuff bytes.Buffer
	// binary.Write(&inventoryBuff, binary.LittleEndian, uint32(2))
	// for _, block := range blocks {
	// 	inventoryBuff.Write(block[:])
	// }

	// inventory := make([]byte, inventoryBuff.Len())
	// _, err = inventoryBuff.Read(inventory)
	// if err != nil {
	// 	log.Fatalf("failed to read inventory: %v", err)
	// }
	// err = wire.WriteGetData(conn, inventory)
	// if err != nil {
	// 	log.Fatalf("failed to write getdata: %v", err)
	// }
}
