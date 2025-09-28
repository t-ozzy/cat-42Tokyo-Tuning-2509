package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

// HashPasswordPBKDF2 はPBKDF2を使用してパスワードをハッシュ化します
func HashPasswordPBKDF2(password string) (string, error) {
	// ランダムなソルトを生成
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// PBKDF2パラメータ
	iterations := 10000 // イテレーション回数
	keyLen := 32        // ハッシュ長

	hash := pbkdf2.Key([]byte(password), salt, iterations, keyLen, sha256.New)

	// パラメータを含めた形式でハッシュを保存
	return fmt.Sprintf("$pbkdf2-sha256$i=%d$%s$%s",
		iterations,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hash)), nil
}
