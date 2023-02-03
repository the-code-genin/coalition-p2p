package coalition

import (
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
)

// Format the node details into a node address
func FormatNodeAddress(key []byte, ip4Address string, port int) string {
	return fmt.Sprintf(
		"node://%s@%s:%d",
		hex.EncodeToString(key),
		ip4Address,
		port,
	)
}

// Parse a node address(node://) into (peer key, ip4Address, port)
func ParseNodeAddress(address string) ([]byte, string, int, error) {
	re, err := regexp.Compile(`^node\:\/\/([0-9A-f]+)\@(.+)\:(\d+)$`)
	if err != nil {
		return nil, "", 0, err
	} else if !re.Match([]byte(address)) {
		return nil, "", 0, fmt.Errorf("invalid node address")
	}

	res := re.FindStringSubmatch(address)
	if len(res) != 4 {
		return nil, "", 0, fmt.Errorf("invalid node address")
	}

	key, err := hex.DecodeString(res[1])
	if err != nil {
		return nil, "", 0, err
	}

	ipAddress := net.ParseIP(res[2])
	if ipAddress == nil {
		return nil, "", 0, fmt.Errorf("invalid ip adddress")
	}
	ip4Address := ipAddress.To4()
	if ip4Address == nil {
		return nil, "", 0, fmt.Errorf("invalid ip4 address")
	}

	port, err := strconv.Atoi(res[3])
	if err != nil {
		return nil, "", 0, err
	}

	return key, ip4Address.String(), port, nil
}
