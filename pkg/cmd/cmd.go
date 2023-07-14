package cmd

import (
	"fmt"
	"go-btc-downloader/pkg/config"
	"net"

	"github.com/btcsuite/btcd/wire"
)

var cfg = config.New()

func SendVersion(conn net.Conn, nonce uint64) error {
	msg := localVersionMsg(nonce)
	return writeMessage(conn, msg)
}

func SendAddrV2(conn net.Conn) error {
	msg := wire.NewMsgSendAddrV2()
	return writeMessage(conn, msg)
}

func SendVerAck(conn net.Conn) error {
	return writeMessage(conn, wire.NewMsgVerAck())
}

func SendGetAddr(conn net.Conn) error {
	msg := wire.NewMsgGetAddr()
	return writeMessage(conn, msg)
}

func SendPing(conn net.Conn, nonce uint64) error {
	msg := wire.NewMsgPing(nonce)
	return writeMessage(conn, msg)
}

func writeMessage(conn net.Conn, msg wire.Message) error {
	if conn == nil {
		return fmt.Errorf("no connection")
	}
	return wire.WriteMessage(conn, msg, cfg.Pver, cfg.Btcnet)
}

// localVersionMsg creates a version message that can be used to send to the
// remote peer.
func localVersionMsg(nonce uint64) *wire.MsgVersion {
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

	// Version message.
	msg := wire.NewMsgVersion(ourNA, &theirNA, nonce, blockNum)
	_ = msg.AddUserAgent("btcd", "0.23.3", "")
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(cfg.Pver)
	// Advertise if inv messages for transactions are desired.
	// msg.DisableRelayTx = p.cfg.DisableRelayTx

	return msg
}

/*
"The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
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
// send genesis block
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
