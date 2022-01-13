//go:build !windows
//+build !windows

package hostsfile

var (
	HostsPerLine  = -1 // unlimited
	HostsFilePath = "/etc/hosts"
	eol           = "\n"
)

func (h *Hosts) preFlushClean() {} // no op
