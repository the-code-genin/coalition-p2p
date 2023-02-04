package coalition

import (
	"encoding/hex"
	"fmt"
)

// Send a ping to the host at the address
func (host *Host) Ping(address string) error {
	response, err := host.SendMessage(address, 1, PingMethod, nil)
	if err != nil {
		return err
	}
	data, ok := response.(string)
	if !ok {
		return fmt.Errorf("expected [%s] as response", PingResponse)
	} else if data != PingResponse {
		return fmt.Errorf("expected [%s] as response", PingResponse)
	}
	return nil
}

// Asks a peer for a list of nodes closest to a key on the network
func (host *Host) FindNode(address string, key []byte) ([]string, error) {
	response, err := host.SendMessage(
		address,
		1,
		FindNodeMethod,
		hex.EncodeToString(key),
	)
	if err != nil {
		return nil, err
	}

	data, ok := response.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected an array of node addresses as response")
	}
	addrs := make([]string, 0)
	for _, raw := range data {
		addr, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string")
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}
