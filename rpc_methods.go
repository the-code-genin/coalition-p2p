package coalition

import "fmt"

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
