// Copyright 2020 The Inet.Af AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package netaddr contains a IP address type that's in many ways
// better than the Go standard library's net.IP type. Building on that
// IP type, the package also contains IPPrefix, IPPort, IPRange, and
// IPSet types.
//
// Notably, this package's IP type takes less memory, is immutable,
// comparable (supports == and being a map key), and more. See
// https://github.com/inetaf/netaddr for background.
package netaddr // import "inet.af/netaddr"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"

	"go4.org/intern"
)

// Sizes: (64-bit)
//   net.IP:     24 byte slice header + {4, 16} = 28 to 40 bytes
//   net.IPAddr: 40 byte slice header + {4, 16} = 44 to 56 bytes + zone length
//   netaddr.IP: 24 bytes (zone is per-name singleton, shared across all users)

// IP represents an IPv4 or IPv6 address (with or without a scoped
// addressing zone), similar to Go's net.IP or net.IPAddr.
//
// Unlike net.IP or net.IPAddr, the netaddr.IP is a comparable value
// type (it supports == and can be a map key) and is immutable.
// Its memory representation is 24 bytes on 64-bit machines (the same
// size as a Go slice header) for both IPv4 and IPv6 address.
type IP struct {
	// addr are the hi and lo bits of an IPv6 address. If z==z4,
	// hi and lo contain the IPv4-mapped IPv6 address.
	//
	// hi and lo are constructed by interpreting a 16-byte IPv6
	// address as a big-endian 128-bit number. The most significant
	// bits of that number go into hi, the rest into lo.
	//
	// For example, 0011:2233:4455:6677:8899:aabb:ccdd:eeff is stored as:
	//  addr.hi = 0x0011223344556677
	//  addr.lo = 0x8899aabbccddeeff
	//
	// We store IPs like this, rather than as [16]byte, because it
	// turns most operations on IPs into arithmetic and bit-twiddling
	// operations on 64-bit registers, which is much faster than
	// bytewise processing.
	addr uint128

	// z is a combination of the address family and the IPv6 zone.
	//
	// nil means invalid IP address (for the IP zero value).
	// z4 means an IPv4 address.
	// z6noz means an IPv6 address without a zone.
	//
	// Otherwise it's the interned zone name string.
	z *intern.Value
}

// z0, z4, and z6noz are sentinel IP.z values.
// See the IP type's field docs.
var (
	z0    = (*intern.Value)(nil)
	z4    = new(intern.Value)
	z6noz = new(intern.Value)
)

// IPv6LinkLocalAllNodes returns the IPv6 link-local all nodes multicast
// address ff02::1.
func IPv6LinkLocalAllNodes() IP { return IPv6Raw([16]byte{0: 0xff, 1: 0x02, 15: 0x01}) }

// IPv6Unspecified returns the IPv6 unspecified address ::.
func IPv6Unspecified() IP { return IP{z: z6noz} }

// IPv4 returns the IP of the IPv4 address a.b.c.d.
func IPv4(a, b, c, d uint8) IP {
	return IP{
		addr: uint128{0, 0xffff00000000 | uint64(a)<<24 | uint64(b)<<16 | uint64(c)<<8 | uint64(d)},
		z:    z4,
	}
}

// IPv6Raw returns the IPv6 address given by the bytes in addr,
// without an implicit Unmap call to unmap any v6-mapped IPv4
// address.
func IPv6Raw(addr [16]byte) IP {
	return IP{
		addr: uint128{
			binary.BigEndian.Uint64(addr[:8]),
			binary.BigEndian.Uint64(addr[8:]),
		},
		z: z6noz,
	}
}

// ipv6Slice is like IPv6Raw, but operates on a 16-byte slice. Assumes
// slice is 16 bytes, caller must enforce this.
func ipv6Slice(addr []byte) IP {
	return IP{
		addr: uint128{
			binary.BigEndian.Uint64(addr[:8]),
			binary.BigEndian.Uint64(addr[8:]),
		},
		z: z6noz,
	}
}

// IPFrom16 returns the IP address given by the bytes in addr,
// unmapping any v6-mapped IPv4 address.
//
// It is equivalent to calling IPv6Raw(addr).Unmap().
func IPFrom16(addr [16]byte) IP {
	return IPv6Raw(addr).Unmap()
}

// ParseIP parses s as an IP address, returning the result. The string
// s can be in dotted decimal ("192.0.2.1"), IPv6 ("2001:db8::68"),
// or IPv6 with a scoped addressing zone ("fe80::1cc0:3e8c:119f:c2e1%ens18").
func ParseIP(s string) (IP, error) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return parseIPv4(s)
		case ':':
			return parseIPv6(s)
		case '%':
			// Assume that this was trying to be an IPv6 address with
			// a zone specifier, but the address is missing.
			return IP{}, parseIPError{in: s, msg: "missing IPv6 address"}
		}
	}
	return IP{}, parseIPError{in: s, msg: "unable to parse IP"}
}

// MustParseIP calls ParseIP(s) and panics on error.
// It is intended for use in tests with hard-coded strings.
func MustParseIP(s string) IP {
	ip, err := ParseIP(s)
	if err != nil {
		panic(err)
	}
	return ip
}

type parseIPError struct {
	in  string // the string given to ParseIP
	msg string // an explanation of the parse failure
	at  string // optionally, the unparsed portion of in at which the error occurred.
}

func (err parseIPError) Error() string {
	if err.at != "" {
		return fmt.Sprintf("ParseIP(%q): %s (at %q)", err.in, err.msg, err.at)
	}
	return fmt.Sprintf("ParseIP(%q): %s", err.in, err.msg)
}

// parseIPv4 parses s as an IPv4 address (in form "192.168.0.1").
func parseIPv4(s string) (ip IP, err error) {
	var fields [3]uint8
	var val, pos int
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			val = val*10 + int(s[i]) - '0'
			if val > 255 {
				return IP{}, parseIPError{in: s, msg: "IPv4 field has value >255"}
			}
		} else if s[i] == '.' {
			// .1.2.3
			// 1.2.3.
			// 1..2.3
			if i == 0 || i == len(s)-1 || s[i-1] == '.' {
				return IP{}, parseIPError{in: s, msg: "IPv4 field must have at least one digit", at: s[i:]}
			}
			// 1.2.3.4.5
			if pos == 3 {
				return IP{}, parseIPError{in: s, msg: "IPv4 address too long"}
			}
			fields[pos] = uint8(val)
			pos++
			val = 0
		} else {
			return IP{}, parseIPError{in: s, msg: "unexpected character", at: s[i:]}
		}
	}
	if pos < 3 {
		return IP{}, parseIPError{in: s, msg: "IPv4 address too short"}
	}
	return IPv4(fields[0], fields[1], fields[2], uint8(val)), nil
}

