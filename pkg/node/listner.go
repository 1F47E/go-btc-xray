package node

import (
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"

	"github.com/btcsuite/btcd/wire"
)

// listen to incoming messages
func (n *Node) listen() {
	a := fmt.Sprintf("◀︎ %s:", n.Endpoint())
	defer func() {
		n.conn = nil
		log.Warnf("%s closed\n", a)
	}()
	for {
		// exit on closed connection
		if n.conn == nil {
			return
		}
		fmt.Println()
		cnt, msg, rawPayload, err := wire.ReadMessageN(n.conn, cfg.Pver, cfg.Btcnet)
		// cnt, msg, rawPayload, err := wire.ReadMessageWithEncodingN(n.Conn, cfg.Pver, cfg.Btcnet, wire.BaseEncoding)
		if err != nil {
			if err == io.EOF {
				log.Warnf("%s EOF, exit\n", a)
				return
			}
			// Since the protocol version is 70016 but we don't
			// implement compact blocks, we have to ignore unknown
			// messages after the version-verack handshake. This
			// matches bitcoind's behavior and is necessary since
			// compact blocks negotiation occurs after the
			// handshake.
			if err == wire.ErrUnknownMessage {
				log.Warnf("%s ERR: unknown message, ignoring\n", a)
				continue
			}

			// log.Fatalf("Cant read buffer, error: %v\n", err)
			log.Warnf("%s ERR: Cant read buffer, error: %v\n", a, err)
			log.Warnf("%s ERR: bytes read: %v\n", a, cnt)
			log.Warnf("%s ERR: msg: %v\n", a, msg)
			log.Warnf("%s ERR: rawPayload: %v\n", a, rawPayload)
			continue
		}
		log.Debugf("%s Got message: %d bytes, cmd: %s rawPayload len: %d\n", a, cnt, msg.Command(), len(rawPayload))
		switch m := msg.(type) {
		case *wire.MsgVersion:
			log.Infof("%s MsgVersion received\n", a)
			log.Debugf("%s version: %v\n", a, m.ProtocolVersion)
			log.Debugf("%s msg: %+v\n", a, m)
		case *wire.MsgVerAck:
			log.Infof("%s MsgVerAck received\n", a)
			log.Debugf("%s msg: %+v\n", a, m)
		case *wire.MsgPing:
			log.Infof("%s MsgPing received\n", a)
			log.Debugf("%s nonce: %v\n", a, m.Nonce)
			log.Debugf("%s msg: %+v\n", a, m)
		case *wire.MsgPong:
			log.Infof("%s MsgPong received\n", a)
			if m.Nonce == n.pingNonce {
				log.Debugf("%s pong OK\n", a)
				n.pingCount++
				n.pingNonce = 0
			} else {
				log.Warnf("%s pong nonce mismatch, expected %v, got %v\n", a, n.pingNonce, m.Nonce)
			}
		case *wire.MsgAddr:
			log.Infof("%s MsgAddr received\n", a)
			log.Debugf("%s got %d addresses\n", a, len(m.AddrList))
			batch := make([]string, len(m.AddrList))
			for i, a := range m.AddrList {
				batch[i] = fmt.Sprintf("[%s]:%d", a.IP.String(), a.Port)
			}
			n.newAddrCh <- batch
		case *wire.MsgAddrV2:
			log.Infof("%s MsgAddrV2 received\n", a)
			log.Debugf("%s got %d addresses\n", a, len(m.AddrList))
			batch := make([]string, len(m.AddrList))
			for i, a := range m.AddrList {
				batch[i] = fmt.Sprintf("[%s]:%d", a.Addr.String(), a.Port)
			}
			n.newAddrCh <- batch

		case *wire.MsgInv:
			log.Infof("%s MsgInv received\n", a)
			log.Debugf("%s data: %d\n", a, len(m.InvList))
			// TODO: answer on inv

		case *wire.MsgFeeFilter:
			log.Infof("%s MsgFeeFilter received\n", a)
			log.Debugf("%s fee: %v\n", a, m.MinFee)
		case *wire.MsgGetHeaders:
			log.Infof("%s MsgGetHeaders received\n", a)
			log.Debugf("%s headers: %d\n", a, len(m.BlockLocatorHashes))

		default:
			log.Infof("%s (%T) message received (unhandled)\n", a, m)
			log.Debugf("%s msg: %+v\n", a, m)
		}
	}
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
	n.pingNonce = nonceBig.Uint64()

	// Version message.
	msg := wire.NewMsgVersion(ourNA, &theirNA, n.pingNonce, blockNum)
	_ = msg.AddUserAgent("btcd", "0.23.3", "")
	msg.Services = wire.SFNodeNetwork
	msg.ProtocolVersion = int32(cfg.Pver)
	// Advertise if inv messages for transactions are desired.
	// msg.DisableRelayTx = p.cfg.DisableRelayTx

	return msg, nil
}
