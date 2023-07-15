// resolve seed nodes via dns
package dns

import (
	"fmt"
	"go-btc-downloader/pkg/config"
	"go-btc-downloader/pkg/logger"
	"time"

	"github.com/miekg/dns"
)

var cfg = config.New()

type DNS struct {
	log       *logger.Logger
	dnsSeeds  []string
	dnsServer string
	timeout   time.Duration
}

func New(log *logger.Logger) *DNS {
	// check config vars
	if cfg.DnsSeeds == nil || cfg.DnsAddress == "" || cfg.DnsTimeout == 0 {
		log.Fatal("dns config is not set")
	}
	return &DNS{
		log:       log,
		dnsSeeds:  cfg.DnsSeeds,
		dnsServer: cfg.DnsAddress,
		timeout:   cfg.DnsTimeout,
	}
}

func (d *DNS) Scan() ([]string, error) {
	// for {
	// 	d.log.Info("scanning dns seeds")
	// 	time.Sleep(1 * time.Second)
	// 	d.log.Debugf("dns seeds: %v", d.dnsSeeds)
	// 	time.Sleep(1 * time.Second)
	// 	d.log.Warn("scanning dns seeds test warning")
	// 	time.Sleep(1 * time.Second)
	// 	d.log.Error("scanning dns seeds test error")
	// 	time.Sleep(1 * time.Second)
	// }
	// return nil, nil
	ret := make([]string, 0)
	for _, seed := range d.dnsSeeds {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(seed), dns.TypeA)
		c := new(dns.Client)
		c.Net = "tcp"
		c.Timeout = cfg.DnsTimeout
		in, _, err := c.Exchange(m, d.dnsServer)
		if err != nil {
			return nil, fmt.Errorf("failed to get nodes from %v: %v", seed, err)
		}
		if len(in.Answer) == 0 {
			d.log.Warnf("no nodes found from %v", seed)
			continue
		}
		// loop through dns records
		for _, ans := range in.Answer {
			// check that record is valid
			if _, ok := ans.(*dns.A); !ok {
				continue
			}
			ret = append(ret, ans.(*dns.A).A.String())
		}
		d.log.Infof("got %v nodes from %v", len(in.Answer), seed)
	}
	return ret, nil
}