// parseIPv6 parses s as an IPv6 address (in form "2001:db8::68").
func parseIPv6(in string) (IP, error) {
	s := in

	// Split off the zone right from the start. Yes it's a second scan
	// of the string, but trying to handle it inline makes a bunch of
	// other inner loop conditionals more expensive, and it ends up
	// being slower.
	zone := ""
	i := strings.IndexByte(s, '%')
	if i != -1 {
		s, zone = s[:i], s[i+1:]
		if zone == "" {
			// Not allowed to have an empty zone if explicitly specified.
			return IP{}, parseIPError{in: in, msg: "zone must be a non-empty string"}
		}
	}

	var ip [16]byte
	ellipsis := -1 // position of ellipsis in ip

	// Might have leading ellipsis
	if len(s) >= 2 && s[0] == ':' && s[1] == ':' {
		ellipsis = 0
		s = s[2:]
		// Might be only ellipsis
		if len(s) == 0 {
			return IPv6Unspecified().WithZone(zone), nil
		}
	}

	// Loop, parsing hex numbers followed by colon.
	i = 0
	for i < 16 {
		// Hex number. Similar to parseIPv4, inlining the hex number
		// parsing yields a significant performance increase.
		off := 0
		acc := uint32(0)
		for ; off < len(s); off++ {
			c := s[off]
			if c >= '0' && c <= '9' {
				acc = (acc << 4) + uint32(c-'0')
			} else if c >= 'a' && c <= 'f' {
				acc = (acc << 4) + uint32(c-'a'+10)
			} else if c >= 'A' && c <= 'F' {
				acc = (acc << 4) + uint32(c-'A'+10)
			} else {
				break
			}
			if acc > math.MaxUint16 {
				// Overflow, fail.
				return IP{}, parseIPError{in: in, msg: "IPv6 field has value >=2^16", at: s}
			}
		}
		if off == 0 {
			// No digits found, fail.
			return IP{}, parseIPError{in: in, msg: "each colon-separated field must have at least one digit", at: s}
		}

		// If followed by dot, might be in trailing IPv4.
		if off < len(s) && s[off] == '.' {
			if ellipsis < 0 && i != 12 {
				// Not the right place.
				return IP{}, parseIPError{in: in, msg: "embedded IPv4 address must replace the final 2 fields of the address", at: s}
			}
			if i+4 > 16 {
				// Not enough room.
				return IP{}, parseIPError{in: in, msg: "too many hex fields to fit an embedded IPv4 at the end of the address", at: s}
			}
			// TODO: could make this a bit faster by having a helper
			// that parses to a [4]byte, and have both parseIPv4 and
			// parseIPv6 use it.
			ip4, err := parseIPv4(s)
			if err != nil {
				return IP{}, parseIPError{in: in, msg: err.Error(), at: s}
			}
			ip[i] = ip4.v4(0)
			ip[i+1] = ip4.v4(1)
			ip[i+2] = ip4.v4(2)
			ip[i+3] = ip4.v4(3)
			s = ""
			i += 4
			break
		}

		// Save this 16-bit chunk.
		ip[i] = byte(acc >> 8)
		ip[i+1] = byte(acc)
		i += 2

		// Stop at end of string.
		s = s[off:]
		if len(s) == 0 {
			break
		}

		// Otherwise must be followed by colon and more.
		if s[0] != ':' {
			return IP{}, parseIPError{in: in, msg: "unexpected character, want colon", at: s}
		} else if len(s) == 1 {
			return IP{}, parseIPError{in: in, msg: "colon must be followed by more characters", at: s}
		}
		s = s[1:]

		// Look for ellipsis.
		if s[0] == ':' {
			if ellipsis >= 0 { // already have one
				return IP{}, parseIPError{in: in, msg: "multiple :: in address", at: s}
			}
			ellipsis = i
			s = s[1:]
			if len(s) == 0 { // can be at end
				break
			}
		}
	}

	// Must have used entire string.
	if len(s) != 0 {
		return IP{}, parseIPError{in: in, msg: "trailing garbage after address", at: s}
	}

	// If didn't parse enough, expand ellipsis.
	if i < 16 {
		if ellipsis < 0 {
			return IP{}, parseIPError{in: in, msg: "address string too short"}
		}
		n := 16 - i
		for j := i - 1; j >= ellipsis; j-- {
			ip[j+n] = ip[j]
		}
		for j := ellipsis + n - 1; j >= ellipsis; j-- {
			ip[j] = 0
		}
	} else if ellipsis >= 0 {
		// Ellipsis must represent at least one 0 group.
		return IP{}, parseIPError{in: in, msg: "the :: must expand to at least one field of zeros"}
	}
	return IPv6Raw(ip).WithZone(zone), nil
}

// FromStdIP returns an IP from the standard library's IP type.
//
// If std is invalid, ok is false.
//
// FromStdIP implicitly unmaps IPv6-mapped IPv4 addresses. That is, if
// len(std) == 16 and contains an IPv4 address, only the IPv4 part is
// returned, without the IPv6 wrapper. This is the common form returned by
// the standard library's ParseIP: https://play.golang.org/p/qdjylUkKWxl.
// To convert a standard library IP without the implicit unmapping, use
// FromStdIPRaw.
func FromStdIP(std net.IP) (ip IP, ok bool) {
	ret, ok := FromStdIPRaw(std)
	if ret.Is4in6() {
		ret.z = z4
	}
	return ret, ok
}

// FromStdIPRaw returns an IP from the standard library's IP type.
// If std is invalid, ok is false.
// Unlike FromStdIP, FromStdIPRaw does not do an implicit Unmap if
// len(std) == 16 and contains an IPv6-mapped IPv4 address.
func FromStdIPRaw(std net.IP) (ip IP, ok bool) {
	switch len(std) {
	case 4:
		return IPv4(std[0], std[1], std[2], std[3]), true
	case 16:
		return ipv6Slice(std), true
	}
	return IP{}, false
}

