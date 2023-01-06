package framework

import "testing"

func TestNewHost(t *testing.T) {
	host, err := NewHost(300)
	if err != nil {
		t.Error(err)
	}
	
	if err = host.Close(); err != nil {
		t.Error(err)
	}
}