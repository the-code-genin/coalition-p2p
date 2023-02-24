package coalition

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"regexp"
	"strconv"
	"time"
)

// Format the node details into a node address
func FormatNodeAddress(key []byte, addr string, port int) (string, error) {
	if len(key) != PeerKeySize {
		return "", fmt.Errorf("invalid peer key size")
	}

	ipAddress := net.ParseIP(addr)
	if ipAddress == nil {
		return "", fmt.Errorf("invalid ip adddress")
	}
	ip4Address := ipAddress.To4()
	if ip4Address == nil {
		return "", fmt.Errorf("invalid ip4 address")
	}

	nodeAddr := fmt.Sprintf(
		"node://%s@%s:%d",
		hex.EncodeToString(key),
		ip4Address.String(),
		port,
	)
	return nodeAddr, nil
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
	} else if len(key) != PeerKeySize {
		return nil, "", 0, fmt.Errorf("invalid peer key")
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

// Get this computer's public ip4 addresses
func GetPublicIP4Addresses() ([]string, error) {
	infaces, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}

	res := make([]string, 0)
	for _, inface := range infaces {
		addrs, err := inface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			res = append(res, ip4.String())
		}
	}
	return res, nil
}

// Do a merge sort on two Peer arrays
func MergeSortPeers(
	bucketA, bucketB []*Peer,
	sortFunc func(*Peer, *Peer) int,
) []*Peer {
	output := make([]*Peer, 0)

	// Atomic sub array
	if len(bucketA) == 0 && len(bucketB) == 0 {
		return output
	} else if len(bucketA) == 1 && len(bucketB) == 0 {
		return bucketA
	} else if len(bucketB) == 1 && len(bucketA) == 0 {
		return bucketB
	}

	// Sort bucketA
	midPointA := int(math.Ceil(float64(len(bucketA)) / 2))
	sortedA := MergeSortPeers(bucketA[:midPointA], bucketA[midPointA:], sortFunc)

	// Sort bucketB
	midPointB := int(math.Ceil(float64(len(bucketB)) / 2))
	sortedB := MergeSortPeers(bucketB[:midPointB], bucketB[midPointB:], sortFunc)

	// Merge arrays
	for i, j := 0, 0; i < len(sortedA) || j < len(sortedB); {
		var peerA, peerB *Peer
		if i < len(sortedA) {
			peerA = sortedA[i]
		}
		if j < len(sortedB) {
			peerB = sortedB[j]
		}

		// If either array has been exhausted
		if peerA == nil {
			output = append(output, peerB)
			j++
			continue
		} else if peerB == nil {
			output = append(output, peerA)
			i++
			continue
		}

		// Compare the Peers
		if res := sortFunc(peerA, peerB); res >= 0 {
			// PeerB is greater than or equal to peerA
			output = append(output, peerA)
			i++
		} else {
			// PeerA is greater than peerB
			output = append(output, peerB)
			j++
		}
	}

	return output
}

// Do a merge sort for peers from closest to farthest from a search key
func SortPeersByClosest(peers []*Peer, searchKey []byte) []*Peer {
	return MergeSortPeers(
		peers,
		make([]*Peer, 0),
		func(peerA, peerB *Peer) int {
			distanceA := XORBytes(peerA.Key(), searchKey)
			distanceB := XORBytes(peerB.Key(), searchKey)
			return bytes.Compare(distanceB, distanceA)
		},
	)
}

// Writes a payload to the connection
func WriteToConn(conn net.Conn, data []byte) error {
	conn.SetWriteDeadline(time.Now().Add(TCPIODeadline))
	payloadSize := uint64(len(data))
	if payloadSize > TCPIOBufferSize {
		return fmt.Errorf("request payload exceeds buffer size")
	}

	if _, err := conn.Write(Uint64ToBytes(payloadSize)); err != nil {
		return err
	}
	if _, err := conn.Write(data); err != nil {
		return err
	}
	return nil
}

// Reads a payload from the connection
func ReadFromConn(conn net.Conn) ([]byte, error) {
	conn.SetReadDeadline(time.Now().Add(TCPIODeadline))
	requestReader := bufio.NewReader(conn)

	// Parse the size of the request payload in bytes
	payloadSizeBuffer := make([]byte, 8)
	_, err := io.ReadFull(requestReader, payloadSizeBuffer)
	if err != nil {
		return nil, err
	}
	payloadSize := BytesToUint64(payloadSizeBuffer)
	if payloadSize > TCPIOBufferSize {
		return nil, fmt.Errorf("request payload exceeds buffer size")
	}

	// Read the payload from the connection
	payload := make([]byte, payloadSize)
	_, err = io.ReadFull(requestReader, payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// Recovers the peer key from a peer signature
func RecoverPeerKeyFromPeerSignature(signature, hash []byte) ([]byte, error) {
	if len(signature) != PeerSignatureSize {
		return nil, fmt.Errorf("invalid peer signature length")
	}
	publicKey := signature[:ed25519.PublicKeySize]
	ecSignature := signature[ed25519.PublicKeySize:]

	// Verify the peer signature
	if !ed25519.Verify(publicKey, hash, ecSignature) {
		return nil, fmt.Errorf("invalid peer signature")
	}

	peerKey := sha1.Sum(publicKey)
	return peerKey[:], nil
}

// Converts a 64 bit integer to bytes
func Uint64ToBytes(data uint64) []byte {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, data)
	return buffer
}

// Converts bytes into a 64 bit integer
func BytesToUint64(data []byte) uint64 {
	return binary.BigEndian.Uint64(data)
}

// Find the XOR operation output of two slices.
// Length of output is min(len(sliceA), len(sliceB))
func XORBytes(sliceA, sliceB []byte) []byte {
	output := make([]byte, 0)
	for i := 0; i < len(sliceA) && i < len(sliceB); i++ {
		output = append(output, sliceA[i]^sliceB[i])
	}
	return output
}

// Find the AND operation output of two slices.
// Length of output is min(len(sliceA), len(sliceB))
func ANDBytes(sliceA, sliceB []byte) []byte {
	output := make([]byte, 0)
	for i := 0; i < len(sliceA) && i < len(sliceB); i++ {
		output = append(output, sliceA[i]&sliceB[i])
	}
	return output
}
