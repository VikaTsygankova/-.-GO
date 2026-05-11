package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

func ComputeHMAC(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func EncryptDemo(data, key string) string {
	return base64.StdEncoding.EncodeToString([]byte(key + ":" + data))
}

func DecryptDemo(data, key string) string {
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ""
	}
	prefix := key + ":"
	s := string(raw)
	return strings.TrimPrefix(s, prefix)
}

func GenerateCardNumber() string {
	prefix := "2200"
	body := prefix
	for len(body) < 15 {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		body += fmt.Sprint(n.Int64())
	}
	return body + fmt.Sprint(luhnCheckDigit(body))
}

func luhnCheckDigit(num string) int {
	sum := 0
	alt := true
	for i := len(num) - 1; i >= 0; i-- {
		d := int(num[i] - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return (10 - (sum % 10)) % 10
}

func RandomCVV() string {
	out := ""
	for i := 0; i < 3; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		out += fmt.Sprint(n.Int64())
	}
	return out
}
