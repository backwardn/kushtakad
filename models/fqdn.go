package models

import (
	"net"
	"time"

	externalip "github.com/glendc/go-external-ip"
)

type FQDN struct {
	Port80  *Port80
	Port443 *Port443
	ARecord *ARecord
	IPMatch *IPMatch

	ExternalIP net.IP
}

type Port80 struct {
	Test bool
	Err  error
}

type Port443 struct {
	Test bool
	Err  error
}

type ARecord struct {
	Test bool
	Err  error
}

type IPMatch struct {
	Test bool
	Err  error
}

func NewFQDN() *FQDN {
	fqdn := &FQDN{
		Port80:  &Port80{},
		Port443: &Port443{},
		ARecord: &ARecord{},
		IPMatch: &IPMatch{},
	}
	return fqdn
}

func (fqdn *FQDN) Test(domain string) {
	fqdn.BuildExternalIP()
	fqdn.Port80.Test, fqdn.Port80.Err = fqdn.TestPort80()
	fqdn.Port443.Test, fqdn.Port443.Err = fqdn.TestPort443()
	fqdn.ARecord.Test, fqdn.ARecord.Err = fqdn.TestARecord(domain)
	//fqdn.IPMatch.Test , fqdn.IPMatch.Err = fqdn.TestIP()
	fqdn.IPMatch.Test = true
	fqdn.IPMatch.Err = nil
}

func (fqdn *FQDN) BuildExternalIP() {
	cfg := &externalip.ConsensusConfig{
		Timeout: time.Second * 3,
	}
	consensus := externalip.DefaultConsensus(cfg, nil)
	fqdn.ExternalIP, _ = consensus.ExternalIP()
}

func (fqdn *FQDN) TestPort80() (bool, error) {
	conn, err := net.Listen("tcp", ":80")
	if err != nil {
		return false, err
	}
	conn.Close()

	return true, nil
}

func (fqdn *FQDN) TestPort443() (bool, error) {
	conn, err := net.Listen("tcp", ":443")
	if err != nil {
		return false, err
	}
	conn.Close()

	return true, nil
}

// Get preferred outbound ip of this machine
func (fqdn *FQDN) GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

func (fqdn *FQDN) TestARecord(domain string) (bool, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return false, err
	}

	for _, ip := range ips {
		log.Debugf("%s. IN A %s\n", domain, ip.String())
		if fqdn.ExternalIP.Equal(ip) {
			return true, nil
		}
	}

	return false, nil
}

func (fqdn *FQDN) TestIP() (bool, error) {
	if fqdn.ExternalIP == nil {
		consensus := externalip.DefaultConsensus(nil, nil)
		fqdn.ExternalIP, _ = consensus.ExternalIP()
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return false, err
	}
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		// handle err
		if err != nil {
			return false, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			log.Debugf("Outbound: %s || Interal %s", fqdn.ExternalIP.String(), ip.String())
			if fqdn.ExternalIP.Equal(ip) {
				return true, nil
			}
		}
	}

	return false, nil
}
