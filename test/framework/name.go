package framework

import (
	"fmt"
	"math/rand"
	"time"
)

// allowedChars specifies the characters suitable for Kubernetes resource names.
var allowedChars = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func UniqueName(name string, length int) string {
	suffix := GenerateRandString(length)
	return fmt.Sprintf("%s-%s", name, string(suffix))
}

func GenerateName(name string, id, suffixLength int) string {
	return fmt.Sprintf("%s-%d", UniqueName(name, suffixLength), id)
}

// GenerateRandString generates a random string of fixed size.
func GenerateRandString(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	suffix := make([]rune, n)
	for i := range suffix {
		suffix[i] = allowedChars[r.Int63()%int64(len(allowedChars))]
	}
	return string(suffix)
}
