package node

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/btcsuite/btcd/wire"
)

// listen to incoming messages
func (n *Node) listen(ctx context.Context) {
	a := fmt.Sprintf("◀︎ %s", n.Endpoint())
	ticker := time.NewTicker(cfg.ListenInterval)
	defer func() {
		// ensure to close the connection on exit
		if n.conn != nil {
			n.conn.Close()
		}
		n.status = disconnected
		n.log.Warnf("%s closed\n", a)
		ticker.Stop()
	}()
	// exit listener if no connection
	if n.conn == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// do not connect if no connection, will panic
			if n.conn == nil || n.status != connected {
				return
			}
			cnt, msg, rawPayload, err := wire.ReadMessageN(n.conn, cfg.Pver, cfg.Btcnet)
			// cnt, msg, rawPayload, err := wire.ReadMessageWithEncodingN(n.Conn, cfg.Pver, cfg.Btcnet, wire.BaseEncoding)
			if err != nil {
				if err == io.EOF {
					n.log.Warnf("%s EOF, exit\n", a)
					return
				}
				// Since the protocol version is 70016 but we don't
				// implement compact blocks, we have to ignore unknown
				// messages after the version-verack handshake. This
				// matches bitcoind's behavior and is necessary since
				// compact blocks negotiation occurs after the
				// handshake.
				if err == wire.ErrUnknownMessage {
					n.log.Warnf("%s ERR: unknown message, ignoring\n", a)
					continue
				}

				// log.Fatalf("Cant read buffer, error: %v\n", err)
				n.log.Warnf("%s ERR: Cant read buffer, error: %v\n", a, err)
				n.log.Warnf("%s ERR: bytes read: %v\n", a, cnt)
				n.log.Warnf("%s ERR: msg: %v\n", a, msg)
				n.log.Warnf("%s ERR: rawPayload: %v\n", a, rawPayload)
				continue
			}
			n.log.Debugf("%s Got message: %d bytes, cmd: %s rawPayload len: %d\n", a, cnt, msg.Command(), len(rawPayload))
			switch m := msg.(type) {
			case *wire.MsgVersion:
				n.log.Infof("%s MsgVersion received\n", a)
				n.log.Debugf("%s version: %v\n", a, m.ProtocolVersion)
				n.log.Debugf("%s msg: %+v\n", a, m)
				n.version = m.ProtocolVersion

			case *wire.MsgVerAck:
				n.log.Infof("%s MsgVerAck received\n", a)
				n.log.Debugf("%s msg: %+v\n", a, m)

			case *wire.MsgPing:
				n.log.Infof("%s MsgPing received\n", a)
				n.log.Debugf("%s nonce: %v\n", a, m.Nonce)
				n.log.Debugf("%s msg: %+v\n", a, m)

			case *wire.MsgPong:
				n.log.Infof("%s MsgPong received\n", a)
				if m.Nonce == n.pingNonce {
					n.log.Debugf("%s pong OK\n", a)
					n.pongCount++
					n.UpdatePingNonce()
				} else {
					n.log.Warnf("%s pong nonce mismatch, expected %v, got %v\n", a, n.pingNonce, m.Nonce)
				}

			case *wire.MsgAddr:
				n.log.Infof("%s MsgAddr received\n", a)
				n.log.Debugf("%s got %d addresses\n", a, len(m.AddrList))
				batch := make([]string, len(m.AddrList))
				for i, a := range m.AddrList {
					batch[i] = fmt.Sprintf("[%s]:%d", a.IP.String(), a.Port)
				}
				n.newAddrCh <- batch
				n.Disconnect()

			case *wire.MsgAddrV2:
				n.log.Infof("%s MsgAddrV2 received\n", a)
				n.log.Debugf("%s got %d addresses\n", a, len(m.AddrList))
				batch := make([]string, len(m.AddrList))
				for i, a := range m.AddrList {
					batch[i] = a.Addr.String()
				}
				n.newAddrCh <- batch
				n.Disconnect()

			case *wire.MsgInv:
				n.log.Infof("%s MsgInv received\n", a)
				n.log.Debugf("%s data: %d\n", a, len(m.InvList))
				// TODO: answer on inv

			case *wire.MsgFeeFilter:
				n.log.Infof("%s MsgFeeFilter received\n", a)
				n.log.Debugf("%s fee: %v\n", a, m.MinFee)

			case *wire.MsgGetHeaders:
				n.log.Infof("%s MsgGetHeaders received\n", a)
				n.log.Debugf("%s headers: %d\n", a, len(m.BlockLocatorHashes))

			default:
				n.log.Infof("%s (%T) message received (unhandled)\n", a, m)
				n.log.Debugf("%s msg: %+v\n", a, m)
			}
		}
	}
}
