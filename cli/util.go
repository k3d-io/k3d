package run

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// GenerateRandomString thanks to https://stackoverflow.com/a/31832326/6450189
// GenerateRandomString is used to generate a random string that is used as a cluster secret
func GenerateRandomString(n int) string {

	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

/*** Cluster Name Validation ***/
const clusterNameMaxSize int = 35

// CheckClusterName ensures that a cluster name is also a valid host name according to RFC 1123.
// We further restrict the length of the cluster name to maximum 'clusterNameMaxSize'
// so that we can construct the host names based on the cluster name, and still stay
// within the 64 characters limit.
func CheckClusterName(name string) error {
	if len(name) > clusterNameMaxSize {
		return fmt.Errorf("[ERROR] Cluster name is too long (%d > %d)", len(name), clusterNameMaxSize)
	}
	return fmt.Errorf("[ERROR] Invalid cluster name\n%+v", ValidateHostname(name))
}

// ValidateHostname ensures that a cluster name is also a valid host name according to RFC 1123.
func ValidateHostname(name string) error {
	if name[0] == '-' || name[len(name)-1] == '-' {
		return fmt.Errorf("[ERROR] Hostname [%s] must not start or end with - (dash)", name)
	}

	for _, c := range name {
		switch {
		case '0' <= c && c <= '9':
		case 'a' <= c && c <= 'z':
		case 'A' <= c && c <= 'Z':
		case c == '-':
			break
		default:
			return fmt.Errorf("[ERROR] Hostname [%s] contains characters other than 'Aa-Zz', '0-9' or '-'", name)

		}
	}

	return nil
}
