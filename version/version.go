package version

// Version is the string that contains version
var Version string

// K3sVersion contains the latest version tag of K3s
var K3sVersion = "v0.4.0"

// GetVersion returns the version for cli, it gets it from "git describe --tags" or returns "dev" when doing simple go build
func GetVersion() string {
	if len(Version) == 0 {
		return "dev"
	}
	return Version
}

// GetK3sVersion returns the version string for K3s
func GetK3sVersion() string {
	return K3sVersion
}
