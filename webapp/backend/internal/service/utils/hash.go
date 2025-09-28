package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashString は文字列の安全なハッシュを返します
func HashString(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}
