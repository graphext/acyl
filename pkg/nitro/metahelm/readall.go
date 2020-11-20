package metahelm

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/errors"
	billy "gopkg.in/src-d/go-billy.v4"
)

var fileSizeMaxBytes = 500 * 1000000 // we won't try to read files larger than this into memory

func readFileSafely(fs billy.Filesystem, fp string) ([]byte, error) {
	fi, err := fs.Stat(fp)
	if err != nil {
		return nil, errors.Wrap(err, "error getting file info")
	}
	if fi.Size() > int64(fileSizeMaxBytes) {
		return nil, fmt.Errorf("file size exceeds limit (%v bytes): %v", fileSizeMaxBytes, fi.Size())
	}
	return readFile(fs, fp)
}

// Copypasta of ioutil.ReadFile except injecting a billy.Filesystem

// readAll reads from r until an error or EOF and returns the data it read
// from the internal buffer allocated with a specified capacity.
func readAll(r io.Reader, capacity int64) (b []byte, err error) {
	var buf bytes.Buffer
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	if int64(int(capacity)) == capacity {
		buf.Grow(int(capacity))
	}
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func readFile(fs billy.Filesystem, filename string) ([]byte, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// It's a good but not certain bet that FileInfo will tell us exactly how much to
	// read, so let's try it but be prepared for the answer to be wrong.
	var n int64 = bytes.MinRead

	if fi, err := fs.Stat(filename); err == nil {
		// As initial capacity for readAll, use Size + a little extra in case Size
		// is zero, and to avoid another allocation after Read has filled the
		// buffer. The readAll call will read into its allocated internal buffer
		// cheaply. If the size was wrong, we'll either waste some space off the end
		// or reallocate as needed, but in the overwhelmingly common case we'll get
		// it just right.
		if size := fi.Size() + bytes.MinRead; size > n {
			n = size
		}
	}
	return readAll(f, n)
}
