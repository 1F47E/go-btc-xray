package client

import (
	"fmt"
	"io"
	"log"

	"github.com/btcsuite/btcd/wire"
)

func (n *Node) connListen() {
	a := fmt.Sprintf("%s:%d <<", n.Address.String(), cfg.NodesPort)
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
			log.Printf("[%s]: MsgVersion received from %v\n", a, n.Address.String())
			log.Printf("[%s]: version: %v\n", a, m.ProtocolVersion)
			log.Printf("[%s]: msg: %+v\n", a, m)
		case *wire.MsgVerAck:
			log.Printf("[%s]: MsgVerAck received from %v\n", a, n.Address.String())
			log.Printf("[%s]: msg: %+v\n", a, m)
		case *wire.MsgPing:
			log.Printf("[%s]: MsgPing received from %v\n", a, n.Address.String())
			log.Printf("[%s]: nonce: %v\n", a, m.Nonce)
			log.Printf("[%s]: msg: %+v\n", a, m)
		case *wire.MsgPong:
			log.Printf("[%s]: MsgPong received from %v\n", a, n.Address.String())
			log.Printf("[%s]: nonce: %v\n", a, m.Nonce)
			log.Printf("[%s]: msg: %+v\n", a, m)
		case *wire.MsgAddr:
			log.Printf("[%s]: MsgAddr received from %v\n", a, n.Address.String())
			log.Printf("[%s]: got %d addresses\n", a, len(m.AddrList))
			for _, a := range m.AddrList {
				addrStr := fmt.Sprintf("%v:%v", a.IP.String(), a.Port)
				n.AddPeer(addrStr)
			}
		case *wire.MsgAddrV2:
			log.Printf("[%s]: MsgAddrV2 received from %v\n", a, n.Address.String())
			// get list of addresses
			// for _, addr := range m.AddrList {
			// 	log.Printf("[listner]: addr: %+v\n", addr)
			// }
			log.Printf("[%s]: got %d addresses\n", a, len(m.AddrList))
			for _, a := range m.AddrList {
				addrStr := fmt.Sprintf("%v:%v", a.Addr.String(), a.Port)
				n.AddPeer(addrStr)
			}
			// send ack
			// mAck := wire.NewMsgSendAddrV2()
			// mAck.(*wire.MsgSendAddrV2).AddrList = make([]*wire.NetAddressV2, 0)
			// mAck.AddrList = make([]*wire.NetAddressV2, 0)
			// mAck.AddrList = append(m.AddrList, m.AddrList...)

		case *wire.MsgInv:
			log.Printf("[%s]: MsgInv received from %v\n", a, n.Address.String())
			log.Printf("[%s]: data: %d\n", a, len(m.InvList))
			// TODO: answer on inv

		case *wire.MsgFeeFilter:
			log.Printf("[%s]: MsgFeeFilter received from %v\n", a, n.Address.String())
			log.Printf("[%s]: fee: %v\n", a, m.MinFee)
		case *wire.MsgGetHeaders:
			log.Printf("[%s]: MsgGetHeaders received from %v\n", a, n.Address.String())
			log.Printf("[%s]: headers: %d\n", a, len(m.BlockLocatorHashes))

		default:
			log.Printf("[%s]: unknown message received from %v\n", a, n.Address.String())
			log.Printf("[%s]: msg: %+v\n", a, m)
		}
	}
}
