package test

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"testing"
)

func TestNet(t *testing.T) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP
		if !ip.IsLoopback() && ip.To4() != nil {
			// 检查IP是否是私有地址范围
			if isPrivateIP(ip) {
				fmt.Println("本机私有IPv4地址:", ip.String())
			}
		}
	}
}

// isPrivateIP 检查一个IPv4地址是否是私有地址
func isPrivateIP(ip net.IP) bool {
	// RFC 1918私有地址范围
	privateRanges := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
	}

	// 检查IP是否在私有地址范围内
	ipAddr, err := netip.ParseAddr(ip.String())
	if err != nil {
		return false
	}

	for _, prefix := range privateRanges {
		if prefix.Contains(ipAddr) {
			return true
		}
	}
	return false
}
