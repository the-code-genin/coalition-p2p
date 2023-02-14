package coalition

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestNodeAddressParsing(t *testing.T) {
	key := make([]byte, PeerKeySize)
	if _, err := rand.Read(key); err != nil {
		t.Error(err)
	}
	ip4 := "0.0.0.0"
	port := 3000

	nodeAddress, err := FormatNodeAddress(key, ip4, port)
	if err != nil {
		t.Error(err)
	}

	parsedKey, parsedIP4, parsedPort, err := ParseNodeAddress(nodeAddress)
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(parsedKey, key) {
		t.Errorf("expected %s to match %s", parsedKey, key)
	} else if ip4 != parsedIP4 {
		t.Errorf("expected %s to match %s", ip4, parsedIP4)
	} else if port != parsedPort {
		t.Errorf("expected %d to match %d", port, parsedPort)
	}
}
