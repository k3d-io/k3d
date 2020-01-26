package run

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type apiPort struct {
	Host   string
	HostIP string
	Port   string
}

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
		return fmt.Errorf("Invalid cluster name\n%+v", ValidateHostname(name))
	}
	if len(name) > clusterNameMaxSize {
		return fmt.Errorf("Cluster name is too long (%d > %d)", len(name), clusterNameMaxSize)
	}
	return nil
}

// ValidateHostname ensures that a cluster name is also a valid host name according to RFC 1123.
func ValidateHostname(name string) error {

	if len(name) == 0 {
		return fmt.Errorf("no name provided")
	}

	if name[0] == '-' || name[len(name)-1] == '-' {
		return fmt.Errorf("Hostname [%s] must not start or end with - (dash)", name)
	}

	for _, c := range name {
		switch {
		case '0' <= c && c <= '9':
		case 'a' <= c && c <= 'z':
		case 'A' <= c && c <= 'Z':
		case c == '-':
			break
		default:
			return fmt.Errorf("Hostname [%s] contains characters other than 'Aa-Zz', '0-9' or '-'", name)

		}
	}

	return nil
}

func parseAPIPort(portSpec string) (*apiPort, error) {
	var port *apiPort
	split := strings.Split(portSpec, ":")
	if len(split) > 2 {
		return nil, fmt.Errorf("api-port format error")
	}

	if len(split) == 1 {
		port = &apiPort{Port: split[0]}
	} else {
		// Make sure 'host' can be resolved to an IP address
		addrs, err := net.LookupHost(split[0])
		if err != nil {
			return nil, err
		}
		port = &apiPort{Host: split[0], HostIP: addrs[0], Port: split[1]}
	}

	// Verify 'port' is an integer and within port ranges
	p, err := strconv.Atoi(port.Port)
	if err != nil {
		return nil, err
	}

	if p < 0 || p > 65535 {
		return nil, fmt.Errorf("--api-port port value out of range")
	}

	return port, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

type dnsNameCheck struct {
	res     chan bool
	err     chan error
	timeout time.Duration
}

func newAsyncNameExists(name string, timeout time.Duration) *dnsNameCheck {
	d := &dnsNameCheck{
		res:     make(chan bool),
		err:     make(chan error),
		timeout: timeout,
	}
	go func() {
		addresses, err := net.LookupHost(name)
		if err != nil {
			d.err <- err
		}
		d.res <- len(addresses) > 0
	}()
	return d
}

func (d dnsNameCheck) Exists() (bool, error) {
	select {
	case r := <-d.res:
		return r, nil
	case e := <-d.err:
		return false, e
	case <-time.After(d.timeout):
		return false, nil
	}
}
