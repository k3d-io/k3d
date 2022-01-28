package hostsfile

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/asaskevich/govalidator"
	"github.com/dimchansky/utfbom"
)

type lookup struct {
	sync.RWMutex
	l map[string][]int
}

type Hosts struct {
	Path  string
	Lines []HostsLine
	ips   lookup
	hosts lookup
}

// NewHosts return a new instance of ``Hosts`` using the default hosts file path.
func NewHosts() (*Hosts, error) {
	osHostsFilePath := os.ExpandEnv(filepath.FromSlash(HostsFilePath))

	if env, isset := os.LookupEnv("HOSTS_PATH"); isset && len(env) > 0 {
		osHostsFilePath = os.ExpandEnv(filepath.FromSlash(env))
	}

	return NewCustomHosts(osHostsFilePath)
}

// NewCustomHosts return a new instance of ``Hosts`` using a custom hosts file path.
func NewCustomHosts(osHostsFilePath string) (*Hosts, error) {
	hosts := &Hosts{
		Path:  osHostsFilePath,
		ips:   lookup{l: make(map[string][]int)},
		hosts: lookup{l: make(map[string][]int)},
	}

	if err := hosts.Load(); err != nil {
		return hosts, err
	}

	return hosts, nil
}

// IsWritable return ```true``` if hosts file is writable.
func (h *Hosts) IsWritable() bool {
	file, err := os.OpenFile(h.Path, os.O_WRONLY, 0660)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}

