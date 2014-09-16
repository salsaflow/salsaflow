package uuid

import (
	"crypto/rand"
	"encoding/hex"
)

const UuidHexLength = 10

func New() (string, error) {
	// Get some random data.
	data := make([]byte, UuidHexLength/2)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}

	// Convert to hex string and return.
	return hex.EncodeToString(data), nil
}
