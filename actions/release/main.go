package main

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/actions-go/toolkit/core"
	gha "github.com/actions-go/toolkit/github"
	"github.com/google/go-github/v42/github"
)

func addUpload(uploads map[string][]string, path string) {
	name := filepath.Base(path)
	u, ok := uploads[name]
	if !ok {
		u = []string{}
	}
	u = append(u, path)
	core.Debugf("found %s asset to upload from %s", name, path)
	uploads[name] = u
}

var (
	checksums = map[string]func() hash.Hash{
		".sha1":   func() hash.Hash { return sha1.New() },
		".sha256": func() hash.Hash { return sha256.New() },
		".md5":    func() hash.Hash { return md5.New() },
	}
)

// allow hooking for test purposes
var uploadReleaseAsset = func(name, label string, release *github.RepositoryRelease, fd *os.File) error {
	mediaType := mime.TypeByExtension(filepath.Ext(fd.Name()))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	core.Infof("uploading asset %s from file %s (mime: %s)", name, fd.Name(), mediaType)
	_, _, err := gha.GitHub.Repositories.UploadReleaseAsset(
		context.Background(),
		gha.Context.Repo.Owner,
		gha.Context.Repo.Repo,
		*release.ID,
		&github.UploadOptions{
			Name:      name,
			MediaType: mediaType,
			Label:     label,
		},
		fd,
	)
	return err
}

func uploadAsset(name string, release *github.RepositoryRelease, path string) error {
	fd, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("unable to open file to upload %s: %v", path, err)
	}
	defer fd.Close()
	err = uploadReleaseAsset(name, name, release, fd)
	if err != nil {
		return fmt.Errorf("failed to upload %s to release %s (%s): %v", name, *release.Name, *release.URL, err)
	}
	for ext, hasher := range checksums {
		h := hasher()
		fd, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to read release asset %s: %v", path, err)
		}
		defer fd.Close()
		_, err = io.Copy(h, fd)
		if err != nil {
			return err
		}
		tmpfile, err := ioutil.TempFile("", "checksum*"+ext)
		if err != nil {
			return err
		}
		defer func(tmpfile *os.File) {
			core.Infof("deleting temp file %s", tmpfile.Name())
			tmpfile.Close()
			os.Remove(tmpfile.Name())
		}(tmpfile)

		sum := hex.EncodeToString(h.Sum(nil))
		core.Infof("asset %s(%s) has %s sum %s", name, path, ext[1:], sum)
		_, err = tmpfile.Write([]byte(sum))
		if err != nil {
			return err
		}
		_, err = tmpfile.Seek(0, 0)
		if err != nil {
			return err
		}
		err = uploadReleaseAsset(name+ext, name+ext, release, tmpfile)
		if err != nil {
			return fmt.Errorf("failed to upload %s to release %s (%s): %v", name, *release.Name, *release.URL, err)
		}
	}
	return nil
}

func getReleaseChangeLog(path, tag string) string {
	changeLog := ""
	fd, err := os.Open(path)
	if err != nil {
		core.Warningf("failed to open CHANGELOG.md: %v. Won't create release body", err)
	} else {
		scanner := bufio.NewScanner(fd)
		dumpReleaseNotes := false
		for scanner.Scan() {
			line := scanner.Text()
			if dumpReleaseNotes {
				if strings.HasPrefix(line, "## ") {
					return strings.TrimSpace(changeLog)
				}
				changeLog += line + "\n"
			}
			if strings.HasPrefix(line, "## "+tag) {
				dumpReleaseNotes = true
			}
		}
	}
	return changeLog
}