// Load the hosts file into ```l.Lines```.
// ```Load()``` is called by ```NewHosts()``` and ```Hosts.Flush()``` so you
// generally you won't need to call this yourself.
func (h *Hosts) Load() error {
	file, err := os.Open(h.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	// if you're reloading from disk confirm you refresh the hash maps and lines
	if len(h.Lines) != 0 {
		h.ips = lookup{l: make(map[string][]int)}
		h.hosts = lookup{l: make(map[string][]int)}
		h.Lines = []HostsLine{}
	}

	scanner := bufio.NewScanner(utfbom.SkipOnly(file))
	for scanner.Scan() {
		hl := NewHostsLine(scanner.Text())
		h.Lines = append(h.Lines, hl)
		pos := len(h.Lines) - 1
		h.addIpPosition(hl.IP, pos)
		for _, host := range hl.Hosts {
			h.addHostPositions(host, pos)
		}
	}

	return scanner.Err()
}

// Flush any changes made to hosts file.
func (h *Hosts) Flush() error {
	h.preFlushClean()
	file, err := os.Create(h.Path)
	if err != nil {
		return err
	}

	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range h.Lines {
		if _, err := fmt.Fprintf(w, "%s%s", line.ToRaw(), eol); err != nil {
			return err
		}
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return h.Load()
}

// AddRaw takes a line from a hosts file and parses/adds the HostsLine
func (h *Hosts) AddRaw(raw ...string) error {
	for _, r := range raw {
		nl := NewHostsLine(r)
		if nl.IP != "" && net.ParseIP(nl.IP) == nil {
			return fmt.Errorf("%q is an invalid IP address", nl.IP)
		}

		for _, host := range nl.Hosts {
			if !govalidator.IsDNSName(host) {
				return fmt.Errorf("hostname is not a valid dns name: %s", host)
			}
		}

		h.Lines = append(h.Lines, nl)
		pos := len(h.Lines) - 1
		h.addIpPosition(nl.IP, pos)
		for _, host := range nl.Hosts {
			h.addHostPositions(host, pos)
		}
	}

	return nil
}

// Add an entry to the hosts file.
func (h *Hosts) Add(ip string, hosts ...string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("%q is an invalid IP address", ip)
	}

	position := h.getIpPositions(ip)
	if len(position) == 0 {
		nl := HostsLine{
			Raw:   buildRawLine(ip, hosts),
			IP:    ip,
			Hosts: hosts,
		}
		h.Lines = append(h.Lines, nl)
		pos := len(h.Lines) - 1
		h.addIpPosition(ip, pos)
		for _, host := range nl.Hosts {
			h.addHostPositions(host, pos)
		}
	} else {
		// add new host to the first one we find
		hostsCopy := h.Lines[position[0]].Hosts
		for _, addHost := range hosts {
			if h.Has(ip, addHost) {
				// this combo already exists
				continue
			}

			if !govalidator.IsDNSName(addHost) {
				return fmt.Errorf("hostname is not a valid dns name: %s", addHost)
			}
			if itemInSliceString(addHost, hostsCopy) {
				continue // host exists for ip already
			}

			hostsCopy = append(hostsCopy, addHost)
			h.addHostPositions(addHost, position[0])
		}
		h.Lines[position[0]].Hosts = hostsCopy
		h.Lines[position[0]].Raw = h.Lines[position[0]].ToRaw() // reset raw
	}

	return nil
}

func (h *Hosts) Clear() {
	h.Lines = []HostsLine{}
}

// Clean merge duplicate ips and hosts per ip
func (h *Hosts) Clean() {
	h.RemoveDuplicateIps()
	h.RemoveDuplicateHosts()
	h.SortHosts()
	h.SortByIp()
	h.HostsPerLine(HostsPerLine)
}

// Has return a bool if ip/host combo in hosts file.
func (h *Hosts) Has(ip string, host string) bool {
	ippos := h.getIpPositions(ip)
	hostpos := h.getHostPositions(host)
	for _, pos := range ippos {
		if itemInSliceInt(pos, hostpos) {
			// if ip and host have matching lookup positions we have a combo match
			return true
		}
	}

	return false
}

// HasHostname return a bool if hostname in hosts file.
func (h *Hosts) HasHostname(host string) bool {
	return len(h.getHostPositions(host)) > 0
}

func (h *Hosts) HasIp(ip string) bool {
	return len(h.getIpPositions(ip)) > 0
}

// Remove an entry from the hosts file.
func (h *Hosts) Remove(ip string, hosts ...string) error {
	var outputLines []HostsLine
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("%q is an invalid IP address", ip)
	}

	for _, line := range h.Lines {
		// Bad lines or comments just get readded.
		if line.Err != nil || line.IsComment() || line.IP != ip {
			outputLines = append(outputLines, line)
			continue
		}

		var newHosts []string
		for _, checkHost := range line.Hosts {
			if !itemInSliceString(checkHost, hosts) {
				newHosts = append(newHosts, checkHost)
			}
		}

		// If hosts is empty, skip the line completely.
		if len(newHosts) > 0 {
			newLineRaw := line.IP

			for _, host := range newHosts {
				newLineRaw = fmt.Sprintf("%s %s", newLineRaw, host)
			}
			newLine := NewHostsLine(newLineRaw)
			outputLines = append(outputLines, newLine)
		}
	}

	h.Lines = outputLines
	return nil
}

// RemoveByHostname remove entries by hostname from the hosts file.
func (h *Hosts) RemoveByHostname(host string) error {
	for _, p := range h.getHostPositions(host) {
		line := &h.Lines[p]
		if len(line.Hosts) > 0 {
			line.Hosts = removeFromSliceString(host, line.Hosts)
			line.RegenRaw()
		}
		h.removeHostPositions(host, p)
	}

	return nil
}

func (h *Hosts) RemoveByIp(ip string) error {
	pos := h.getIpPositions(ip)
	for len(pos) > 0 {
		for _, p := range pos {
			h.removeByPosition(p)
		}
	}

	return nil
}

func (h *Hosts) RemoveDuplicateIps() {
	ipCount := make(map[string]int)
	for _, line := range h.Lines {
		ipCount[line.IP]++
	}
	for ip, count := range ipCount {
		if count > 1 {
			h.combineIp(ip)
		}
	}
}

func (h *Hosts) RemoveDuplicateHosts() {
	for pos, line := range h.Lines {
		line.RemoveDuplicateHosts()
		h.Lines[pos] = line
	}
}

