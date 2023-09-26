package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/actions-go/toolkit/core"
	"github.com/spf13/afero"
	"github.com/whalebrew/whalebrew/actions/release/pkg/bump"
)

var (
	exit        = os.Exit
	fs          = afero.NewOsFs()
	gitWorktree = "."
	now         = time.Now
	envHandler  = newHandler()
)

func newHandler() handler {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return ghaHandler{}
	}
	return &localHandler{
		outputs: map[string]string{},
	}
}

type handler interface {
	GetInput(name string) (string, bool)
	Errorf(format string, args ...interface{})
	SetOutput(name, value string)
	Done()
}

type ghaHandler struct{}

func (ghaHandler) GetInput(name string) (string, bool) {
	return core.GetInput(name)
}
func (ghaHandler) Errorf(format string, args ...interface{}) {
	core.Errorf(format, args...)
}
func (ghaHandler) SetOutput(name, value string) {
	core.SetOutput(name, value)
}

func (ghaHandler) Done() {

}

type localHandler struct {
	outputs map[string]string
}

func (l *localHandler) GetInput(name string) (string, bool) {
	if len(os.Args) < 2 {
		return "", false
	}
	return os.Args[1], true
}

func (l *localHandler) Errorf(format string, args ...interface{}) {
	fmt.Println("error:", fmt.Sprintf(format, args...))
}

func (l *localHandler) SetOutput(name, value string) {
	if l.outputs == nil {
		l.outputs = map[string]string{}
	}
	l.outputs[name] = value
}

func (l *localHandler) Done() {
	if len(l.outputs) > 0 {
		fmt.Println("outputs:")
		for k, v := range l.outputs {
			fmt.Println("  ", k, ":", v)
		}
	}
}

func git(args ...string) error {
	c := exec.Command("git", "-C", gitWorktree)
	c.Args = append(c.Args, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func headSha() (string, error) {
	c := exec.Command("git", "-C", gitWorktree, "rev-parse", "HEAD")
	b := bytes.Buffer{}
	c.Stdout = &b
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(b.String()), nil
}

func main() {

	version, ok := envHandler.GetInput("version")
	if !ok {
		envHandler.Errorf(`Unable to find version "github actioninput"`)
		exit(1)
	}
	v, err := semver.NewVersion(version)
	if err != nil {
		envHandler.Errorf("Version %v is invalid: %v, it must follow the semver layout", version, err)
		exit(1)
	}

	if v.Metadata() != "" {
		envHandler.Errorf("Released version %v must not unclude build metadata", version)
		exit(1)
	}

	versionWithDate := fmt.Sprintf("%s - %s", version, now().Format("2006-01-02"))

	inTree, err := v.SetMetadata("from-sources")
	if err != nil {
		envHandler.Errorf("Failed to set build metadata to the version %v: %v", version, err)
		exit(1)
	}

	releaseMessage, err := bump.ExtractReleaseMessage(fs)
	if err != nil {
		envHandler.Errorf("Failed to extract release message: %v", err)
		exit(1)
	}

	// Generate the relevant version update
	err = bump.BumpInTreeVersion(fs, inTree.String())
	if err != nil {
		envHandler.Errorf("unable to bump in-tree version")
		exit(1)
	}
	if v.Prerelease() == "" {
		err = bump.BumpREADMEVersion(fs, version)
		if err != nil {
			envHandler.Errorf("unable to bump README version")
			exit(1)
		}
	}
	err = bump.ReleaseChangeLog(fs, versionWithDate)
	if err != nil {
		envHandler.Errorf("unable to release changelog")
		exit(1)
	}
	err = git("add", "-u")
	if err != nil {
		envHandler.Errorf("unable to update git repository")
		exit(1)
	}
	err = git("commit", "-m", "Release version "+v.String())
	if err != nil {
		envHandler.Errorf("unable to update git repository")
		exit(1)
	}
	sha, err := headSha()
	if err != nil {
		envHandler.Errorf("failed to get head SHA")
		exit(1)
	}
	envHandler.SetOutput("release_sha", sha)

	err = git("tag", "-a", v.String(), "-m", versionWithDate, "-m", releaseMessage)
	if err != nil {
		envHandler.Errorf("unable to tag new release")
		exit(1)
	}
	envHandler.SetOutput("release_tag", v.String())

	if v.Prerelease() == "" {
		*v = v.IncPatch()
		fmt.Println(v.String())
	}
	*v, err = v.SetPrerelease("dev")
	if err != nil {
		envHandler.Errorf("unable to set pre-release for the new version")
		exit(1)
	}

	err = bump.BumpInTreeVersion(fs, v.String())
	if err != nil {
		envHandler.Errorf("unable to bump in-tree version")
		exit(1)
	}
	err = bump.StartUnreleased(fs)
	if err != nil {
		envHandler.Errorf("unable to open the change log for the new version")
		exit(1)
	}

	err = git("add", "-u")
	if err != nil {
		envHandler.Errorf("unable to update git repository")
		exit(1)
	}
	err = git("commit", "-m", "Open development for version "+v.String())
	if err != nil {
		envHandler.Errorf("unable to update git repository")
		exit(1)
	}
	sha, err = headSha()
	if err != nil {
		envHandler.Errorf("failed to get head SHA")
		exit(1)
	}
	envHandler.SetOutput("dev_sha", sha)
}
