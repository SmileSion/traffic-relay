package internal

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

const maxLogSize = 1024

func DumpRequest(r *http.Request, body []byte) string {
	var b bytes.Buffer

	// 请求行
	fmt.Fprintf(&b, "=== HTTP REQUEST START ===\n")
	fmt.Fprintf(&b, "Method: %s\n", r.Method)
	fmt.Fprintf(&b, "URI: %s\n", r.URL.RequestURI())
	fmt.Fprintf(&b, "Protocol: %s\n", r.Proto)

	// 请求头
	fmt.Fprintf(&b, "\n--- Headers ---\n")
	for k, v := range r.Header {
		fmt.Fprintf(&b, "%s: %s\n", k, strings.Join(v, ", "))
	}

	// 请求体
	fmt.Fprintf(&b, "\n--- Body ---\n")
	if len(body) == 0 {
		fmt.Fprintf(&b, "[空]\n")
	} else if len(body) > maxLogSize {
		fmt.Fprintf(&b, "%s\n[Body已截断，原始长度 %d 字节]\n", string(body[:maxLogSize]), len(body))
	} else {
		fmt.Fprintf(&b, "%s\n", string(body))
	}

	fmt.Fprintf(&b, "=== HTTP REQUEST END ===\n")

	return b.String()
}

func DumpResponse(resp *http.Response) string {
	var b strings.Builder
	b.WriteString("=== HTTP RESPONSE START ===\n")
	fmt.Fprintf(&b, "Status Code: %d\n", resp.StatusCode)

	b.WriteString("\n--- Headers ---\n")
	for k, v := range resp.Header {
		fmt.Fprintf(&b, "%s: %s\n", k, strings.Join(v, ","))
	}
	b.WriteString("=== HTTP RESPONSE END ===")
	return b.String()
}


func GetMethod(r *http.Request, override string) string {
	method := r.Method
	if override != "" && strings.ToUpper(override) != r.Method {
		method = strings.ToUpper(override)
	}
	return method
}

func BuildTargetURL(base string, r *http.Request) string {
	if base == "" {
		return ""
	}
	target := strings.TrimRight(base, "/") + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	return target
}

func CopyHeaders(w http.ResponseWriter, headers http.Header) {
	for k, v := range headers {
		w.Header()[k] = v
	}
}
