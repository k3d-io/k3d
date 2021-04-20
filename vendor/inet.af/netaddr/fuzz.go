// Copyright 2020 The Inet.Af AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build gofuzz

package netaddr

import (
	"fmt"
	"net"
	"strings"
)

func Fuzz(b []byte) int {
	s := string(b)

	ip, err := ParseIP(s)
	if err == nil {
		s2 := ip.String()
		// There's no guarantee that ip.String() will match s.
		// But a round trip the other direction ought to succeed.
		ip2, err := ParseIP(s2)
		if err != nil {
			panic(err)
		}
		if ip2 != ip {
			fmt.Printf("ip=%#v ip2=%#v\n", ip, ip2)
			panic("IP round trip identity failure")
		}
		if s2 != ip2.String() {
			panic("IP String round trip identity failure")
		}
	}
	// Check that we match the standard library's IP parser, modulo zones.
	if !strings.Contains(s, "%") {
		stdip := net.ParseIP(s)
		if ip.IsZero() != (stdip == nil) {
			fmt.Println("stdip=", stdip, "ip=", ip)
			panic("net.ParseIP nil != ParseIP zero")
		} else if !ip.IsZero() && !ip.Is4in6() && ip.String() != stdip.String() {
			fmt.Println("ip=", ip, "stdip=", stdip)
			panic("net.IP.String() != IP.String()")
		}
	}
	// Check that .Next().Prior() and .Prior().Next() preserve the IP.
	if !ip.IsZero() && !ip.Next().IsZero() && ip.Next().Prior() != ip {
		fmt.Println("ip=", ip, ".next=", ip.Next(), ".next.prior=", ip.Next().Prior())
		panic(".Next.Prior did not round trip")
	}
	if !ip.IsZero() && !ip.Prior().IsZero() && ip.Prior().Next() != ip {
		fmt.Println("ip=", ip, ".prior=", ip.Prior(), ".prior.next=", ip.Prior().Next())
		panic(".Prior.Next did not round trip")
	}

	port, err := ParseIPPort(s)
	if err == nil {
		s2 := port.String()
		port2, err := ParseIPPort(s2)
		if err != nil {
			panic(err)
		}
		if port2 != port {
			panic("IPPort round trip identity failure")
		}
		if port2.String() != s2 {
			panic("IPPort String round trip identity failure")
		}
	}

	ipp, err := ParseIPPrefix(s)
	if err == nil {
		s2 := ipp.String()
		ipp2, err := ParseIPPrefix(s2)
		if err != nil {
			panic(err)
		}
		if ipp2 != ipp {
			fmt.Printf("ipp=%#v ipp=%#v\n", ipp, ipp2)
			panic("IPPrefix round trip identity failure")
		}
		if ipp2.String() != s2 {
			panic("IPPrefix String round trip identity failure")
		}
	}

	return 0
}
