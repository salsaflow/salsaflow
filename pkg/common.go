package pkg

import (
	// Stdlib
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"

	// Internal
	"github.com/salsaflow/salsaflow/action"
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/fileutil"
	"github.com/salsaflow/salsaflow/log"

	// Other
	"github.com/google/go-github/github"
	"github.com/kardianos/osext"
)

const (
	DefaultGitHubOwner = "salsaflow"
	DefaultGitHubRepo  = "salsaflow"
)

var (
	ErrAborted            = errors.New("aborted by the user")
	ErrInstallationFailed = errors.New("failed to install SalsaFlow")
)

func listReleases(client *github.Client, owner, repo string) ([]github.RepositoryRelease, error) {
	// Set PerPage to 100, which is the maximum.
	listOpts := &github.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	// Loop until all releases are downloaded.
	var releases []github.RepositoryRelease
	for {
		// Fetch another page.
		rs, _, err := client.Repositories.ListReleases(owner, repo, listOpts)
		if err != nil {
			return nil, err
		}
		releases = append(releases, rs...)

		// In case the page is not full, this is the last page.
		if len(rs) != 100 {
			return releases, nil
		}

		// Increment the page number.
		listOpts.Page += 1
	}
}

// doInstall performs the common step that both install and upgrade need to do.
//
// Given a GitHub release, it downloads and unpacks the fitting artifacts
// and replaces the current executables with the ones just downloaded.
func doInstall(
	client *github.Client,
	owner string,
	repo string,
	assets []github.ReleaseAsset,
	version string,
	dstDir string,
) (err error) {

	// Choose the asset to be downloaded.
	task := "Pick the most suitable release asset"
	var (
		assetName = getAssetName(version)
		assetURL  string
	)
	for _, asset := range assets {
		if *asset.Name == assetName {
			assetURL = *asset.BrowserDownloadURL
		}
	}
	if assetURL == "" {
		return errs.NewError(task, errors.New("no suitable release asset found"))
	}

	// Make sure the destination folder exists.
	task = "Make sure the destination directory exists"
	dstDir, act, err := ensureDstDirExists(dstDir)
	if err != nil {
		return errs.NewError(task, err)
	}
	defer action.RollbackOnError(&err, act)

	// Download the selected release asset.
	return downloadAndInstallAsset(assetName, assetURL, dstDir)
}

func getAssetName(version string) string {
	return fmt.Sprintf("salsaflow-%v-%v-%v.zip", version, runtime.GOOS, runtime.GOARCH)
}

func ensureDstDirExists(dstDir string) (string, action.Action, error) {
	// In case dst is empty, use the location of the current executable.
	// In that case the directory obviously already exists.
	if dstDir == "" {
		dstDir, err := osext.ExecutableFolder()
		return dstDir, action.Noop, err
	}

	// Make sure the path exists and is a directory.
	act, err := fileutil.EnsureDirectoryExists(dstDir)
	if err != nil {
		return "", nil, err
	}
	return dstDir, act, nil
}

func downloadAndInstallAsset(assetName, assetURL, dstDir string) error {
	// Download the asset.
	task := "Download " + assetName
	log.Run(task)
	resp, err := http.Get(assetURL)
	if err != nil {
		return errs.NewError(task, err)
	}
	defer resp.Body.Close()

	// Unpack the asset (in-memory).
	// We keep the asset in the memory since it is never going to be that big.
	task = "Read the asset into an internal buffer"
	var capacity = resp.ContentLength
	if capacity == -1 {
		capacity = 0
	}
	bodyBuffer := bytes.NewBuffer(make([]byte, 0, capacity))
	_, err = io.Copy(bodyBuffer, resp.Body)
	if err != nil {
		return errs.NewError(task, err)
	}

	task = "Replace SalsaFlow executables"
	archive, err := zip.NewReader(bytes.NewReader(bodyBuffer.Bytes()), int64(bodyBuffer.Len()))
	if err != nil {
		return errs.NewError(task, err)
	}

	var numThreads int
	errCh := make(chan errs.Err, len(archive.File))

	// Uncompress all the executables in the archive and move them into place.
	// This part replaces the current executables with new ones just downloaded.
	for _, file := range archive.File {
		if file.CompressedSize64 == 0 {
			continue
		}

		numThreads++

		go func(file *zip.File) {
			baseName := filepath.Base(file.Name)
			task := fmt.Sprintf("Uncompress executable '%v'", baseName)
			log.Go(task)

			src, err := file.Open()
			if err != nil {
				errCh <- errs.NewError(task, err)
				return
			}

			task = fmt.Sprintf("Move executable '%v' into place", baseName)
			log.Go(task)
			if err := replaceExecutable(src, dstDir, baseName); err != nil {
				src.Close()
				errCh <- errs.NewError(task, err)
				return
			}

			src.Close()
			errCh <- nil
		}(file)
	}

	task = "install given SalsaFlow package"
	var ex error
	for i := 0; i < numThreads; i++ {
		if err := <-errCh; err != nil {
			errs.Log(err)
			ex = errs.NewError(task, ErrInstallationFailed)
		}
	}
	return ex
}
