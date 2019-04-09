package version

// Version is the string that contains version
var Version string

// GetVersion returns the version for cli, it gets it from "git describe --tags" or returns "dev" when doing simple go build
func GetVersion() string {
	if len(Version) == 0 {
		return "dev"
	}
	return Version
}
