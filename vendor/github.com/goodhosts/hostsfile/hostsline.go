package hostsfile

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

// HostsLine represents a line of the hosts file after being parsed into their respective parts
type HostsLine struct {
	IP      string   // IP found at the beginning of the line
	Hosts   []string // Hosts split into a slice on the space char
	Comment string   // Contents of everything after the comment char in the line

	Raw string // Raw contents of the line as parsed in or updated after changes
	Err error  // Used for error checking during parsing
}

const commentChar string = "#"

// NewHostsLine takes a raw line as a string and parses it into a new instance of HostsLine e.g. "192.168.1.1 host1 host2 # comments"
func NewHostsLine(raw string) HostsLine {
	output := HostsLine{Raw: raw}

	if output.HasComment() { //trailing comment
		commentSplit := strings.Split(output.Raw, commentChar)
		raw = commentSplit[0]
		output.Comment = commentSplit[1]
	}

	if output.IsComment() { //whole line is comment
		return output
	}

	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return output
	}

	rawIP := fields[0]
	if net.ParseIP(rawIP) == nil {
		output.Err = fmt.Errorf("bad hosts line: %q", raw)
	}

	output.IP = rawIP
	output.Hosts = fields[1:]

	return output
}

// String to make HostsLine a fmt.Stringer
func (l *HostsLine) String() string {
	return l.ToRaw()
}

// ToRaw returns the HostsLine's contents as a raw string
func (l *HostsLine) ToRaw() string {
	var comment string
	if l.IsComment() { //Whole line is comment
		return l.Raw
	}

	if l.Comment != "" {
		comment = fmt.Sprintf(" %s%s", commentChar, l.Comment)
	}

	return fmt.Sprintf("%s %s%s", l.IP, strings.Join(l.Hosts, " "), comment)
}

// RemoveDuplicateHosts checks all hosts in a line and removes duplicates
func (l *HostsLine) RemoveDuplicateHosts() {
	unique := make(map[string]struct{})
	hosts := make([]string, len(l.Hosts))
	copy(hosts, l.Hosts)

	l.Hosts = []string{}
	for _, host := range hosts {
		if _, ok := unique[host]; !ok {
			unique[host] = struct{}{}
			l.Hosts = append(l.Hosts, host)
		}
	}

	l.RegenRaw()
}

// Deprecated: will be made internal, combines the hosts and comments of two lines together,
func (l *HostsLine) Combine(hostline HostsLine) {
	l.combine(hostline)
}

func (l *HostsLine) combine(hostline HostsLine) {
	l.Hosts = append(l.Hosts, hostline.Hosts...)
	if l.Comment == "" {
		l.Comment = hostline.Comment
	} else {
		l.Comment = fmt.Sprintf("%s %s", l.Comment, hostline.Comment)
	}
	l.RegenRaw()
}

func (l *HostsLine) SortHosts() {
	sort.Strings(l.Hosts)
	l.RegenRaw()
}

func (l *HostsLine) IsComment() bool {
	return strings.HasPrefix(strings.TrimSpace(l.Raw), commentChar)
}

func (l *HostsLine) HasComment() bool {
	return strings.Contains(l.Raw, commentChar)
}

func (l *HostsLine) IsValid() bool {
	return l.IP != ""
}

func (l *HostsLine) IsMalformed() bool {
	return l.Err != nil
}

func (l *HostsLine) RegenRaw() {
	l.Raw = l.ToRaw()
}
