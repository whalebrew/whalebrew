package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-github/v29/github"
	"github.com/stretchr/testify/assert"
)

func TestUploadAssetReutrnsErrorWhenFileDoesNotExist(t *testing.T) {
	release := &github.RepositoryRelease{}
	uploadReleaseAsset = func(name, label string, release *github.RepositoryRelease, fd *os.File) error {
		t.Errorf("no asset should be uploaded if the path does not exist")
		return nil
	}
	assert.Error(t, uploadAsset("hello-world", release, "./not-exist"))
}
func TestUploadAssetSucceeds(t *testing.T) {
	release := &github.RepositoryRelease{}
	contents := map[string]string{
		"hello-world":        "test\n",
		"hello-world.md5":    "d8e8fca2dc0f896fd7cb4cb0031ba249",
		"hello-world.sha1":   "4e1243bd22c66e76c2ba9eddc1f91394e57f9f83",
		"hello-world.sha256": "f2ca1bb6c7e907d06dafe4687e579fce76b37e4e93b7605022da52e6ccc26fd2",
	}
	openedFiles := []*os.File{}
	uploadReleaseAsset = func(name, label string, release *github.RepositoryRelease, fd *os.File) error {
		b, err := ioutil.ReadAll(fd)
		assert.NoError(t, err)
		assert.Equal(t, contents[name], string(b))
		delete(contents, name)
		openedFiles = append(openedFiles, fd)
		return nil
	}
	path := "./resources/test"
	assert.NoError(t, uploadAsset("hello-world", release, path))
	assert.Len(t, contents, 0)
	for _, f := range openedFiles {
		_, err := f.Seek(0, 0)
		assert.Errorf(t, err, "all files should be closed after leaving the upload function")
		if f.Name() != path {
			_, err := os.Stat(f.Name())
			assert.Truef(t, os.IsNotExist(err), "Temporary checksum files should be removed after leafing the upload function")

		}
		fmt.Println(f.Name())
	}
}

func TestMultipleAssetsCanBeUploaded(t *testing.T) {
	release := &github.RepositoryRelease{}
	uploadReleaseAsset = func(name, label string, release *github.RepositoryRelease, fd *os.File) error {
		return nil
	}
	assert.NoError(t, uploadAsset("test", release, "./resources/test"))
	assert.NoError(t, uploadAsset("main", release, "./main.go"))
	assert.NoError(t, uploadAsset("main_test", release, "./main_test.go"))
}
