package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func HTTPRangeRequest(url string, start, end int64) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	req.Header.Set("Range", rangeHeader)
	req.Header.Set("User-Agent", "video-clipper/1.0")
	return client.Do(req)
}

func CheckRangeSupport(url string) (bool, int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	acceptRanges := resp.Header.Get("Accept-Ranges")
	contentLength := resp.Header.Get("Content-Length")

	var size int64
	if contentLength != "" {
		size, _ = strconv.ParseInt(contentLength, 10, 64)
	}

	return acceptRanges == "bytes", size, nil
}

func WriteChunkedResponse(w http.ResponseWriter, reader io.Reader, contentType string) error {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="clip.mp4"`)
	w.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	buffer := make([]byte, 8192) // 8KB chunks
	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
	}

	return nil
}
