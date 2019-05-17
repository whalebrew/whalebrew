package version

import (
	"testing"

	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
)

func TestCheckCompatible(t *testing.T) {
	parsedVersion = semver.MustParse("0.1.1")
	assert.Error(t, CheckCompatible(">1.0.0"))
	assert.NoError(t, CheckCompatible("<1.0.0"))
	assert.NoError(t, CheckCompatible("<1.0.0 >0.1.0"))
	assert.Error(t, CheckCompatible("not a semver spec"))
	parsedVersion = semver.MustParse("0.1.0")
	assert.Error(t, CheckCompatible(">0.1.0 <1.0.0"))
}
