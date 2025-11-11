package compress

import (
	"compress/gzip"
	"io"
)

// Compress оборачивает reader в gzip сжатие
func Compress(r io.Reader) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		gw := gzip.NewWriter(pw)
		_, err := io.Copy(gw, r)
		
		// Закрываем gzip writer
		if closeErr := gw.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		
		// Закрываем pipe с возможной ошибкой
		pw.CloseWithError(err)
	}()

	return pr, nil
}

// CompressedReader возвращает reader со сжатием
type CompressedReader struct {
	io.ReadCloser
	originalReader io.ReadCloser
}

// NewCompressedReader создает новый сжатый reader
func NewCompressedReader(r io.ReadCloser) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		gw := gzip.NewWriter(pw)
		_, err := io.Copy(gw, r)
		
		if closeErr := gw.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		
		if closeErr := r.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		
		pw.CloseWithError(err)
	}()

	return pr, nil
}
