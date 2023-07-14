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
	Addr      net.TCPAddr `json:"address"`
	Conn      net.Conn    `json:"-"`
	PingNonce uint64
	PingCount uint8
	IsDead    bool // mark to delete
	newAddrCh chan []string
}

func NewNode(addr net.TCPAddr, newAddrCh chan []string) *Node {
	return &Node{
		Addr:      addr,
		newAddrCh: newAddrCh,
	}
}

func (n *Node) Endpoint() string {
	return fmt.Sprintf("%s:%d", n.Addr.IP.String(), n.Addr.Port)
}

func (n *Node) EndpointSafe() string {
	return fmt.Sprintf("[%s]:%d", n.Addr.IP.String(), n.Addr.Port)
}

func (n *Node) IsGood() bool {
	return (!n.IsDead && n.PingCount > 0)
}

func (n *Node) IsConnected() bool {
	return n.Conn != nil
}

func (n *Node) Connect() {
	a := fmt.Sprintf("▶︎ %s:", n.Endpoint())
	log.Debugf("%s connecting...\n", a)
	defer func() {
		n.Conn = nil
		log.Warnf("%s closed\n", a)
	}()
	timeout := time.Duration(5 * time.Second)
	conn, err := net.DialTimeout("tcp", n.Endpoint(), timeout)
	if err != nil {
		n.IsDead = true
		log.Debugf("%s failed to connect: %v\n", a, err)
		return
	}
	log.Debugf("%s connected\n", a)
	n.Conn = conn
	// handle answers
	// TODO: fix goroutine leak
	go n.listen()

	// ===== NEGOTIATION
	// 1. sending version
	if n.Conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	msg, err := n.localVersionMsg()
	if err != nil {
		log.Debugf("%s failed to create version: %v", a, err)
		return
	}
	log.Debugf("%s sending version...\n", a)
	cnt, err := wire.WriteMessageN(n.Conn, msg, cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Debugf("[%s]: failed to write version: %v", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// 2. send addr v2
	// if pver < wire.AddrV2Version {
	// 	return nil
	// }
	if n.Conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	log.Debugf("%s sending sendaddrv2...\n", a)
	sendAddrMsg := wire.NewMsgSendAddrV2()
	cnt, err = wire.WriteMessageN(n.Conn, sendAddrMsg, cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Debugf("%s failed to write sendaddrv2: %v\n", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// 3. send verAck
	if n.Conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	log.Debugf("%s sending verack...\n", a)
	cnt, err = wire.WriteMessageN(n.Conn, wire.NewMsgVerAck(), cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Debugf("%s failed to write verack: %v\n", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// ====== NEGOTIATION DONE

	time.Sleep(1 * time.Second)

	// ask for peers once
	if n.Conn == nil {
		log.Debugf("%s disconnected\n", a)
		return
	}
	log.Debugf("%s sending getaddr...\n", a)
	msgAddr := wire.NewMsgGetAddr()
	cnt, err = wire.WriteMessageN(n.Conn, msgAddr, cfg.Pver, cfg.Btcnet)
	if err != nil {
		// Log and handle the error
		log.Debugf("%s failed to write getaddr: %v\n", a, err)
		return
	}
	log.Debugf("%s OK. sent %d bytes\n", a, cnt)

	// send pings
	time.Sleep(1 * time.Second)
	for {
		if n.Conn == nil {
			log.Debugf("%s disconnected\n", a)
			return
		}
		if n.PingCount >= 1 {
			log.Debugf("%s ping count reached\n", a)
			return
		}
		log.Debugf("%s sending ping...\n", a)
		nonceBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(2, 62))))
		n.PingNonce = nonceBig.Uint64()
		msgPing := wire.NewMsgPing(n.PingNonce)
		cnt, err = wire.WriteMessageN(n.Conn, msgPing, cfg.Pver, cfg.Btcnet)
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

/*
python genesis.py -z "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks" -t 1501755824
04ffff001d0104455468652054696d65732030332f4a616e2f32303039204368616e63656c6c6f72206f6e206272696e6b206f66207365636f6e64206261696c6f757420666f722062616e6b73
algorithm: SHA256
merkle hash: 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
pszTimestamp: The Times 03/Jan/2009 Chancellor on brink of second bailout for banks
pubkey: 04678afdb0fe5548271967f1a67130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112de5c384df7ba0b8d578a4c702b6bf11d5f
time: 1501755824
bits: 0x1d00ffff
Searching for genesis hash..
183984.0 hash/s, estimate: 6.5 h
nonce: 835054047
genesis hash: 00000000ad3d3d6aa486313522fdd4328509feefe8c37ead2a609884c6cbab92
*/
// func HashBlock() []byte {

// 	version := 1
// 	var time int = 1231006505
// 	var bits int = 0x1d00ffff
// 	var nonce int = 2083236893
// 	merkleRoot, _ := hex.DecodeString("3BA3EDFD7A7B12B27AC72C3E67768F617FC81BC3888A51323A9FB8AA4B1E5E4A")
// 	return nil
// }

// localVersionMsg creates a version message that can be used to send to the
// remote peer.
func (n *Node) localVersionMsg() (*wire.MsgVersion, error) {
	var blockNum int32
	theirNA := wire.NetAddress{
		Services: wire.SFNodeNetwork,
		IP:       net.ParseIP("::ffff:127.0.0.1"),
		Port:     0,
	}

	// Older nodes previously added the IP and port information to the
	// address manager which proved to be unreliable as an inbound
	// connection from a peer didn't necessarily mean the peer itself
	// accepted inbound connections.
	//
	// Also, the timestamp is unused in the version message.
	ourNA := &wire.NetAddress{
		Services: wire.SFNodeNetwork,
	}

	// Generate a unique nonce for this peer so self connections can be
	// detected.  This is accomplished by adding it to a size-limited map of
	// recently seen nonces.
	nonceBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(2, 62))))
	n.PingNonce = nonceBig.Uint64()

	// Version message.
	msg := wire.NewMsgVersion(ourNA, &theirNA, n.PingNonce, blockNum)
	_ = msg.AddUserAgent("btcd", "0.23.3", "")
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(cfg.Pver)
	// Advertise if inv messages for transactions are desired.
	// msg.DisableRelayTx = p.cfg.DisableRelayTx

	return msg, nil
}
