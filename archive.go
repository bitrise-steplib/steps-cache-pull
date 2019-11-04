package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
)

// uncompressArchive invokes tar tool against a local archive file.
func uncompressArchive(pth string) error {
	cmd := command.New("tar", "-xpPf", pth)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		errMsg := err.Error()
		if errorutil.IsExitStatusError(err) {
			errMsg = out
		}
		return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), errMsg)
	}

	tarFile, err := os.Open(pth)
	if err != nil {
		return err
	}

	tr := tar.NewReader(tarFile)

	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		// return any other error
		case err != nil:
			return err
		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dir := strings.TrimSuffix(header.Name, "/")

			log.Debugf("D-chtimes: %s - %s - %s", dir, header.AccessTime, header.ModTime)

			if err := os.Chtimes(dir, header.AccessTime, header.ModTime); err != nil {
				log.Debugf("failed to chtimes (%s), error: %s", dir, err)
			}
		case tar.TypeReg, tar.TypeLink, tar.TypeSymlink:
			log.Debugf("F-chtimes: %s - %s - %s", header.Name, header.AccessTime, header.ModTime)

			if err := os.Chtimes(header.Name, header.AccessTime, header.ModTime); err != nil {
				log.Debugf("failed to chtimes (%s), error: %s", header.Name, err)
			}
		}
	}
}

// extractCacheArchive invokes tar tool by piping the archive to the command's input.
func extractCacheArchive(r io.Reader) error {
	cmd := command.New("tar", "-xpPf", "/dev/stdin")
	cmd.SetStdin(r)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		errMsg := err.Error()
		if errorutil.IsExitStatusError(err) {
			errMsg = out
		}
		return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), errMsg)
	}

	if rc, ok := r.(io.ReadCloser); ok {
		return rc.Close()
	}
	return nil
}

// readFirstEntry reads the first entry from a given archive.
func readFirstEntry(r io.Reader) (*tar.Reader, *tar.Header, error) {
	restoreReader := NewRestoreReader(r)

	var archive io.Reader
	var err error

	log.Debugf("attempt to read archive as .gzip")

	archive, err = gzip.NewReader(restoreReader)
	if err != nil {
		// might the archive is not compressed
		log.Debugf("failed to open the archive as .gzip: %s", err)
		log.Debugf("restoring reader and trying as .tar")

		restoreReader.Restore()
		archive = restoreReader
	}

	tr := tar.NewReader(archive)
	hdr, err := tr.Next()
	if err == io.EOF {
		// no entries in the archive
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	return tr, hdr, nil
}
