package client

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"go-btc-downloader/pkg/config"
	"log"
	"math"
	"math/big"
	"net"
	"os"
	"time"

	wire "github.com/btcsuite/btcd/wire"

	"github.com/miekg/dns"
)

var cfg = config.New()

type Node struct {
	Addr      net.TCPAddr `json:"address"`
	Conn      net.Conn    `json:"-"`
	PingNonce uint64
	PingCount uint8
	IsAlive   bool
}

func NewNode(addr net.TCPAddr) *Node {
	return &Node{Addr: addr}
}

func (n *Node) Endpoint() string {
	return fmt.Sprintf("%s:%d", n.Addr.IP, n.Addr.Port)
}

func (n *Node) IsGood() bool {
	return (n.IsAlive && n.PingCount > 0)
}

func (n *Node) Connect() {
	a := fmt.Sprintf("-> %s", n.Endpoint())
	log.Printf("[%s]: connecting...\n", a)
	defer func() {
		n.Conn = nil
		fmt.Printf("[%s]: closed\n", a)
	}()
	timeout := time.Duration(5 * time.Second)
	conn, err := net.DialTimeout("tcp", n.Endpoint(), timeout)
	if err != nil {
		n.IsAlive = false
		log.Printf("[%s]: failed to connect: %v\n", a, err)
		return
	}
	n.IsAlive = true
	log.Printf("[%s]: connected\n", a)
	n.Conn = conn
	// handle answers
	go n.connListen()

	// ===== NEGOTIATION
	// 1. sending version
	if n.Conn == nil {
		log.Printf("[%s]: disconnected\n", a)
		return
	}
	msg, err := n.localVersionMsg()
	if err != nil {
		log.Printf("[%s]: failed to create version: %v", a, err)
		return
	}
	log.Printf("[%s]: sending version...\n", a)
	cnt, err := wire.WriteMessageN(n.Conn, msg, cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Printf("[%s]: failed to write version: %v", a, err)
		return
	}
	log.Printf("[%s]: OK. sent %d bytes\n", a, cnt)

	// 2. send addr v2
	// if pver < wire.AddrV2Version {
	// 	return nil
	// }
	if n.Conn == nil {
		log.Printf("[%s]: disconnected\n", a)
		return
	}
	log.Printf("[%s]: sending sendaddrv2...\n", a)
	sendAddrMsg := wire.NewMsgSendAddrV2()
	cnt, err = wire.WriteMessageN(n.Conn, sendAddrMsg, cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Printf("[%s]: failed to write sendaddrv2: %v\n", a, err)
		return
	}
	log.Printf("[%s]: OK. sent %d bytes\n", a, cnt)

	// 3. send verAck
	if n.Conn == nil {
		log.Printf("[%s]: disconnected\n", a)
		return
	}
	log.Printf("[%s]: sending verack...\n", a)
	cnt, err = wire.WriteMessageN(n.Conn, wire.NewMsgVerAck(), cfg.Pver, cfg.Btcnet)
	if err != nil {
		log.Printf("[%s]: failed to write verack: %v\n", a, err)
		return
	}
	log.Printf("[%s]: OK. sent %d bytes\n", a, cnt)

	// ====== NEGOTIATION DONE

	// ask for peers once
	if n.Conn == nil {
		log.Printf("[%s]: disconnected\n", a)
		return
	}
	time.Sleep(1 * time.Second)
	log.Printf("[%s]: sending getaddr...\n", a)
	msgAddr := wire.NewMsgGetAddr()
	cnt, err = wire.WriteMessageN(n.Conn, msgAddr, cfg.Pver, cfg.Btcnet)
	if err != nil {
		// Log and handle the error
		log.Printf("[%s]: failed to write getaddr: %v\n", a, err)
		return
	}
	log.Printf("[%s]: OK. sent %d bytes\n", a, cnt)

	// send pings
	time.Sleep(1 * time.Second)
	for {
		if n.Conn == nil {
			log.Printf("[%s]: disconnected\n", a)
			return
		}
		log.Printf("[%s]: sending ping...\n", a)
		nonceBig, _ := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(2, 62))))
		n.PingNonce = nonceBig.Uint64()
		msgPing := wire.NewMsgPing(n.PingNonce)
		cnt, err = wire.WriteMessageN(n.Conn, msgPing, cfg.Pver, cfg.Btcnet)
		if err != nil {
			log.Printf("[%s]: failed to write ping: %v\n", a, err)
			return
		}
		log.Printf("[%s]: OK. sent %d bytes\n", a, cnt)
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

func NodesRead(filename string) ([]*Node, error) {
	ret := make([]*Node, 0)
	// read from json
	fData, err := os.ReadFile(filename)
	if err != nil {
		return ret, err
	}
	var data []string
	err = json.Unmarshal(fData, &data)
	if err != nil {
		return ret, err
	}
	for _, addr := range data {
		a, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			log.Println("failed to resolve addr: ", addr)
			continue
		}
		ret = append(ret, &Node{Addr: *a})
	}

	return ret, nil
}

func SeedScan() ([]*Node, error) {
	nodes := make([]*Node, 0)
	fmt.Println("Getting nodes from dns seeds... via ", cfg.DnsAddress)
	now := time.Now()
	if cfg.DnsSeeds == nil {
		return nil, fmt.Errorf("no dns seeds")
	}
	for _, seed := range cfg.DnsSeeds {
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
				if node.Addr.IP.Equal(record.A) {
					continue
				}
			}
			n := Node{Addr: net.TCPAddr{IP: record.A, Port: int(cfg.NodesPort)}}
			nodes = append(nodes, &n)
		}
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("No nodes found")
	}
	fmt.Printf("Got %v nodes in %v\n", len(nodes), time.Since(now))

	// save nodes as json
	fData := make([]string, len(nodes))
	for i, n := range nodes {
		fData[i] = n.Endpoint()
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
