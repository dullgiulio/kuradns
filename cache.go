package main

import "net"

type ipentry struct {
	ip  net.IP
	err error
}

type ipcache map[string]ipentry

func makeIPCache() ipcache {
	return make(map[string]ipentry)
}

func (c ipcache) lookup(host string) (net.IP, error) {
	if res, ok := c[host]; ok {
		return res.ip, res.err
	}
	var ip net.IP
	iplist, err := net.LookupIP(host)
	if err == nil {
		ip = iplist[0]
	}
	c[host] = ipentry{ip, err}
	return ip, err
}
