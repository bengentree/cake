package cloudinit

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kdomanski/iso9660"
)

// GenerateSeedISO creates an ISO with user and meta data
func GenerateSeedISO(userdata string, metadata string) ([]byte, error) {
	writer, err := iso9660.NewWriter()
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %v", err)
	}
	defer writer.Cleanup()
	r := strings.NewReader(metadata)
	err = writer.AddFile(r, "meta-data")
	if err != nil {
		return nil, fmt.Errorf("failed to add file: %v", err)
	}
	r = strings.NewReader(userdata)
	err = writer.AddFile(r, "user-data")
	if err != nil {
		return nil, fmt.Errorf("failed to add file: %v", err)
	}
	wabuf := newWriteAtBuffer(nil)
	err = writer.WriteTo(wabuf, "cidata")
	if err != nil {
		return nil, fmt.Errorf("failed to write ISO image: %v", err)
	}

	return wabuf.Bytes(), nil
}

// a stripped down version of the WriteAtBuffer from
// https://github.com/aws/aws-sdk-go/blob/master/aws/types.go and
// https://github.com/LINBIT/virter/blob/4d8e32cc43da51cfed5e357456dd540a12ced0d2/internal/virter/writeatbuffer.go

type writeAtBuffer struct {
	buf []byte
	m   sync.Mutex
}

func newWriteAtBuffer(buf []byte) *writeAtBuffer {
	return &writeAtBuffer{buf: buf}
}

func (b *writeAtBuffer) WriteAt(p []byte, pos int64) (n int, err error) {
	pLen := len(p)
	expLen := pos + int64(pLen)
	b.m.Lock()
	defer b.m.Unlock()
	if int64(len(b.buf)) < expLen {
		if int64(cap(b.buf)) < expLen {
			newBuf := make([]byte, expLen)
			copy(newBuf, b.buf)
			b.buf = newBuf
		}
		b.buf = b.buf[:expLen]
	}
	copy(b.buf[pos:], p)
	return pLen, nil
}

func (b *writeAtBuffer) Bytes() []byte {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buf
}