// v4 returns the i'th byte of ip. If ip is not an IPv4, v4 returns
// unspecified garbage.
func (ip IP) v4(i uint8) uint8 {
	return uint8(ip.addr.lo >> ((3 - i) * 8))
}

// v6 returns the i'th byte of ip. If ip is an IPv4 address, this
// accesses the IPv4-mapped IPv6 address form of the IP.
func (ip IP) v6(i uint8) uint8 {
	return uint8(*(ip.addr.halves()[(i/8)%2]) >> ((7 - i%8) * 8))
}

func (ip IP) v6u16(i uint8) uint16 {
	return uint16(*(ip.addr.halves()[(i/4)%2]) >> ((3 - i%4) * 16))
}

// IsZero reports whether ip is the zero value of the IP type.
// The zero value is not a valid IP address of any type.
//
// Note that "0.0.0.0" and "::" are not the zero value.
func (ip IP) IsZero() bool {
	// Faster than comparing ip == IP{}, but effectively equivalent,
	// as there's no way to make an IP with a nil z from this package.
	return ip.z == z0
}

// BitLen returns the number of bits in the IP address:
// 32 for IPv4 or 128 for IPv6.
// For the zero value (see IP.IsZero), it returns 0.
// For IP4-mapped IPv6 addresses, it returns 128.
func (ip IP) BitLen() uint8 {
	switch ip.z {
	case z0:
		return 0
	case z4:
		return 32
	}
	return 128
}

// Zone returns ip's IPv6 scoped addressing zone, if any.
func (ip IP) Zone() string {
	if ip.z == nil {
		return ""
	}
	zone, _ := ip.z.Get().(string)
	return zone
}

// Compare returns an integer comparing two IPs.
// The result will be 0 if ip==ip2, -1 if ip < ip2, and +1 if ip > ip2.
// The definition of "less than" is the same as the IP.Less method.
func (ip IP) Compare(ip2 IP) int {
	f1, f2 := ip.BitLen(), ip2.BitLen()
	if f1 < f2 {
		return -1
	}
	if f1 > f2 {
		return 1
	}
	if hi1, hi2 := ip.addr.hi, ip2.addr.hi; hi1 < hi2 {
		return -1
	} else if hi1 > hi2 {
		return 1
	}
	if lo1, lo2 := ip.addr.lo, ip2.addr.lo; lo1 < lo2 {
		return -1
	} else if lo1 > lo2 {
		return 1
	}
	if ip.Is6() {
		za, zb := ip.Zone(), ip2.Zone()
		if za < zb {
			return -1
		} else if za > zb {
			return 1
		}
	}
	return 0
}

// Less reports whether ip sorts before ip2.
// IP addresses sort first by length, then their address.
// IPv6 addresses with zones sort just after the same address without a zone.
func (ip IP) Less(ip2 IP) bool { return ip.Compare(ip2) == -1 }

func (ip IP) lessOrEq(ip2 IP) bool { return ip.Compare(ip2) <= 0 }

// ipZone returns the standard library net.IP from ip, as well
// as the zone.
// The optional reuse IP provides memory to reuse.
func (ip IP) ipZone(reuse net.IP) (stdIP net.IP, zone string) {
	base := reuse[:0]
	switch {
	case ip.z == z0:
		return nil, ""
	case ip.Is4():
		a4 := ip.As4()
		return append(base, a4[:]...), ""
	default:
		a16 := ip.As16()
		return append(base, a16[:]...), ip.Zone()
	}
}

// IPAddr returns the net.IPAddr representation of an IP. The returned value is
// always non-nil, but the IPAddr.IP will be nil if ip is the zero value.
// If ip contains a zone identifier, IPAddr.Zone is populated.
func (ip IP) IPAddr() *net.IPAddr {
	stdIP, zone := ip.ipZone(nil)
	return &net.IPAddr{IP: stdIP, Zone: zone}
}

// Is4 reports whether ip is an IPv4 address.
//
// It returns false for IP4-mapped IPv6 addresses. See IP.Unmap.
func (ip IP) Is4() bool {
	return ip.z == z4
}

// Is4in6 reports whether ip is an IPv4-mapped IPv6 address.
func (ip IP) Is4in6() bool {
	return ip.Is6() && ip.addr.hi == 0 && ip.addr.lo>>32 == 0xffff
}

// Is6 reports whether ip is an IPv6 address, including IPv4-mapped
// IPv6 addresses.
func (ip IP) Is6() bool {
	return ip.z != z0 && ip.z != z4
}

// Unmap returns ip with any IPv4-mapped IPv6 address prefix removed.
//
// That is, if ip is an IPv6 address wrapping an IPv4 adddress, it
// returns the wrapped IPv4 address. Otherwise it returns ip, regardless
// of its type.
func (ip IP) Unmap() IP {
	if ip.Is4in6() {
		ip.z = z4
	}
	return ip
}

// WithZone returns an IP that's the same as ip but with the provided
// zone. If zone is empty, the zone is removed. If ip is an IPv4
// address it's returned unchanged.
func (ip IP) WithZone(zone string) IP {
	if !ip.Is6() {
		return ip
	}
	if zone == "" {
		ip.z = z6noz
		return ip
	}
	ip.z = intern.GetByString(zone)
	return ip
}

// IsLinkLocalUnicast reports whether ip is a link-local unicast address.
// If ip is the zero value, it will return false.
func (ip IP) IsLinkLocalUnicast() bool {
	if ip.Is4() {
		return ip.v4(0) == 169 && ip.v4(1) == 254
	}
	if ip.Is6() {
		return ip.v6u16(0) == 0xfe80
	}
	return false // zero value
}

// IsLoopback reports whether ip is a loopback address. If ip is the zero value,
// it will return false.
func (ip IP) IsLoopback() bool {
	if ip.Is4() {
		return ip.v4(0) == 127
	}
	if ip.Is6() {
		return ip.addr.hi == 0 && ip.addr.lo == 1
	}
	return false // zero value
}

