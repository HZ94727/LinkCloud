package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type result struct {
	HTTPStatus int
	Code       int
	Err        error
}

func main() {
	var (
		method      = flag.String("method", "POST", "HTTP method: GET or POST")
		url         = flag.String("url", "http://127.0.0.1:8080/api/v1/auth/login", "request URL")
		body        = flag.String("body", `{"user_name":"demo","password":"123456"}`, "request body for POST")
		seconds     = flag.Int("seconds", 5, "test duration in seconds")
		rps         = flag.Int("rps", 20, "requests per second")
		bearer      = flag.String("bearer", "", "optional bearer token")
		contentType = flag.String("content-type", "application/json", "request Content-Type")
	)
	flag.Parse()

	if *rps <= 0 {
		fmt.Fprintln(os.Stderr, "rps must be > 0")
		os.Exit(1)
	}
	if *seconds <= 0 {
		fmt.Fprintln(os.Stderr, "seconds must be > 0")
		os.Exit(1)
	}

	total := (*seconds) * (*rps)
	interval := time.Second / time.Duration(*rps)

	client := &http.Client{Timeout: 5 * time.Second}
	resCh := make(chan result, total)

	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resCh <- doRequest(client, strings.ToUpper(*method), *url, *body, *bearer, *contentType)
		}()
		time.Sleep(interval)
	}

	wg.Wait()
	close(resCh)

	var (
		okCount        int64
		http429Count   int64
		code1902Count  int64
		otherHTTPCount int64
		errCount       int64
	)

	for r := range resCh {
		if r.Err != nil {
			errCount++
			continue
		}
		switch r.HTTPStatus {
		case http.StatusOK:
			okCount++
		case http.StatusTooManyRequests:
			http429Count++
		default:
			otherHTTPCount++
		}
		if r.Code == 1902 {
			code1902Count++
		}
	}

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Printf("target=%s method=%s duration=%ds rps=%d total=%d elapsed=%s\n", *url, strings.ToUpper(*method), *seconds, *rps, total, elapsed)
	fmt.Printf("http_200=%d http_429=%d code_1902=%d other_http=%d errors=%d\n", okCount, http429Count, code1902Count, otherHTTPCount, errCount)
}

func doRequest(client *http.Client, method, url, body, bearer, contentType string) result {
	var reader io.Reader
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		reader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return result{Err: err}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return result{Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return result{HTTPStatus: resp.StatusCode, Err: err}
	}

	var parsed struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(raw, &parsed)

	return result{
		HTTPStatus: resp.StatusCode,
		Code:       parsed.Code,
	}
}
