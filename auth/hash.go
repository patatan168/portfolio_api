package auth

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

func CreateHexSha3(str string) string {
	shaByte := sha3.Sum256([]byte(str))
	return hex.EncodeToString(shaByte[:])
}