func main() {
	// Create or update release
	tag, ok := core.GetInput("tag_name")
	if !ok && gha.Context.Payload.PushEvent != nil && gha.Context.Payload.PushEvent.Ref != nil {
		if strings.HasPrefix(*gha.Context.Payload.Ref, "refs/tags/") {
			tag = *gha.Context.Payload.Ref
		}
	}
	tag = strings.TrimPrefix(tag, "refs/tags/")

	targetCommitish, ok := core.GetInput("target_commitish")
	if !ok {
		targetCommitish = "master"
	}
	isDraft, ok := core.GetInput("draft")
	if !ok {
		isDraft = "true"
	}
	isPreRelease, ok := core.GetInput("pre_release")
	if !ok {
		isPreRelease = "true"
	}
	var err error
	var release *github.RepositoryRelease

	releaseName := fmt.Sprintf("%s %s", tag, time.Now().Format("2006-01-02"))

	releaseChangeLog := getReleaseChangeLog("CHANGELOG.md", tag)

	core.Group("upserting the release", func() {
		release, _, err = gha.GitHub.Repositories.GetReleaseByTag(context.Background(), gha.Context.Repo.Owner, gha.Context.Repo.Repo, tag)
		if err != nil {
			core.Infof("Did not find release to update, creating a new one")
			release, _, err = gha.GitHub.Repositories.CreateRelease(context.Background(), gha.Context.Repo.Owner, gha.Context.Repo.Repo, &github.RepositoryRelease{
				Name:            github.String(releaseName),
				TagName:         github.String(tag),
				TargetCommitish: github.String(targetCommitish),
				Body:            github.String(releaseChangeLog),
				Draft:           github.Bool(isDraft == "true"),
				Prerelease:      github.Bool(isPreRelease == "true"),
			})
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			core.Infof("Found existing release. Updating it with latest details")
			release.Body = github.String(releaseChangeLog)
			release.Name = github.String(releaseName)
			release, _, err = gha.GitHub.Repositories.EditRelease(context.Background(), gha.Context.Repo.Owner, gha.Context.Repo.Repo, *release.ID, release)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
		core.SetOutput("release_id", fmt.Sprintf("%d", *release.ID))
		core.SetOutput("release_url", *release.URL)
		core.SetOutput("release_upload_url", *release.UploadURL)
		core.SetOutput("release_htmlurl", *release.HTMLURL)
	})()

	assetsFolder := core.GetInputOrDefault("folder", ".")
	uploads := map[string][]string{}
	core.Group(fmt.Sprintf("finding assets in %s", assetsFolder), func() {
		if err := filepath.Walk(assetsFolder, func(path string, info os.FileInfo, err error) error {
			if info == nil || path == "" || info.IsDir() {
				return nil
			}
			addUpload(uploads, path)
			return nil
		}); err != nil {
			core.SetFailed(fmt.Sprintf("failed to list all files: %v", err))
			os.Exit(1)
		}
	})()

	if len(release.Assets) > 0 {
		core.Group("Cleaning previous release", func() {
			for _, asset := range release.Assets {
				core.Infof("Removing previously existing asset %s from release %s", *asset.Name, *release.Name)
				_, err := gha.GitHub.Repositories.DeleteReleaseAsset(context.Background(),
					gha.Context.Repo.Owner,
					gha.Context.Repo.Repo,
					*asset.ID,
				)
				if err != nil {
					core.SetFailed(fmt.Sprintf("failed to delete release asset %d: %v", *asset.ID, err))
					os.Exit(1)
				}
			}
		})()
	}

	for name, paths := range uploads {
		switch len(paths) {
		case 0:
		case 1:
			core.Group(fmt.Sprintf("uploading release asset %s", name), func() {
				err := uploadAsset(name, release, paths[0])
				if err != nil {

					core.SetFailed(err.Error())
					os.Exit(1)
				}
			})()
		default:
			core.SetFailed(fmt.Sprintf("Unsupported multiple files with name %s: %v", name, paths))
			os.Exit(1)
		}
	}

	core.AddStepSummary(fmt.Sprintf("# Release created\n\nTag: %s\nURL: [%s](%s)", tag, *release.HTMLURL, *release.HTMLURL))
}

func init() {
	for digest := range checksums {
		if err := mime.AddExtensionType(digest, "text/plain"); err != nil {
			panic(err)
		}
	}
}
