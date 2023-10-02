//go:build !darwin || !arm64

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultInstallDirWithNotMacOSARM64(t *testing.T) {
	assert.Equal(t, "/usr/local/bin", defaultInstallDir())
}