func (h *Hosts) SortHosts() {
	for pos, line := range h.Lines {
		line.SortHosts()
		h.Lines[pos] = line
	}
}

// SortByIp convert to net.IP and byte.Compare
func (h *Hosts) SortByIp() {
	sortedIps := make([]net.IP, 0, len(h.Lines))
	for _, l := range h.Lines {
		sortedIps = append(sortedIps, net.ParseIP(l.IP))
	}
	sort.Slice(sortedIps, func(i, j int) bool {
		return bytes.Compare(sortedIps[i], sortedIps[j]) < 0
	})

	var sortedLines []HostsLine
	for _, ip := range sortedIps {
		for _, l := range h.Lines {
			if ip.String() == l.IP {
				sortedLines = append(sortedLines, l)
			}
		}
	}
	h.Lines = sortedLines
}

func (h *Hosts) HostsPerLine(count int) {
	// restacks everything into 1 ip again so we can do the split, do this even if count is -1 so it can reset the slice
	h.RemoveDuplicateIps()
	if count <= 0 {
		return
	}
	var newLines []HostsLine
	for _, line := range h.Lines {
		if len(line.Hosts) <= count {
			newLines = append(newLines, line)
			continue
		}

		for i := 0; i < len(line.Hosts); i += count {
			lineCopy := line
			end := len(line.Hosts)
			if end > i+count {
				end = i + count
			}

			lineCopy.Hosts = line.Hosts[i:end]
			lineCopy.Raw = lineCopy.ToRaw()
			newLines = append(newLines, lineCopy)
		}
	}
	h.Lines = newLines
}

func (h *Hosts) combineIp(ip string) {
	newLine := HostsLine{
		IP: ip,
	}

	linesCopy := make([]HostsLine, len(h.Lines))
	copy(linesCopy, h.Lines)
	for _, line := range linesCopy {
		if line.IP == ip {
			newLine.Combine(line)
		}
	}
	newLine.SortHosts()
	h.removeIp(ip)
	h.Lines = append(h.Lines, newLine)
}

func (h *Hosts) removeByPosition(pos int) {
	if pos == 0 && len(h.Lines) == 1 {
		h.Clear()
		return
	}
	if pos == len(h.Lines) {
		h.Lines = h.Lines[:pos-1]
		return
	}
	h.Lines = append(h.Lines[:pos], h.Lines[pos+1:]...)
}

func (h *Hosts) removeIp(ip string) {
	var newLines []HostsLine
	for _, line := range h.Lines {
		if line.IP != ip {
			newLines = append(newLines, line)
		}
	}

	h.Lines = newLines
}

func (h *Hosts) getHostPositions(host string) []int {
	h.hosts.RLock()
	defer h.hosts.RUnlock()
	i, ok := h.hosts.l[host]
	if ok {
		return i
	}
	return []int{}
}

func (h *Hosts) addHostPositions(host string, pos int) {
	h.hosts.Lock()
	defer h.hosts.Unlock()
	h.hosts.l[host] = append(h.hosts.l[host], pos)
}

func (h *Hosts) removeHostPositions(host string, pos int) {
	h.hosts.Lock()
	defer h.hosts.Unlock()
	positions := h.hosts.l[host]
	h.hosts.l[host] = removeFromSliceInt(pos, positions)
}

func (h *Hosts) getIpPositions(ip string) []int {
	h.ips.RLock()
	defer h.ips.RUnlock()
	i, ok := h.ips.l[ip]
	if ok {
		return i
	}

	return []int{}
}

func (h *Hosts) addIpPosition(ip string, pos int) {
	h.ips.Lock()
	defer h.ips.Unlock()
	h.ips.l[ip] = append(h.ips.l[ip], pos)
}

func buildRawLine(ip string, hosts []string) string {
	output := ip
	for _, host := range hosts {
		output = fmt.Sprintf("%s %s", output, host)
	}

	return output
}
