package run

import (
	"fmt"
	"log"
	"math/rand"
	"os"
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
	if err := ValidateHostname(name); err != nil {
		return fmt.Errorf("[ERROR] Invalid cluster name\n%+v", ValidateHostname(name))
	}
	if len(name) > clusterNameMaxSize {
		return fmt.Errorf("[ERROR] Cluster name is too long (%d > %d)", len(name), clusterNameMaxSize)
	}
	return nil
}

// ValidateHostname ensures that a cluster name is also a valid host name according to RFC 1123.
func ValidateHostname(name string) error {

	if len(name) == 0 {
		return fmt.Errorf("[ERROR] no name provided")
	}

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

// checkDefaultBindMounts takes a volumes slice and a defaults slice
// and appends the volumes to the defaults, with respect to already set vaules
// in defaults in volumes, so as to not cause duplication of the same string.
func checkDefaultBindMounts(volumes []string, defaults []string) []string {
	newVols := make([]string, 0)
	destHm := make(map[string]int)

	// populate the desthm with indexes in the defaults
	for i, v := range defaults {
		p := strings.Split(v, ":")
		if len(p) != 2 {
			continue
		}

		destPath := p[1]
		destHm[destPath] = i
	}

	// check if we overrode a default in the -v, if so, remove
	// that default bind mount from ever being processed.
	for _, v := range volumes {
		p := strings.Split(v, ":")
		if len(p) != 2 {
			continue
		}

		destPath := p[1]
		if i, ok := destHm[destPath]; ok {
			defaults[i] = defaults[len(defaults)-1]
			defaults[len(defaults)-1] = ""
			defaults = defaults[:len(defaults)-1]
		}
	}

	// add the defaults, checking to ensure that the local path (if it's a :) exists
	for _, v := range defaults {
		p := strings.Split(v, ":")
		if len(p) > 1 {
			// check if the path of a local:remote exists
			if _, err := os.Stat(p[0]); os.IsNotExist(err) {
				continue
			}

			log.Printf("Including extra mount '%s' due to local path existing", v)
		}

		newVols = append(newVols, v)
	}

	newVols = append(newVols, volumes...)

	return newVols
}
