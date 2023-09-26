package bump_test

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/whalebrew/whalebrew/actions/release/pkg/bump"
)

var (
	versionFileContentPattern = `package version

	var (
		Version = "%s"
	)
`
	readmeContentPattern = `# title
download link:
[click here](https://github.com/org/repo/releases/download/%s/download)
`

	changeLogContentPattern = `# title

## %s

### Some
 * stuff
`
)

func testActualVersionBump(t *testing.T, bump func(fs afero.Fs, version string) error, path string) {
	fs := afero.NewMemMapFs()
	fd, err := fs.Create(path)
	require.NoError(t, err)

	originalData, err := os.ReadFile("../../../../" + path)
	require.NoError(t, err)
	fd.Write(originalData)
	fd.Close()

	assert.NoError(t, bump(fs, "5.6.7-my-test-version-bump+local"))

	fd, err = fs.Open(path)
	require.NoError(t, err)
	data, err := io.ReadAll(fd)
	require.NoError(t, err)

	assert.Contains(t, string(data), "5.6.7-my-test-version-bump+local")

	dataAfterTest, err := os.ReadFile("../../../../" + path)
	require.NoError(t, err)
	assert.Equal(t, string(originalData), string(dataAfterTest))
}

func TestInTreeVersionBump(t *testing.T) {
	fs := afero.NewMemMapFs()

	fd, err := fs.Create("version/version.go")
	require.NoError(t, err)
	fd.Write([]byte(fmt.Sprintf(versionFileContentPattern, "whatever")))
	fd.Close()

	assert.NoError(t, bump.BumpInTreeVersion(fs, "1.2.3"))

	fd, err = fs.Open("version/version.go")
	require.NoError(t, err)
	data, err := io.ReadAll(fd)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf(versionFileContentPattern, "1.2.3"), string(data))
}

func TestActualInTreeVersionBump(t *testing.T) {
	testActualVersionBump(t, bump.BumpInTreeVersion, "version/version.go")
}

func TestReadmeVersionBump(t *testing.T) {
	fs := afero.NewMemMapFs()

	fd, err := fs.Create("README.md")
	require.NoError(t, err)
	fd.Write([]byte(fmt.Sprintf(readmeContentPattern, "whatever")))
	fd.Close()

	assert.NoError(t, bump.BumpREADMEVersion(fs, "1.2.3"))

	fd, err = fs.Open("README.md")
	require.NoError(t, err)
	data, err := io.ReadAll(fd)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf(readmeContentPattern, "1.2.3"), string(data))
}

func TestActualREADMEVersionBump(t *testing.T) {
	testActualVersionBump(t, bump.BumpREADMEVersion, "README.md")
}

func TestReleaseChangeLog(t *testing.T) {
	fs := afero.NewMemMapFs()

	fd, err := fs.Create("CHANGELOG.md")
	require.NoError(t, err)
	fd.Write([]byte(fmt.Sprintf(changeLogContentPattern, "Unreleased")))
	fd.Close()

	assert.NoError(t, bump.ReleaseChangeLog(fs, "1.2.3"))

	fd, err = fs.Open("CHANGELOG.md")
	require.NoError(t, err)
	data, err := io.ReadAll(fd)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf(changeLogContentPattern, "1.2.3"), string(data))
}

func TestActualReleaseChangeLog(t *testing.T) {
	testActualVersionBump(t, bump.ReleaseChangeLog, "CHANGELOG.md")
}

func TestOpenUnreleased(t *testing.T) {
	fs := afero.NewMemMapFs()

	fd, err := fs.Create("CHANGELOG.md")
	require.NoError(t, err)
	fd.Write([]byte(fmt.Sprintf(changeLogContentPattern, "0.0.0")))
	fd.Close()

	assert.NoError(t, bump.StartUnreleased(fs))

	fd, err = fs.Open("CHANGELOG.md")
	require.NoError(t, err)
	data, err := io.ReadAll(fd)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf(changeLogContentPattern, "Unreleased\n\n## 0.0.0"), string(data))
}
