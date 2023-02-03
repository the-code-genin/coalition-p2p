package coalition

import "crypto/ed25519"

// Size of peer key in bytes
const PeerKeySize = 20

// Size of peer payload signature in bytes
const PeerSignatureSize = ed25519.PublicKeySize + ed25519.SignatureSize