// IsMulticast reports whether ip is a multicast address. If ip is the zero
// value, it will return false.
func (ip IP) IsMulticast() bool {
	if ip.Is4() {
		return ip.v4(0)&0xf0 == 0xe0
	}
	if ip.Is6() {
		return ip.addr.hi>>(64-8) == 0xff // ip.v6(0) == 0xff
	}
	return false // zero value
}

// IsInterfaceLocalMulticast reports whether ip is an IPv6 interface-local
// multicast address. If ip is the zero value or an IPv4 address, it will return
// false.
func (ip IP) IsInterfaceLocalMulticast() bool {
	if ip.Is6() {
		return ip.v6u16(0)&0xff0f == 0xff01
	}
	return false // zero value
}

// IsLinkLocalMulticast reports whether ip is a link-local multicast address.
// If ip is the zero value, it will return false.
func (ip IP) IsLinkLocalMulticast() bool {
	if ip.Is4() {
		return ip.v4(0) == 224 && ip.v4(1) == 0 && ip.v4(2) == 0
	}
	if ip.Is6() {
		return ip.v6u16(0)&0xff0f == 0xff02
	}
	return false // zero value
}

// Prefix applies a CIDR mask of leading bits to IP, producing an IPPrefix
// of the specified length. If IP is the zero value, a zero-value IPPrefix and
// a nil error are returned. If bits is larger than 32 for an IPv4 address or
// 128 for an IPv6 address, an error is returned.
func (ip IP) Prefix(bits uint8) (IPPrefix, error) {
	effectiveBits := bits
	switch ip.z {
	case z0:
		return IPPrefix{}, nil
	case z4:
		if bits > 32 {
			return IPPrefix{}, fmt.Errorf("prefix length %d too large for IPv4", bits)
		}
		effectiveBits += 96
	default:
		if bits > 128 {
			return IPPrefix{}, fmt.Errorf("prefix length %d too large for IPv6", bits)
		}
	}
	ip.addr = ip.addr.and(mask6[effectiveBits])
	return IPPrefix{ip, bits}, nil
}

// Netmask applies a bit mask to IP, producing an IPPrefix. If IP is the
// zero value, a zero-value IPPrefix and a nil error are returned. If the
// netmask length is not 4 for IPv4 or 16 for IPv6, an error is
// returned. If the netmask is non-contiguous, an error is returned.
func (ip *IP) Netmask(mask []byte) (IPPrefix, error) {
	l := len(mask)

	switch ip.z {
	case z0:
		return IPPrefix{}, nil
	case z4:
		if l != net.IPv4len {
			return IPPrefix{}, fmt.Errorf("netmask length %d incorrect for IPv4", l)
		}
	default:
		if l != net.IPv6len {
			return IPPrefix{}, fmt.Errorf("netmask length %d incorrect for IPv6", l)
		}
	}

	ones, bits := net.IPMask(mask).Size()
	if ones == 0 && bits == 0 {
		return IPPrefix{}, errors.New("netmask is non-contiguous")
	}

	return ip.Prefix(uint8(ones))
}

// As16 returns the IP address in its 16 byte representation.
// IPv4 addresses are returned in their v6-mapped form.
// IPv6 addresses with zones are returned without their zone (use the
// Zone method to get it).
// The ip zero value returns all zeroes.
func (ip IP) As16() [16]byte {
	var ret [16]byte
	binary.BigEndian.PutUint64(ret[:8], ip.addr.hi)
	binary.BigEndian.PutUint64(ret[8:], ip.addr.lo)
	return ret
}

// As4 returns an IPv4 or IPv4-in-IPv6 address in its 4 byte representation.
// If ip is the IP zero value or an IPv6 address, As4 panics.
// Note that 0.0.0.0 is not the zero value.
func (ip IP) As4() [4]byte {
	if ip.z == z4 || ip.Is4in6() {
		var ret [4]byte
		binary.BigEndian.PutUint32(ret[:], uint32(ip.addr.lo))
		return ret
	}
	if ip.z == z0 {
		panic("As4 called on IP zero value")
	}
	panic("As4 called on IPv6 address")
}

// String returns the string form of the IP address ip.
// It returns one of 4 forms:
//
//   - "invalid IP", if ip is the zero value
//   - IPv4 dotted decimal ("192.0.2.1")
//   - IPv6 ("2001:db8::1")
//   - IPv6 with zone ("fe80:db8::1%eth0")
//
// Note that unlike the Go standard library's IP.String method,
// IP4-mapped IPv6 addresses do not format as dotted decimals.
func (ip IP) String() string {
	switch ip.z {
	case z0:
		return "invalid IP"
	case z4:
		return ip.string4()
	default:
		return ip.string6()
	}
}

// digits is a string of the hex digits from 0 to f. It's used in
// appendDecimal and appendHex to format IP addresses.
const digits = "0123456789abcdef"

// appendDecimal appends the decimal string representation of x to b.
func appendDecimal(b []byte, x uint8) []byte {
	// Using this function rather than strconv.AppendUint makes IPv4
	// string building 2x faster.

	if x >= 100 {
		b = append(b, digits[x/100])
	}
	if x >= 10 {
		b = append(b, digits[x/10%10])
	}
	return append(b, digits[x%10])
}

// appendHex appends the hex string representation of x to b.
func appendHex(b []byte, x uint16) []byte {
	// Using this function rather than strconv.AppendUint makes IPv6
	// string building 2x faster.

	if x >= 0x1000 {
		b = append(b, digits[x>>12])
	}
	if x >= 0x100 {
		b = append(b, digits[x>>8&0xf])
	}
	if x >= 0x10 {
		b = append(b, digits[x>>4&0xf])
	}
	return append(b, digits[x&0xf])
}

func (ip IP) string4() string {
	const max = len("255.255.255.255")
	ret := make([]byte, 0, max)
	ret = appendDecimal(ret, ip.v4(0))
	ret = append(ret, '.')
	ret = appendDecimal(ret, ip.v4(1))
	ret = append(ret, '.')
	ret = appendDecimal(ret, ip.v4(2))
	ret = append(ret, '.')
	ret = appendDecimal(ret, ip.v4(3))
	return string(ret)
}

