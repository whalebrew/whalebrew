//go:build darwin && arm64

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultInstallDirWithMacOSARM64(t *testing.T) {
	assert.Equal(t, "/opt/whalebrew/bin", defaultInstallDir())
}
