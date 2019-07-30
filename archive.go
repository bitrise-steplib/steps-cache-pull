package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// uncompressArchive invokes tar tool against a local archive file.
func uncompressArchive(pth string) error {
	cmd := command.New("tar", "-xPf", pth)
	return cmd.Run()
}

// extractCacheArchive invokes tar tool by piping the archive to the command's input.
func extractCacheArchive(r io.Reader) error {
	cmd := command.New("tar", "-xPf", "/dev/stdin")
	cmd.SetStdin(r)
	if output, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("failed to extract tar archive, output: %s, error: %s", output, err)
	}

	if rc, ok := r.(io.ReadCloser); ok {
		return rc.Close()
	}
	return nil
}

// readFirstEntry reads the first entry from a given archive.
func readFirstEntry(r io.Reader) (*tar.Reader, *tar.Header, error) {
	replayReader := NewRecorderReader(r)
	replayReader.Record()

	var archive io.Reader
	var err error

	log.Debugf("attempt to read archive as .gzip")

	archive, err = gzip.NewReader(replayReader)
	if err != nil {
		// might the archive is not compressed
		log.Debugf("failed to open the archive as .gzip: %s", err)

		replayReader.Replay()
		archive = replayReader
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
