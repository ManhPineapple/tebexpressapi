package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
)

// GetMD5Hash -- get md5 hash from a string
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetSHA256Hash -- get sha_256 hash from a string
func GetSHA256Hash(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func hash(salted string) string {
	shaHash := sha512.New()
	shaHash.Write([]byte(salted))
	return string(shaHash.Sum(nil))
}

// GetSHA512Hash -- get sha_512 hash from a string
func GetSHA512Hash(text string) string {
	shaHash := sha512.New()
	shaHash.Write([]byte(text))
	return hex.EncodeToString(shaHash.Sum(nil))
}

func GetHmacSHA256Hash(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))

	encode := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return encode
}