// string6 formats ip in IPv6 textual representation. It follows the
// guidelines in section 4 of RFC 5952
// (https://tools.ietf.org/html/rfc5952#section-4): no unnecessary
// zeros, use :: to elide the longest run of zeros, and don't use ::
// to compact a single zero field.
func (ip IP) string6() string {
	zeroStart, zeroEnd := uint8(255), uint8(255)
	for i := uint8(0); i < 8; i++ {
		j := i
		for j < 8 && ip.v6u16(j) == 0 {
			j++
		}
		if l := j - i; l >= 2 && l > zeroEnd-zeroStart {
			zeroStart, zeroEnd = i, j
		}
	}

	// Use a zone with a "plausibly long" name, so that most zone-ful
	// IP addresses won't require additional allocation.
	//
	// The compiler does a cool optimization here, where ret ends up
	// stack-allocated and so the only allocation this function does
	// is to construct the returned string. As such, it's okay to be a
	// bit greedy here, size-wise.
	const max = len("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff%enp5s0")
	ret := make([]byte, 0, max)
	for i := uint8(0); i < 8; i++ {
		if i == zeroStart {
			ret = append(ret, ':', ':')
			i = zeroEnd
			if i >= 8 {
				break
			}
		} else if i > 0 {
			ret = append(ret, ':')
		}

		ret = appendHex(ret, ip.v6u16(i))
	}

	if ip.z != z6noz {
		ret = append(ret, '%')
		ret = append(ret, ip.Zone()...)
	}
	return string(ret)
}

