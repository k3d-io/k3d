package hostsfile

var (
	HostsPerLine  = 9
	HostsFilePath = "${SystemRoot}/System32/drivers/etc/hosts"
	eol           = "\r\n"
)

func (h *Hosts) preFlush() error {
	// need to force hosts per line always on windows see https://github.com/goodhosts/hostsfile/issues/18
	h.HostsPerLine(HostsPerLine)
	return nil
}

func (h *Hosts) postFlush() error { return nil }
