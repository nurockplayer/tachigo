package services

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// siweMessage builds the EIP-4361 message that the user must sign.
func siweMessage(address, nonce string) string {
	return fmt.Sprintf(
		"tachigo.io wants you to sign in with your Ethereum account:\n%s\n\nSign in to Tachigo\n\nNonce: %s\nIssued At: %s",
		address, nonce, time.Now().UTC().Format(time.RFC3339),
	)
}

// verifyEthSignature recovers the signer address from a personal_sign
// signature and compares it to expectedAddress case-insensitively.
func verifyEthSignature(message, sigHex, expectedAddress string) bool {
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(sigHex, "0x"))
	if err != nil || len(sigBytes) != 65 {
		return false
	}

	prefixed := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(prefixed))

	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false
	}
	recovered := strings.ToLower(crypto.PubkeyToAddress(*pubKey).Hex())
	return recovered == strings.ToLower(expectedAddress)
}
