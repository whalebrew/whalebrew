package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func redirectReaderTo(r io.ReadCloser, w io.Writer, c chan error) {
	b := bufio.NewWriter(w)
	_, err := b.ReadFrom(r)
	c <- err
}

func run(path string, arg ...string) error {
	cmd := exec.Command(path, arg...)
	sync := make(chan error)
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go redirectReaderTo(stdOut, os.Stdout, sync)
	go redirectReaderTo(stdErr, os.Stdout, sync)
	err = cmd.Run()
	if err != nil {
		return err
	}
	// wait for the completion of stdout and stderr goroutines
	// and report errors if any
	err = <-sync
	if err != nil {
		return err
	}

	err = <-sync
	if err != nil {
		return err
	}
	return nil
}

// Repo represents a git repository
type Repo struct {
	root string
	git  string
}

// NewRepo creates a new repository handler
func NewRepo(p string) *Repo {
	p, err := filepath.EvalSymlinks(p)
	if err != nil {
		return nil
	}
	git, err := exec.LookPath("git")
	if err != nil {
		// If the user has no git, don't fail installing the package
		return nil
	}
	cmd := exec.Command(git, "-C", path.Dir(p), "rev-parse", "--show-toplevel")

	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil
	}
	gitDir := strings.TrimSpace(out.String())
	return &Repo{gitDir, git}
}

// Add the provided path to git and commits the change
func (r *Repo) Add(filePath string) error {
	if r == nil {
		return nil
	}
	relPath, err := r.getRelPath(filePath)
	err = r.run("add", relPath)
	if err != nil {
		return err
	}
	return r.commit(fmt.Sprintf("Install %s with whalebrew", relPath))
}

// Rm removes a path from the git index and commits the change
func (r *Repo) Rm(filePath string) error {
	if r == nil {
		return nil
	}
	relPath, err := r.getRelPath(filePath)
	err = r.run("rm", "-q", relPath)
	if err != nil {
		return err
	}
	return r.commit(fmt.Sprintf("Remove %s with whalebrew", relPath))
}

func (r *Repo) commit(message string) error {
	return r.run("commit", "-m", message)
}

func (r *Repo) run(command string, arg ...string) error {
	arg = append([]string{"-C", r.root, command}, arg...)
	return run(r.git, arg...)
}

func (r *Repo) getRelPath(filePath string) (string, error) {
	filePath, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(r.root, filePath)
	if err != nil {
		return "", err
	}
	return relPath, nil
}
