package framework

import (
	"net"
)

// Get the computer's network IPs
func NetworkIPs() ([]net.IP, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ips := make([]net.IP, 0)
	for _, netInterface := range netInterfaces {
		if netInterface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if netInterface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		addrs, err := netInterface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}

			exists := false
			for _, existingIp := range ips {
				if net.IP.Equal(existingIp, ip) {
					exists = true
					break
				}
			}
			if !exists {
				ips = append(ips, ip)
			}
		}
	}

	return ips, nil
}

// Get the computer's network IPv4 addresses
func NetworkIPv4Addresses() ([]string, error) {
	ips, err := NetworkIPs()
	if err != nil {
		return nil, err
	}

	addresses := make([]string, 0)
	for _, ip := range ips {
		ipv4 := ip.To4()
		if ipv4 == nil {
			continue
		}
		addresses = append(addresses, ipv4.String())
	}

	return addresses, nil
}
