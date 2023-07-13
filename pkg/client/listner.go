package client

import (
	"fmt"
	"io"
	"log"

	"github.com/btcsuite/btcd/wire"
)

func (n *Node) connListen() {
	a := fmt.Sprintf("<- %s", n.Endpoint())
	defer func() {
		n.Conn = nil
		log.Printf("[%s]: closed\n", a)
	}()
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
				log.Printf("[%s]: EOF, exit\n", a)
				return
			}
			// Since the protocol version is 70016 but we don't
			// implement compact blocks, we have to ignore unknown
			// messages after the version-verack handshake. This
			// matches bitcoind's behavior and is necessary since
			// compact blocks negotiation occurs after the
			// handshake.
			if err == wire.ErrUnknownMessage {
				log.Printf("[%s]: ERR: unknown message, ignoring\n", a)
				continue
			}

			// log.Fatalf("Cant read buffer, error: %v\n", err)
			log.Printf("[%s]: ERR: Cant read buffer, error: %v\n", a, err)
			log.Printf("[%s]: ERR: bytes read: %v\n", a, cnt)
			log.Printf("[%s]: ERR: msg: %v\n", a, msg)
			log.Printf("[%s]: ERR: rawPayload: %v\n", a, rawPayload)
			continue
		}
		log.Printf("[%s]: Got message: %d bytes, cmd: %s rawPayload len: %d\n", a, cnt, msg.Command(), len(rawPayload))
		switch m := msg.(type) {
		case *wire.MsgVersion:
			log.Printf("[%s]: MsgVersion received\n", a)
			log.Printf("[%s]: version: %v\n", a, m.ProtocolVersion)
			log.Printf("[%s]: msg: %+v\n", a, m)
		case *wire.MsgVerAck:
			log.Printf("[%s]: MsgVerAck received\n", a)
			log.Printf("[%s]: msg: %+v\n", a, m)
		case *wire.MsgPing:
			log.Printf("[%s]: MsgPing received\n", a)
			log.Printf("[%s]: nonce: %v\n", a, m.Nonce)
			log.Printf("[%s]: msg: %+v\n", a, m)
		case *wire.MsgPong:
			log.Printf("[%s]: MsgPong received\n", a)
			if m.Nonce == n.PingNonce {
				log.Printf("[%s]: pong OK\n", a)
				n.PingCount++
				n.PingNonce = 0
			} else {
				log.Printf("[%s]: pong nonce mismatch, expected %v, got %v\n", a, n.PingNonce, m.Nonce)
			}
		case *wire.MsgAddr:
			log.Printf("[%s]: MsgAddr received\n", a)
			log.Printf("[%s]: got %d addresses\n", a, len(m.AddrList))
			batch := make([]string, len(m.AddrList))
			for i, a := range m.AddrList {
				batch[i] = fmt.Sprintf("%s:%d", a.IP.String(), a.Port)
			}
			newNodesCh <- batch
		case *wire.MsgAddrV2:
			log.Printf("[%s]: MsgAddrV2 received\n", a)
			log.Printf("[%s]: got %d addresses\n", a, len(m.AddrList))
			batch := make([]string, len(m.AddrList))
			for i, a := range m.AddrList {
				batch[i] = fmt.Sprintf("%s:%d", a.Addr.String(), a.Port)
			}
			newNodesCh <- batch

		case *wire.MsgInv:
			log.Printf("[%s]: MsgInv received\n", a)
			log.Printf("[%s]: data: %d\n", a, len(m.InvList))
			// TODO: answer on inv

		case *wire.MsgFeeFilter:
			log.Printf("[%s]: MsgFeeFilter received\n", a)
			log.Printf("[%s]: fee: %v\n", a, m.MinFee)
		case *wire.MsgGetHeaders:
			log.Printf("[%s]: MsgGetHeaders received\n", a)
			log.Printf("[%s]: headers: %d\n", a, len(m.BlockLocatorHashes))

		default:
			log.Printf("[%s]: unknown message received\n", a)
			log.Printf("[%s]: msg: %+v\n", a, m)
		}
	}
}
