package main

import (
	"bytes"
	"io"

	"github.com/bitrise-io/go-utils/log"
)

// RecorderReader can record once reads and can replay the previous reads.
type RecorderReader struct {
	buff bytes.Buffer
	orig io.Reader
	tee  io.Reader

	record bool
	replay bool
}

// NewRecorderReader creates a new RecorderReader
func NewRecorderReader(r io.Reader) *RecorderReader {
	a := RecorderReader{}
	a.orig = r
	a.tee = io.TeeReader(r, &a.buff)
	return &a
}

// Record instructs the reader to record upcoming reads.
func (a *RecorderReader) Record() {
	a.record = true
	a.replay = false
}

// Replay instructs the reader to replay previous reads.
func (a *RecorderReader) Replay() {
	log.Debugf("using buffer with %d bytes", a.buff.Len())

	a.replay = true
	a.record = false
}

// Read implements the io.Reader interface.
func (a *RecorderReader) Read(p []byte) (n int, err error) {
	log.Debugf("----------")
	log.Debugf("attempting to read %d bytes", len(p))

	if a.record {
		log.Debugf("reading from the original reader")

		return a.tee.Read(p)
	}

	if a.replay && a.buff.Len() > 0 {
		log.Debugf("reading from buffer with size %d", a.buff.Len())

		n, err := a.buff.Read(p)
		if err != nil {
			return n, err
		}
		log.Debugf("%d bytes read from buffer", n)

		if len(p) > n {
			log.Debugf("%d remaining bytes to read", len(p)-n)

			b := make([]byte, len(p)-n)

			m, err := a.tee.Read(b)
			if err != nil {
				return n + m, err
			}

			log.Debugf("%d bytes read from original reader", m)
			log.Debugf("%d bytes read all together", n+m)

			if n+m >= len(p) {
				log.Debugf("buffer drained")

				a.replay = false
			}

			return n + m, nil
		}

		return n, nil
	}

	log.Debugf("reading from the original reader")

	return a.orig.Read(p)
}
