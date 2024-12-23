package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

func genHMACKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func CheckHMACwithUserID(hmacKey []byte, userID string, code string, expectedHMAC string) (bool, error) {
	// Decode the hex-encoded HMAC
	expectedHMACBytes, err := hex.DecodeString(expectedHMAC)
	if err != nil {
		return false, fmt.Errorf("failed to decode HMAC hex string: %w", err)
	}

	// Create a new HMAC-SHA256 instance
	mac := hmac.New(sha256.New, hmacKey)

	mac.Write([]byte(userID))
	mac.Write([]byte(code))

	// Calculate the HMAC
	calculatedHMAC := mac.Sum(nil)

	// Compare the HMACs using a constant-time comparison
	return hmac.Equal(calculatedHMAC, expectedHMACBytes), nil
}

func CheckHMAC(hmacKey []byte, code string, expectedHMAC string) (bool, error) {
	// Decode the hex-encoded HMAC
	expectedHMACBytes, err := hex.DecodeString(expectedHMAC)
	if err != nil {
		return false, fmt.Errorf("failed to decode HMAC hex string: %w", err)
	}

	// Create a new HMAC-SHA256 instance
	mac := hmac.New(sha256.New, hmacKey)

	mac.Write([]byte(code))

	// Calculate the HMAC
	calculatedHMAC := mac.Sum(nil)

	// Compare the HMACs using a constant-time comparison
	return hmac.Equal(calculatedHMAC, expectedHMACBytes), nil
}
