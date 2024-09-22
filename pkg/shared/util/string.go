package util

import (
	"crypto/rand"
	"math/big"
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
