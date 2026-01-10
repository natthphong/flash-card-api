package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

func BuildCacheKey(text string) string {
	speed := "1.0"
	format := "mp3"
	locale := "en-US"
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	text = reg.ReplaceAllString(text, "")
	raw := fmt.Sprintf("%s|%s|%s|%s", text, speed, format, locale)

	hash := sha256.Sum256([]byte(raw))

	return hex.EncodeToString(hash[:])
}
