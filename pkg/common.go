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
	"github.com/salsaflow/salsaflow/errs"
	"github.com/salsaflow/salsaflow/log"

	// Other
	"github.com/google/go-github/github"
	"github.com/kardianos/osext"
)

var ErrAborted = errors.New("aborted by the user")

// doInstall performs the common step that both install and upgrade need to do.
//
// Given a GitHub release, it downloads and unpacks the fitting artifacts
// and replaces the current executables with the ones just downloaded.
func doInstall(client *github.Client, owner, repo string, assets []github.ReleaseAsset, version string) error {
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
		return errs.NewError(task, errors.New("no suitable release asset found"), nil)
	}

	// Download the selected release asset.
	return downloadAndInstallAsset(assetName, assetURL)
}

func getAssetName(version string) string {
	return fmt.Sprintf("salsaflow-%v-%v-%v.zip", version, runtime.GOOS, runtime.GOARCH)
}

func downloadAndInstallAsset(assetName, assetURL string) error {
	// Download the asset.
	task := "Download " + assetName
	log.Run(task)
	resp, err := http.Get(assetURL)
	if err != nil {
		return errs.NewError(task, err, nil)
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
		return errs.NewError(task, err, nil)
	}

	task = "Replace SalsaFlow executables"
	archive, err := zip.NewReader(bytes.NewReader(bodyBuffer.Bytes()), int64(bodyBuffer.Len()))
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	exeDir, err := osext.ExecutableFolder()
	if err != nil {
		return errs.NewError(task, err, nil)
	}

	var numThreads int
	errCh := make(chan *errs.Error, len(archive.File))

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
				errCh <- errs.NewError(task, err, nil)
				return
			}

			task = fmt.Sprintf("Move executable '%v' into place", baseName)
			log.Go(task)
			if err := replaceExecutable(src, exeDir, baseName); err != nil {
				src.Close()
				errCh <- errs.NewError(task, err, nil)
				return
			}

			src.Close()
			errCh <- nil
		}(file)
	}

	for i := 0; i < numThreads; i++ {
		if err := <-errCh; err != nil {
			err.Log(log.V(log.Info))
		}
	}

	return nil
}
