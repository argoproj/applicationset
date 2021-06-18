package common

// version information populated during build time
var (
	version   = "" // value from VERSION file
	gitCommit = "" // output of git rev-parse HEAD
)

// Version of the applicationset controller
type Version struct {
	Version   string
	GitCommit string
}

// GetVersion returns the version of the applicationset controller
func GetVersion() Version {
	return Version{
		Version:   "v" + version,
		GitCommit: gitCommit,
	}
}
