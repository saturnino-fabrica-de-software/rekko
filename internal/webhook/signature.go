package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func Verify(secret string, payload []byte, signature string) bool {
	expectedSignature := Sign(secret, payload)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
