package client

import (
	"fmt"
	"io"
	"log"

	"github.com/btcsuite/btcd/wire"
)

func (n *Node) connListen() {
	// buf := make([]byte, 65536)
	// bufReader := bufio.NewReader(n.Conn)
	for {
		fmt.Println()
		// cnt, err := bufReader.Read(buf)
		// if err != nil {
		// 	log.Fatalf("failed to read from conn: %v", err)
		// 	return
		// }
		// log.Printf("[listner]: raw bytes read: %v\n", cnt)
		// data := buf[:cnt]
		// log.Printf("[listner]: raw buf: %v\n", data)
		// log.Printf("[listner]: raw buf: %v\n", string(data))
		cnt, msg, rawPayload, err := wire.ReadMessageN(n.Conn, cfg.Pver, cfg.Btcnet)
		// cnt, msg, rawPayload, err := wire.ReadMessageWithEncodingN(n.Conn, cfg.Pver, cfg.Btcnet, wire.BaseEncoding)
		if err != nil {
			if err == io.EOF {
				log.Fatal("[listner]: EOF, exit")
				return
			}
			// Since the protocol version is 70016 but we don't
			// implement compact blocks, we have to ignore unknown
			// messages after the version-verack handshake. This
			// matches bitcoind's behavior and is necessary since
			// compact blocks negotiation occurs after the
			// handshake.
			if err == wire.ErrUnknownMessage {
				log.Println("[listner]: ERR: unknown message, ignoring")
				continue
			}

			// log.Fatalf("Cant read buffer, error: %v\n", err)
			log.Printf("[listner]: ERR: Cant read buffer, error: %v\n", err)
			log.Printf("[listner]: ERR: bytes read: %v\n", cnt)
			log.Printf("[listner]: ERR: msg: %v\n", msg)
			log.Printf("[listner]: ERR: rawPayload: %v\n", rawPayload)
			continue
		}
		log.Printf("[listner]: Got message: %d bytes, cmd: %s rawPayload len: %d\n", cnt, msg.Command(), len(rawPayload))
		switch m := msg.(type) {
		case *wire.MsgVersion:
			log.Printf("[listner]: MsgVersion received from %v\n", n.Address.String())
			log.Printf("[listner]: version: %v\n", m.ProtocolVersion)
			log.Printf("[listner]: msg: %+v\n", m)
		case *wire.MsgVerAck:
			log.Printf("[listner]: MsgVerAck received from %v\n", n.Address.String())
			log.Printf("[listner]: msg: %+v\n", m)
		case *wire.MsgPing:
			log.Printf("[listner]: MsgPing received from %v\n", n.Address.String())
			log.Printf("[listner]: nonce: %v\n", m.Nonce)
			log.Printf("[listner]: msg: %+v\n", m)
		case *wire.MsgPong:
			log.Printf("[listner]: MsgPong received from %v\n", n.Address.String())
			log.Printf("[listner]: nonce: %v\n", m.Nonce)
			log.Printf("[listner]: msg: %+v\n", m)
		case *wire.MsgAddrV2:
			log.Printf("[listner]: MsgAddrV2 received from %v\n", n.Address.String())
			// get list of addresses
			// for _, addr := range m.AddrList {
			// 	log.Printf("[listner]: addr: %+v\n", addr)
			// }
			log.Printf("[listner]: got %d addresses\n", len(m.AddrList))
		case *wire.MsgInv:
			log.Printf("[listner]: MsgInv received from %v\n", n.Address.String())
			log.Printf("[listner]: data: %+v\n", m.InvList)
		case *wire.MsgFeeFilter:
			log.Printf("[listner]: MsgFeeFilter received from %v\n", n.Address.String())
			log.Printf("[listner]: fee: %v\n", m.MinFee)
		case *wire.MsgGetHeaders:
			log.Printf("[listner]: MsgGetHeaders received from %v\n", n.Address.String())
			log.Printf("[listner]: headers: %+v\n", m.BlockLocatorHashes)

		default:
			log.Printf("[listner]: unknown message received from %v\n", n.Address.String())
			log.Printf("[listner]: msg: %+v\n", m)
		}
	}
}
