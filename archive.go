package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
)

// uncompressArchive invokes tar tool against a local archive file.
func uncompressArchive(pth string, relative, compressed bool) error {
	cmd := command.New("tar", processArgs(relative, compressed), pth)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		errMsg := err.Error()
		if errorutil.IsExitStatusError(err) {
			errMsg = out
		}
		return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), errMsg)
	}
	return nil
}

// extractCacheArchive invokes tar tool by piping the archive to the command's input.
func extractCacheArchive(r io.Reader, relative, compressed bool) error {
	cmd := command.New("tar", processArgs(relative, compressed), "-")
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

func processArgs(relative, compressed bool) string {
	/*
		GNU  tar options

		-f "-" : reads the archive from standard input
		https://www.gnu.org/software/tar/manual/html_node/Device.html#SEC155

		-x : extract files from an archive
		https://www.gnu.org/software/tar/manual/html_node/extract.html#SEC25

		-P : Don't strip an initial `/' from member names
		https://www.gnu.org/software/tar/manual/html_node/absolute.html#SEC120

		-z : tells tar to read or write archives through gzip
		https://www.gnu.org/software/tar/manual/html_node/gzip.html#SEC135
	*/

	args := "-x"
	if !relative {
		args += "P"
	}
	if compressed {
		args += "z"
	}
	args += "f"
	return args
}

// readFirstEntry reads the first entry from a given archive.
func readFirstEntry(r io.Reader) (*tar.Reader, *tar.Header, bool, error) {
	restoreReader := NewRestoreReader(r)

	var archive io.Reader
	var err error
	compressed := true

	log.Debugf("attempt to read archive as .gzip")

	archive, err = gzip.NewReader(restoreReader)
	if err != nil {
		// might the archive is not compressed
		log.Debugf("failed to open the archive as .gzip: %s", err)
		log.Debugf("restoring reader and trying as .tar")

		restoreReader.Restore()
		archive = restoreReader
		compressed = false
	}

	tr := tar.NewReader(archive)
	hdr, err := tr.Next()
	if err == io.EOF {
		// no entries in the archive
		return nil, nil, compressed, nil
	}
	if err != nil {
		return nil, nil, compressed, err
	}

	return tr, hdr, compressed, nil
}
