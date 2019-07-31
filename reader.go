package main

import (
	"bytes"
	"io"

	"github.com/bitrise-io/go-utils/log"
)

// RestoreReader can restore previous read sequence once.
type RestoreReader struct {
	buff bytes.Buffer
	r    io.Reader

	orig io.Reader
	tee  io.Reader

	restore bool
}

// NewRestoreReader creates a new RestoreReader.
func NewRestoreReader(r io.Reader) *RestoreReader {
	a := RestoreReader{}
	a.orig = r
	a.tee = io.TeeReader(r, &a.buff)
	a.r = a.tee
	return &a
}

// Restore instructs the reader to restore previous read sequences.
func (a *RestoreReader) Restore() {
	a.restore = true
}

// Read implements the io.Reader interface.
func (a *RestoreReader) Read(p []byte) (int, error) {
	if a.restore && a.buff.Len() > 0 {
		return a.restoreRead(p)
	}
	return a.r.Read(p)
}

func (a *RestoreReader) restoreRead(p []byte) (int, error) {
	log.Debugf("reading from buffer with size %d", a.buff.Len())

	n, err := a.buff.Read(p)
	if err != nil {
		return n, err
	}
	log.Debugf("%d bytes read from buffer", n)

	if n >= a.buff.Len() {
		log.Debugf("buffer drained")

		a.restore = false
		a.r = a.orig
	}

	if len(p) <= n {
		return n, nil
	}

	log.Debugf("%d remaining bytes to read", len(p)-n)

	b := make([]byte, len(p)-n)

	m, err := a.r.Read(b)
	if err != nil {
		return n + m, err
	}

	log.Debugf("%d bytes read from reader", m)
	log.Debugf("%d bytes read all together", n+m)

	_ = copy(p[n:], b)

	return n + m, nil
}
