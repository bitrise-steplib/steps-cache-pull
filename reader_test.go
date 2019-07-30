package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/bitrise-io/go-utils/log"
)

func TestRestoreReader_Read(t *testing.T) {
	content := []byte("test")

	t.Log("simple read")
	{
		r := bytes.NewReader(content)
		rr := NewRestoreReader(r)
		p := make([]byte, len(content))

		// first read - should read the whole content of the reader
		n, err := rr.Read(p)
		if err != nil {
			t.Errorf("RestoreReader.Read() error = %v, wantErr %v", err, nil)
			return
		}
		if n != len(content) {
			t.Errorf("RestoreReader.Read() n = %d, want %d", n, len(content))
			return
		}
		if string(p) != string(content) {
			t.Errorf("RestoreReader.Read() p = %s, want %s", p, content)
			return
		}

		// second read - should read from an empty reader
		p = make([]byte, len(content))
		n, err = rr.Read(p)
		if err != io.EOF {
			t.Errorf("Second RestoreReader.Read() error = %v, wantErr %v", err, io.EOF)
			return
		}
		if n != 0 {
			t.Errorf("Second RestoreReader.Read() n = %d, want %d", n, 0)
			return
		}
	}

	t.Log("restore read")
	{
		r := bytes.NewReader(content)
		rr := NewRestoreReader(r)
		p := make([]byte, len(content))

		// first read - should read the whole content of the reader
		n, err := rr.Read(p)
		if err != nil {
			t.Errorf("RestoreReader.Read() error = %v, wantErr %v", err, nil)
			return
		}
		if n != len(content) {
			t.Errorf("RestoreReader.Read() n = %d, want %d", n, len(content))
			return
		}
		if string(p) != string(content) {
			t.Errorf("RestoreReader.Read() p = %s, want %s", p, content)
			return
		}

		rr.Restore()

		// second read - should read the same content
		p = make([]byte, len(content))
		n, err = rr.Read(p)
		if err != nil {
			t.Errorf("RestoreReader.Read() error = %v, wantErr %v", err, nil)
			return
		}
		if n != len(content) {
			t.Errorf("RestoreReader.Read() n = %d, want %d", n, len(content))
			return
		}
		if string(p) != string(content) {
			t.Errorf("RestoreReader.Read() p = %s, want %s", p, content)
			return
		}
	}

	t.Log("restore read - continue reading")
	{
		r := bytes.NewReader(content)
		rr := NewRestoreReader(r)
		p := make([]byte, 1)

		// first read - should read 1 byte from the reader
		n, err := rr.Read(p)
		if err != nil {
			t.Errorf("RestoreReader.Read() error = %v, wantErr %v", err, nil)
			return
		}
		if n != 1 {
			t.Errorf("RestoreReader.Read() n = %d, want %d", n, 1)
			return
		}
		if string(p) != "t" {
			t.Errorf("RestoreReader.Read() p = %s, want %s", p, "t")
			return
		}

		rr.Restore()
		log.SetEnableDebugLog(true)

		// second read - should read the same content
		p = make([]byte, len(content))
		n, err = rr.Read(p)
		if err != nil {
			t.Errorf("RestoreReader.Read() error = %v, wantErr %v", err, nil)
			return
		}
		if n != len(content) {
			t.Errorf("RestoreReader.Read() n = %d, want %d", n, len(content))
			return
		}
		if string(p) != string(content) {
			t.Errorf("RestoreReader.Read() p = %s, want %s", p, content)
			return
		}
	}
}
