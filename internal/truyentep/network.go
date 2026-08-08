package truyentep

import (
	"net"
	"strings"
)

func interfaceAddrs() ([]net.Addr, error) {
	return net.InterfaceAddrs()
}

func addressIP(addr net.Addr) string {
	var raw string
	switch value := addr.(type) {
	case *net.IPNet:
		raw = value.IP.String()
	case *net.IPAddr:
		raw = value.IP.String()
	default:
		raw = strings.Split(addr.String(), "/")[0]
	}
	ip := net.ParseIP(raw)
	if ip == nil || ip.To4() == nil || ip.IsLoopback() || !isTrustedLANIP(ip) {
		return ""
	}
	return ip.String()
}

func isTrustedLANIP(ip net.IP) bool {
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

func requestIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	host = strings.Trim(host, "[]")
	return net.ParseIP(host)
}
