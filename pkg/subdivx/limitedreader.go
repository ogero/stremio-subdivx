// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subdivx

import (
	"errors"
	"io"
)

var ErrReadBeyondLimit = errors.New("read beyond limit")

// LimitReader returns a Reader that reads from r
// but stops with Err after n bytes.
// The underlying implementation is a *LimitedReader.
func LimitReader(r io.Reader, n int64, err error) io.Reader { return &LimitedReader{r, n, err} }

// A LimitedReader reads from R but limits the amount of
// data returned to just N bytes. Each call to Read
// updates N to reflect the new amount remaining.
// Read returns Err when N <= 0 or EOF when the underlying R returns EOF.
type LimitedReader struct {
	R   io.Reader // underlying reader
	N   int64     // max bytes remaining
	Err error     // the error to return when N <= 0
}

func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, l.Err
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}
