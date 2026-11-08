package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        64,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     60 * time.Second,
		ForceAttemptHTTP2:   true,
	},
}

const userAgent = "Mirai/0.1 (+local)"

func fetchBytes(method, target string, body []byte) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			wait := time.Duration(400<<(attempt-1)) * time.Millisecond
			wait += time.Duration(rand.Intn(200)) * time.Millisecond
			time.Sleep(wait)
		}
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, target, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("%s: status %d", target, resp.StatusCode)
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("%s: status %d", target, resp.StatusCode)
		}
		return data, nil
	}
	return nil, lastErr
}

func fetchJSON(method, target string, body []byte, out any) error {
	data, err := fetchBytes(method, target, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func runJobs(count, workers int, job func(i int)) {
	next := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				job(i)
			}
		}()
	}
	for i := 0; i < count; i++ {
		next <- i
	}
	close(next)
	wg.Wait()
}

func parallelRange(n, workers int, fn func(lo, hi int)) {
	if n == 0 {
		return
	}
	size := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for lo := 0; lo < n; lo += size {
		hi := min(lo+size, n)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(lo, hi)
		}()
	}
	wg.Wait()
}

type flexInt int

func (f *flexInt) UnmarshalJSON(data []byte) error {
	text := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if text == "" || text == "null" {
		*f = 0
		return nil
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		*f = 0
		return nil
	}
	*f = flexInt(number)
	return nil
}

type flexFloat float64

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	text := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if text == "" || text == "null" {
		*f = 0
		return nil
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		*f = 0
		return nil
	}
	*f = flexFloat(number)
	return nil
}

type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(text, `"`) {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*f = flexString(value)
		return nil
	}
	*f = flexString(text)
	return nil
}
