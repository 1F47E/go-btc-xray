// resolve seed nodes via dns
package dns

import (
	"time"

	"github.com/1F47E/go-btc-xray/internal/config"
	"github.com/1F47E/go-btc-xray/internal/logger"

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

func (d *DNS) Scan() []string {
	ips := make(map[string]struct{}, 0)
	c := new(dns.Client)
	m := new(dns.Msg)
	c.Net = "tcp"
	for _, seed := range d.dnsSeeds {
		d.log.Infof("[DNS]:[%s] asking for nodes\n", seed)
		c.Timeout = cfg.DnsTimeout
		m.SetQuestion(dns.Fqdn(seed), dns.TypeA)
		in, _, err := c.Exchange(m, d.dnsServer)
		if err != nil {
			d.log.Warnf("[DNS]:[%s] error %v\n", seed, err)
			continue
		}
		if len(in.Answer) == 0 {
			d.log.Warnf("[DNS]:[%s] no nodes found\n", seed)
			continue
		}
		// loop through dns records
		new := 0
		for _, ans := range in.Answer {
			// check that record is valid
			if _, ok := ans.(*dns.A); !ok {
				d.log.Warnf("[DNS]:[%s] invalid dns record, skipping\n", seed)
				continue
			}
			// only add new ones
			ip := ans.(*dns.A).A.String()
			if _, ok := ips[ip]; ok {
				d.log.Debugf("[DNS]:[%s] got duplicate ip %v\n", seed, ip)
				continue
			}
			ips[ip] = struct{}{}
			new++
		}
		if new > 0 {
			d.log.Infof("[DNS]:[%s] found %d new nodes\n", seed, new)
		} else {
			d.log.Debugf("[DNS]:[%s] no new nodes\n", seed)
		}
	}
	d.log.Infof("[DNS]: finished scan. Got %d nodes from %d seeds\n", len(ips), len(d.dnsSeeds))
	ret := make([]string, 0, len(ips))
	for ip := range ips {
		ret = append(ret, ip)
	}
	return ret
}
