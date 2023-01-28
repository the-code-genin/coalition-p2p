package coalition

import (
	"fmt"
)

func XORBytes(partA, partB []byte) ([]byte, error) {
	if len(partA) != len(partB) {
		return nil, fmt.Errorf("byte lengths must match")
	}
	res := make([]byte, 0)
	for i := 0; i < len(partA); i++ {
		res = append(res, partA[i]^partB[i])
	}
	return res, nil
}
