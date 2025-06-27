package internal

import (
	"bytes"
	"io"
	"net/http"
)

func ReadAndRestoreBody(r *http.Request) []byte {
	if r.Body == nil {
		return nil
	}
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes
}
