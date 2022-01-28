package hostsfile

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

type HostsLine struct {
	IP      string
	Hosts   []string
	Raw     string
	Err     error
	Comment string
}

const commentChar string = "#"

// NewHostsLine return a new instance of ```HostsLine```.
func NewHostsLine(raw string) HostsLine {
	output := HostsLine{Raw: raw}
	if output.IsComment() { //whole line is comment
		return output
	}

	if output.HasComment() { //trailing comment
		commentSplit := strings.Split(output.Raw, commentChar)
		raw = commentSplit[0]
		output.Comment = commentSplit[1]
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

func (l *HostsLine) RemoveDuplicateHosts() {
	unique := make(map[string]struct{})
	for _, h := range l.Hosts {
		unique[h] = struct{}{}
	}

	l.Hosts = []string{}
	for k := range unique {
		l.Hosts = append(l.Hosts, k)
	}
	l.Raw = l.ToRaw()
}

func (l *HostsLine) Combine(hostline HostsLine) {
	l.Hosts = append(l.Hosts, hostline.Hosts...)
	if l.Comment == "" {
		l.Comment = hostline.Comment
	} else {
		l.Comment = fmt.Sprintf("%s %s", l.Comment, hostline.Comment)
	}
	l.Raw = l.ToRaw()
}

func (l *HostsLine) SortHosts() {
	sort.Strings(l.Hosts)
	l.Raw = l.ToRaw()
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
	l.Raw = fmt.Sprintf("%s %s", l.IP, strings.Join(l.Hosts, " "))
}
