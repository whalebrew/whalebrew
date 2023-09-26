package version

import (
	"fmt"

	"github.com/blang/semver/v4"
)

// Version is the current Whalebrew version
// This flag is an interface for some builds like in homebrew:
// https://github.com/Homebrew/homebrew-core/pull/144615
var Version = "0.4.2-test+from-sources"

var parsedVersion = semver.MustParse(Version)

type incompatibleVersion struct {
	expected string
	current  string
}

func (iv incompatibleVersion) Error() string {
	return fmt.Sprintf("current whalebrew version %s is incompatible with range %s", iv.current, iv.expected)
}

// CheckCompatible returns nil if the current whalebrew version is compatible with the specifications
// otherwise, an error is returned
func CheckCompatible(versionSpec string) error {
	r, err := semver.ParseRange(versionSpec)
	if err != nil {
		return err
	}
	if r(parsedVersion) {
		return nil
	}
	return incompatibleVersion{
		expected: versionSpec,
		current:  Version,
	}
}
