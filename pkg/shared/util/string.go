package util

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

// generate a random string with given length
func RandomString(length int) string {
	seeds := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(seeds))))
		result[i] = seeds[num.Int64()]
	}
	return string(result)
}

// ConvertToDNSLabel converts an arbitrary string into a valid Kubernetes DNS label
func ConvertToDNSLabel(input string) string {
	// Lowercase the string
	label := strings.ToLower(input)

	// Replace invalid characters with "-"
	label = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(label, "-")

	// Trim leading and trailing non-alphanumeric characters
	label = strings.Trim(label, "-")

	// Ensure the label is not longer than 63 characters
	if len(label) > 63 {
		label = label[:63]
	}

	return label
}
