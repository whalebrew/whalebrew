package bump

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/afero"
)

var (
	goVersionBump     = regexp.MustCompilePOSIX(`(Version[ ]*=[ ]*")[^"]*(")`)
	readMeVersionBump = regexp.MustCompilePOSIX(`(releases/download/)[^/]*(/)`)
	changeLogBump     = regexp.MustCompile(`(##[ ]*)Unreleased([ ]*)`)
	unrealeased       = "\n## Unreleased\n"
)

func mutateContent(fs afero.Fs, path string, mutate func(data []byte) []byte) error {
	fd, err := fs.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(fd)
	if err != nil {
		return err
	}
	_, err = fd.Seek(0, 0)
	if err != nil {
		return err
	}
	err = fd.Truncate(0)
	if err != nil {
		return err
	}
	_, err = fd.Write(mutate(data))
	if err != nil {
		return err
	}
	return nil
}

func bumpVersionInFile(fs afero.Fs, re *regexp.Regexp, path, newVersion string) error {
	return mutateContent(fs, path, func(data []byte) []byte {
		return re.ReplaceAll(data, []byte(fmt.Sprintf("${1}%s${2}", newVersion)))
	})
}

func BumpInTreeVersion(fs afero.Fs, newVersion string) error {
	return bumpVersionInFile(fs, goVersionBump, "version/version.go", newVersion)
}

func BumpREADMEVersion(fs afero.Fs, newVersion string) error {
	return bumpVersionInFile(fs, readMeVersionBump, "README.md", newVersion)
}
func ReleaseChangeLog(fs afero.Fs, newVersion string) error {
	return bumpVersionInFile(fs, changeLogBump, "CHANGELOG.md", newVersion)
}

func StartUnreleased(fs afero.Fs) error {
	return mutateContent(fs, "CHANGELOG.md", func(data []byte) []byte {
		content := bytes.SplitN(data, []byte{'\n'}, 2)
		data = content[0]
		data = append(data, '\n')
		data = append(data, []byte(unrealeased)...)
		if len(content) > 1 {
			data = append(data, content[1]...)
		}
		return data
	})
}

func ExtractReleaseMessage(fs afero.Fs) (string, error) {
	fd, err := fs.Open("CHANGELOG.md")
	if err != nil {
		return "", err
	}
	defer fd.Close()
	scan := bufio.NewScanner(fd)
	inUnreleased := false
	r := ""
	for scan.Scan() {
		line := scan.Text()
		if strings.HasPrefix(line, strings.TrimSpace(unrealeased)) {
			inUnreleased = true
		} else if inUnreleased {
			if strings.HasPrefix(line, "## ") {
				inUnreleased = false
			} else {
				r = r + line + "\n"
			}
		}
	}
	return r, nil
}
