// resolve seed nodes via dns
package dns

import (
	"fmt"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/logger"

	"github.com/miekg/dns"
)

var cfg = config.New()
var log *logger.Logger = logger.New()

func Scan() ([]string, error) {
	ret := make([]string, 0)
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
			return nil, fmt.Errorf("failed to get nodes from %v: %v", seed, err)
		}
		if len(in.Answer) == 0 {
			log.Warnf("no nodes found from %v", seed)
			continue
		}
		// loop through dns records
		for _, ans := range in.Answer {
			// check that record is valid
			if _, ok := ans.(*dns.A); !ok {
				continue
			}
			addr := fmt.Sprintf("[%s]:%d", ans.(*dns.A).A.String(), cfg.NodesPort)
			ret = append(ret, addr)
		}
		log.Infof("got %v nodes from %v", len(in.Answer), seed)
	}
	return ret, nil
}