// MarshalText implements the encoding.TextMarshaler interface,
// The encoding is the same as returned by String, with one exception:
// If ip is the zero value, the encoding is the empty string.
func (ip IP) MarshalText() ([]byte, error) {
	if ip.z == z0 {
		return []byte(""), nil
	}
	return []byte(ip.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The IP address is expected in a form accepted by ParseIP.
// It returns an error if *ip is not the IP zero value.
func (ip *IP) UnmarshalText(text []byte) error {
	if ip.z != z0 {
		return errors.New("netaddr: refusing to Unmarshal into non-zero IP")
	}
	if len(text) == 0 {
		return nil
	}
	var err error
	*ip, err = ParseIP(string(text))
	return err
}

// IPPort is an IP & port number.
//
// It's meant to be used as a value type.
type IPPort struct {
	IP   IP
	Port uint16
}

// splitIPPort splits s into an IP address string and a port
// string. It splits strings shaped like "foo:bar" or "[foo]:bar",
// without further validating the substrings. v6 indicates whether the
// ip string should parse as an IPv6 address or an IPv4 address, in
// order for s to be a valid ip:port string.
func splitIPPort(s string) (ip, port string, v6 bool, err error) {
	i := strings.LastIndexByte(s, ':')
	if i == -1 {
		return "", "", false, errors.New("not an ip:port")
	}

	ip, port = s[:i], s[i+1:]
	if len(ip) == 0 {
		return "", "", false, errors.New("no IP")
	}
	if len(port) == 0 {
		return "", "", false, errors.New("no port")
	}
	if ip[0] == '[' {
		if len(ip) < 2 || ip[len(ip)-1] != ']' {
			return "", "", false, errors.New("missing ]")
		}
		ip = ip[1 : len(ip)-1]
		v6 = true
	}

	return ip, port, v6, nil
}

// ParseIPPort parses s as an IPPort.
//
// It doesn't do any name resolution, and ports must be numeric.
func ParseIPPort(s string) (IPPort, error) {
	var ipp IPPort
	ip, port, v6, err := splitIPPort(s)
	if err != nil {
		return ipp, err
	}
	port16, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return ipp, fmt.Errorf("invalid port %q parsing %q", port, s)
	}
	ipp.Port = uint16(port16)
	ipp.IP, err = ParseIP(ip)
	if err != nil {
		return IPPort{}, err
	}
	if v6 && ipp.IP.Is4() {
		return IPPort{}, fmt.Errorf("invalid ip:port %q, square brackets can only be used with IPv6 addresses", s)
	} else if !v6 && ipp.IP.Is6() {
		return IPPort{}, fmt.Errorf("invalid ip:port %q, IPv6 addresses must be surrounded by square brackets", s)
	}
	return ipp, nil
}

// MustParseIPPort calls ParseIPPort(s) and panics on error.
// It is intended for use in tests with hard-coded strings.
func MustParseIPPort(s string) IPPort {
	ip, err := ParseIPPort(s)
	if err != nil {
		panic(err)
	}
	return ip
}

// IsZero reports whether p is its zero value.
func (p IPPort) IsZero() bool { return p == IPPort{} }

func (p IPPort) String() string {
	if p.IP.z == z4 {
		a := p.IP.As4()
		return fmt.Sprintf("%d.%d.%d.%d:%d", a[0], a[1], a[2], a[3], p.Port)
	}
	// TODO: this could be more efficient allocation-wise:
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

// MarshalText implements the encoding.TextMarshaler interface. The
// encoding is the same as returned by String, with one exception: if
// p.IP is the zero value, the encoding is the empty string.
func (p IPPort) MarshalText() ([]byte, error) {
	if p.IP.z == z0 {
		return []byte(""), nil
	}
	return []byte(p.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler
// interface. The IPPort is expected in a form accepted by
// ParseIPPort. It returns an error if *p is not the IPPort zero
// value.
func (p *IPPort) UnmarshalText(text []byte) error {
	if p.IP.z != z0 || p.Port != 0 {
		return errors.New("netaddr: refusing to UnmarshalText into non-zero IP")
	}
	if len(text) == 0 {
		return nil
	}
	var err error
	*p, err = ParseIPPort(string(text))
	return err
}

// FromStdAddr maps the components of a standard library TCPAddr or
// UDPAddr into an IPPort.
func FromStdAddr(stdIP net.IP, port int, zone string) (_ IPPort, ok bool) {
	ip, ok := FromStdIP(stdIP)
	if !ok || port < 0 || port > math.MaxUint16 {
		return
	}
	ip = ip.Unmap()
	if zone != "" {
		if ip.Is4() {
			ok = false
			return
		}
		ip = ip.WithZone(zone)
	}
	return IPPort{IP: ip, Port: uint16(port)}, true
}

// UDPAddr returns a standard library net.UDPAddr from p.
// The returned value is always non-nil. If p.IP is the zero
// value, then UDPAddr.IP is nil.
//
// UDPAddr necessarily does two allocations. If you have an existing
// UDPAddr already allocated, see UDPAddrAt.
func (p IPPort) UDPAddr() *net.UDPAddr {
	ret := &net.UDPAddr{
		Port: int(p.Port),
	}
	ret.IP, ret.Zone = p.IP.ipZone(nil)
	return ret
}

// UDPAddrAt is like UDPAddr, but reuses the provided UDPAddr, which
// must be non-nil. If at.IP has a capacity of 16, UDPAddrAt is
// allocation-free. It returns at to facilitate using this method as a
// wrapper.
func (p IPPort) UDPAddrAt(at *net.UDPAddr) *net.UDPAddr {
	at.Port = int(p.Port)
	at.IP, at.Zone = p.IP.ipZone(at.IP)
	return at
}

// TCPAddr returns a standard library net.TCPAddr from p.
// The returned value is always non-nil. If p.IP is the zero
// value, then TCPAddr.IP is nil.
func (p IPPort) TCPAddr() *net.TCPAddr {
	ip, zone := p.IP.ipZone(nil)
	return &net.TCPAddr{
		IP:   ip,
		Port: int(p.Port),
		Zone: zone,
	}
}

// IPPrefix is an IP address prefix (CIDR) representing an IP network.
//
// The first Bits of IP are specified, the remaining bits match any address.
// The range of Bits is [0,32] for IPv4 or [0,128] for IPv6.
type IPPrefix struct {
	IP   IP
	Bits uint8
}

// Valid reports whether whether p.Bits has a valid range for p.IP.
// If p.IP is zero, Valid returns false.
func (p IPPrefix) Valid() bool { return !p.IP.IsZero() && p.Bits <= p.IP.BitLen() }

// IsZero reports whether p is its zero value.
func (p IPPrefix) IsZero() bool { return p == IPPrefix{} }

// IsSingleIP reports whether p contains exactly one IP.
func (p IPPrefix) IsSingleIP() bool { return p.Bits != 0 && p.Bits == p.IP.BitLen() }

// FromStdIPNet returns an IPPrefix from the standard library's IPNet type.
// If std is invalid, ok is false.
func FromStdIPNet(std *net.IPNet) (prefix IPPrefix, ok bool) {
	ip, ok := FromStdIP(std.IP)
	if !ok {
		return IPPrefix{}, false
	}

	if l := len(std.Mask); l != net.IPv4len && l != net.IPv6len {
		// Invalid mask.
		return IPPrefix{}, false
	}

	ones, bits := std.Mask.Size()
	if ones == 0 && bits == 0 {
		// IPPrefix does not support non-contiguous masks.
		return IPPrefix{}, false
	}

	return IPPrefix{
		IP:   ip,
		Bits: uint8(ones),
	}, true
}

// ParseIPPrefix parses s as an IP address prefix.
// The string can be in the form "192.168.1.0/24" or "2001::db8::/32",
// the CIDR notation defined in RFC 4632 and RFC 4291.
//
// Note that masked address bits are not zeroed. Use Masked for that.
func ParseIPPrefix(s string) (IPPrefix, error) {
	i := strings.IndexByte(s, '/')
	if i < 0 {
		return IPPrefix{}, fmt.Errorf("netaddr.ParseIPPrefix(%q): no '/'", s)
	}
	ip, err := ParseIP(s[:i])
	if err != nil {
		return IPPrefix{}, fmt.Errorf("netaddr.ParseIPPrefix(%q): %v", s, err)
	}
	s = s[i+1:]
	bits, err := strconv.Atoi(s)
	if err != nil {
		return IPPrefix{}, fmt.Errorf("netaddr.ParseIPPrefix(%q): bad prefix: %v", s, err)
	}
	maxBits := 32
	if ip.Is6() {
		maxBits = 128
	}
	if bits < 0 || bits > maxBits {
		return IPPrefix{}, fmt.Errorf("netaddr.ParseIPPrefix(%q): prefix length out of range", s)
	}
	return IPPrefix{
		IP:   ip,
		Bits: uint8(bits),
	}, nil
}

// MustParseIPPrefix calls ParseIPPrefix(s) and panics on error.
// It is intended for use in tests with hard-coded strings.
func MustParseIPPrefix(s string) IPPrefix {
	ip, err := ParseIPPrefix(s)
	if err != nil {
		panic(err)
	}
	return ip
}

// Masked returns p in its canonical form, with bits of p.IP not in p.Bits masked off.
// If p is zero or otherwise invalid, Masked returns the zero value.
func (p IPPrefix) Masked() IPPrefix {
	if m, err := p.IP.Prefix(p.Bits); err == nil {
		return m
	}
	return IPPrefix{}
}

// Range returns the inclusive range of IPs that p covers.
//
// If p is zero or otherwise invalid, Range returns the zero value.
func (p IPPrefix) Range() IPRange {
	p = p.Masked()
	if p.IsZero() {
		return IPRange{}
	}
	return IPRange{From: p.IP, To: p.lastIP()}
}

// IPNet returns the net.IPNet representation of an IPPrefix.
// The returned value is always non-nil.
// Any zone identifier is dropped in the conversion.
func (p IPPrefix) IPNet() *net.IPNet {
	if !p.Valid() {
		return &net.IPNet{}
	}
	stdIP, _ := p.IP.ipZone(nil)
	return &net.IPNet{
		IP:   stdIP,
		Mask: net.CIDRMask(int(p.Bits), int(p.IP.BitLen())),
	}
}

// Contains reports whether the network p includes ip.
//
// An IPv4 address will not match an IPv6 prefix.
// A v6-mapped IPv6 address will not match an IPv4 prefix.
// A zero-value IP will not match any prefix.
func (p IPPrefix) Contains(ip IP) bool {
	if !p.Valid() {
		return false
	}
	if f1, f2 := p.IP.BitLen(), ip.BitLen(); f1 == 0 || f2 == 0 || f1 != f2 {
		return false
	}
	if ip.Is4() {
		// xor the IP addresses together; mismatched bits are now ones.
		// Shift away the number of bits we don't care about.
		// Shifts in Go are more efficient if the compiler can prove
		// that the shift amount is smaller than the width of the shifted type (64 here).
		// We know that p.Bits is in the range 0..32 because p is Valid;
		// the compiler doesn't know that, so mask with 63 to help it.
		// Now truncate to 32 bits, because this is IPv4.
		// If all the bits we care about are equal, the result will be zero.
		return uint32((ip.addr.lo^p.IP.addr.lo)>>((32-p.Bits)&63)) == 0
	} else {
		// xor the IP addresses together.
		// Mask away the bits we don't care about.
		// If all the bits we care about are equal, the result will be zero.
		return ip.addr.xor(p.IP.addr).and(mask6[p.Bits]).isZero()
	}
}

// Overlaps reports whether p and o overlap at all.
//
// If p and o are of different address families or either have a zero
// IP, it reports false. Like the Contains method, a prefix with a
// v6-mapped IPv4 IP is still treated as an IPv6 mask.
//
// If either has a Bits of zero, it returns true.
func (p IPPrefix) Overlaps(o IPPrefix) bool {
	if !p.Valid() || !o.Valid() {
		return false
	}
	if p == o {
		return true
	}
	if p.IP.Is4() != o.IP.Is4() {
		return false
	}
	var minBits uint8
	if p.Bits < o.Bits {
		minBits = p.Bits
	} else {
		minBits = o.Bits
	}
	if minBits == 0 {
		return true
	}
	// One of these Prefix calls might look redundant, but we don't require
	// that p and o values are normalized (via IPPrefix.Masked) first,
	// so the Prefix call on the one that's already minBits serves to zero
	// out any remaining bits in IP.
	var err error
	if p, err = p.IP.Prefix(minBits); err != nil {
		return false
	}
	if o, err = o.IP.Prefix(minBits); err != nil {
		return false
	}
	return p.IP == o.IP
}

// MarshalText implements the encoding.TextMarshaler interface,
// The encoding is the same as returned by String, with one exception:
// If p is the zero value, the encoding is the empty string.
func (p IPPrefix) MarshalText() ([]byte, error) {
	if p == (IPPrefix{}) {
		return []byte(""), nil
	}

	return []byte(p.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The IP address is expected in a form accepted by ParseIPPrefix.
// It returns an error if *p is not the IPPrefix zero value.
func (p *IPPrefix) UnmarshalText(text []byte) error {
	if *p != (IPPrefix{}) {
		return errors.New("netaddr: refusing to Unmarshal into non-zero IPPrefix")
	}

	if len(text) == 0 {
		return nil
	}

	var err error
	*p, err = ParseIPPrefix(string(text))
	return err
}

// String returns the CIDR notation of p: "<ip>/<bits>".
func (p IPPrefix) String() string {
	if !p.Valid() {
		return "invalid IP prefix"
	}
	return fmt.Sprintf("%s/%d", p.IP, p.Bits)
}

// lastIP returns the last IP in the prefix.
func (p IPPrefix) lastIP() IP {
	if !p.Valid() {
		return IP{}
	}
	a16 := p.IP.As16()
	var off uint8
	var bits uint8 = 128
	if p.IP.Is4() {
		off = 12
		bits = 32
	}
	for b := p.Bits; b < bits; b++ {
		byteNum, bitInByte := b/8, 7-(b%8)
		a16[off+byteNum] |= 1 << uint(bitInByte)
	}
	if p.IP.Is4() {
		return IPFrom16(a16)
	} else {
		return IPv6Raw(a16) // doesn't unmap
	}
}

// IPRange represents an inclusive range of IP addresses
// from the same address family.
//
// The From and To IPs are inclusive bounds, both included in the
// range.
//
// To be valid, the From and To values be non-zero, have matching
// address families (IPv4 vs IPv6), be in the same IPv6 zone (if any),
// and From must be less than or equal to To.
// An invalid range may be ignored.
type IPRange struct {
	// From is the initial IP address in the range.
	From IP

	// To is the final IP address in the range.
	To IP
}

// ParseIPRange parses a range out of two IPs separated by a hyphen.
//
// It returns an error if the range is not valid.
func ParseIPRange(s string) (IPRange, error) {
	var r IPRange
	h := strings.IndexByte(s, '-')
	if h == -1 {
		return r, fmt.Errorf("no hyphen in range %q", s)
	}
	from, to := s[:h], s[h+1:]
	var err error
	r.From, err = ParseIP(from)
	if err != nil {
		return r, fmt.Errorf("invalid From IP %q in range %q", from, s)
	}
	r.To, err = ParseIP(to)
	if err != nil {
		return r, fmt.Errorf("invalid To IP %q in range %q", to, s)
	}
	if !r.Valid() {
		return r, fmt.Errorf("range %v to %v not valid", r.From, r.To)
	}
	return r, nil
}

// String returns a string representation of the range.
//
// For a valid range, the form is "From-To" with a single hyphen
// separating the IPs, the same format recognized by
// ParseIPRange.
func (r IPRange) String() string {
	if r.Valid() {
		return fmt.Sprintf("%s-%s", r.From, r.To)
	}
	if r.From.IsZero() || r.To.IsZero() {
		return "zero IPRange"
	}
	return "invalid IPRange"
}

// Valid reports whether r.From and r.To are both non-zero and obey
// the documented requirements: address families match, same IPv6
// zone, and From is less than or equal to To.
func (r IPRange) Valid() bool {
	return !r.From.IsZero() &&
		r.From.z == r.To.z &&
		!r.To.Less(r.From)
}

// Contains reports whether the range r includes addr.
//
// An invalid range always reports false.
func (r IPRange) Contains(addr IP) bool {
	return r.Valid() && r.contains(addr)
}

// contains is like Contains, but without the validity check. For internal use.
func (r IPRange) contains(addr IP) bool {
	return r.From.Compare(addr) <= 0 && r.To.Compare(addr) >= 0
}

// less returns whether r is "before" other. It is before if r.From is
// before other.From, or if they're equal, the shorter range is
// before.
func (r IPRange) less(other IPRange) bool {
	if cmp := r.From.Compare(other.From); cmp != 0 {
		return cmp < 0
	}
	return r.To.Less(other.To)
}

// entirelyBefore returns whether r lies entirely before other in IP
// space.
func (r IPRange) entirelyBefore(other IPRange) bool {
	return r.To.Less(other.From)
}

// entirelyWithin returns whether r is entirely contained within
// other.
func (r IPRange) coveredBy(other IPRange) bool {
	return other.From.lessOrEq(r.From) && r.To.lessOrEq(other.To)
}

// inMiddleOf returns whether r is inside other, but not touching the
// edges of other.
func (r IPRange) inMiddleOf(other IPRange) bool {
	return other.From.Less(r.From) && r.To.Less(other.To)
}

// overlapsStartOf returns whether r entirely overlaps the start of
// other, but not all of other.
func (r IPRange) overlapsStartOf(other IPRange) bool {
	return r.From.lessOrEq(other.From) && r.To.Less(other.To)
}

// overlapsEndOf returns whether r entirely overlaps the end of
// other, but not all of other.
func (r IPRange) overlapsEndOf(other IPRange) bool {
	return other.From.Less(r.From) && other.To.lessOrEq(r.To)
}

// mergeIPRanges returns the minimum and sorted set of IP ranges that
// cover r.
func mergeIPRanges(rr []IPRange) (out []IPRange, valid bool) {
	// Always return a copy of r, to avoid aliasing slice memory in
	// the caller.
	switch len(rr) {
	case 0:
		return nil, true
	case 1:
		return []IPRange{rr[0]}, true
	}

	sort.Slice(rr, func(i, j int) bool { return rr[i].less(rr[j]) })
	out = make([]IPRange, 1, len(rr))
	out[0] = rr[0]
	for _, r := range rr[1:] {
		prev := &out[len(out)-1]
		switch {
		case !r.Valid():
			// Invalid ranges make no sense to merge, refuse to
			// perform.
			return nil, false
		case prev.To.Next() == r.From:
			// prev and r touch, merge them.
			//
			//   prev     r
			// f------tf-----t
			prev.To = r.To
		case prev.To.Less(r.From):
			// No overlap and not adjacent (per previous case), no
			// merging possible.
			//
			//   prev       r
			// f------t  f-----t
			out = append(out, r)
		case prev.To.Less(r.To):
			// Partial overlap, update prev
			//
			//   prev
			// f------t
			//     f-----t
			//        r
			prev.To = r.To
		default:
			// r entirely contained in prev, nothing to do.
			//
			//    prev
			// f--------t
			//  f-----t
			//     r
		}
	}
	return out, true
}

// Overlaps reports whether p and o overlap at all.
//
// If p and o are of different address families or either are invalid,
// it reports false.
func (r IPRange) Overlaps(o IPRange) bool {
	return r.Valid() &&
		o.Valid() &&
		r.From.Compare(o.To) <= 0 &&
		o.From.Compare(r.To) <= 0
}

// prefixMaker returns a address-family-corrected IPPrefix from a and bits,
// where the input bits is always in the IPv6-mapped form for IPv4 addresses.
type prefixMaker func(a uint128, bits uint8) IPPrefix

// Prefixes returns the set of IPPrefix entries that covers r.
//
// If either of r's bounds are invalid, in the wrong order, or if
// they're of different address families, then Prefixes returns nil.
//
// Prefixes necessarily allocates. See AppendPrefixes for a version that uses
// memory you provide.
func (r IPRange) Prefixes() []IPPrefix {
	return r.AppendPrefixes(nil)
}

// AppendPrefixes is an append version of IPRange.Prefixes. It appends
// the IPPrefix entries that cover r to dst.
func (r IPRange) AppendPrefixes(dst []IPPrefix) []IPPrefix {
	if !r.Valid() {
		return nil
	}
	return appendRangePrefixes(dst, r.prefixFrom128AndBits, r.From.addr, r.To.addr)
}

func (r IPRange) prefixFrom128AndBits(a uint128, bits uint8) IPPrefix {
	ip := IP{addr: a, z: r.From.z}
	if r.From.Is4() {
		bits -= 12 * 8
	}
	return IPPrefix{ip, bits}
}

// aZeroBSet is whether, after the common bits, a is all zero bits and
// b is all set (one) bits.
func comparePrefixes(a, b uint128) (common uint8, aZeroBSet bool) {
	common = a.commonPrefixLen(b)

	// See whether a and b, after their common shared bits, end
	// in all zero bits or all one bits, respectively.
	if common == 128 {
		return common, true
	}

	m := mask6[common]
	return common, (a.xor(a.and(m)).isZero() &&
		b.or(m) == uint128{^uint64(0), ^uint64(0)})
}

// Prefix returns r as an IPPrefix, if it can be presented exactly as such.
// If r is not valid or is not exactly equal to one prefix, ok is false.
func (r IPRange) Prefix() (p IPPrefix, ok bool) {
	if !r.Valid() {
		return
	}
	if common, ok := comparePrefixes(r.From.addr, r.To.addr); ok {
		return r.prefixFrom128AndBits(r.From.addr, common), true
	}
	return
}

func appendRangePrefixes(dst []IPPrefix, makePrefix prefixMaker, a, b uint128) []IPPrefix {
	common, ok := comparePrefixes(a, b)
	if ok {
		// a to b represents a whole range, like 10.50.0.0/16.
		// (a being 10.50.0.0 and b being 10.50.255.255)
		return append(dst, makePrefix(a, common))
	}
	// Otherwise recursively do both halves.
	dst = appendRangePrefixes(dst, makePrefix, a, a.bitsSetFrom(common+1))
	dst = appendRangePrefixes(dst, makePrefix, b.bitsClearedFrom(common+1), b)
	return dst
}

// Next returns the IP following ip.
// If there is none, it returns the IP zero value.
func (ip IP) Next() IP {
	ip.addr = ip.addr.addOne()
	if ip.Is4() {
		if uint32(ip.addr.lo) == 0 {
			// Overflowed.
			return IP{}
		}
	} else {
		if ip.addr.isZero() {
			// Overflowed
			return IP{}
		}
	}
	return ip
}

// Prior returns the IP before ip.
// If there is none, it returns the IP zero value.
func (ip IP) Prior() IP {
	if ip.Is4() {
		if uint32(ip.addr.lo) == 0 {
			return IP{}
		}
	} else if ip.addr.isZero() {
		return IP{}
	}
	ip.addr = ip.addr.subOne()
	return ip
}
