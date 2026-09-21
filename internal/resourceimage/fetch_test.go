package resourceimage

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func imagePNG(t *testing.T) []byte {
	t.Helper()
	m := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	m.Set(0, 0, color.NRGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestFetcherNormalizesAndDoesNotForwardCredentials(t *testing.T) {
	f := NewFetcher()
	body := append(imagePNG(t), []byte("<script>evil()</script>")...)
	f.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("Referer") != "" {
			t.Fatal("forwarded viewer credentials")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/html"}}, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})
	got, err := f.Fetch(context.Background(), "https://images.example/picture")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got, []byte("script")) {
		t.Fatal("preserved trailing content")
	}
	if _, err := png.Decode(bytes.NewReader(got)); err != nil {
		t.Fatal(err)
	}
}

func TestFetcherRejectsUnsafeResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		body     []byte
		encoding string
		length   int64
	}{
		{"redirect", 302, imagePNG(t), "", -1},
		{"html", 200, []byte("<html><img src='https://evil.example'></html>"), "", -1},
		{"svg", 200, []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), "", -1},
		{"compressed", 200, imagePNG(t), "gzip", -1},
		{"large-header", 200, imagePNG(t), "", MaxDownloadBytes + 1},
		{"large-body", 200, bytes.Repeat([]byte{0}, MaxDownloadBytes+1), "", -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := NewFetcher()
			calls := 0
			f.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: test.status, ContentLength: test.length, Header: http.Header{"Location": {"https://internal.example/secret"}, "Content-Encoding": {test.encoding}}, Body: io.NopCloser(bytes.NewReader(test.body)), Request: r}, nil
			})
			if _, err := f.Fetch(context.Background(), "https://images.example/picture"); err == nil {
				t.Fatal("accepted unsafe response")
			}
			if calls != 1 {
				t.Fatal("followed redirect")
			}
		})
	}
}

func TestFetcherRejectsUnsafeURLsBeforeTransport(t *testing.T) {
	f := NewFetcher()
	f.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected request"); return nil, ErrImage })
	for _, source := range []string{"http://images.example/picture", "https://127.0.0.1/x", "https://[::1]/x", "https://user:password@images.example/x", "https://images.example:444/x", "//images.example/x", "data:image/png;base64,abc"} {
		if _, err := f.Fetch(context.Background(), source); err == nil {
			t.Errorf("accepted %s", source)
		}
	}
}

func TestProductionTransportBlocksPrivateAddresses(t *testing.T) {
	f := NewFetcher()
	transport := f.client.Transport.(*http.Transport)
	if transport.Proxy != nil || !transport.DisableCompression || f.client.Jar != nil {
		t.Fatal("unsafe transport configuration")
	}
	for _, address := range []string{"127.0.0.1:443", "[::1]:443", "10.0.0.1:443", "169.254.169.254:443", "[::ffff:127.0.0.1]:443"} {
		conn, err := transport.DialContext(context.Background(), "tcp", address)
		if conn != nil {
			conn.Close()
			t.Fatalf("connected to %s", address)
		}
		if err == nil || !strings.Contains(err.Error(), "blocked") {
			t.Fatalf("address was not rejected by guard: %s: %v", address, err)
		}
	}
}

func TestNormalizeRejectsExcessiveDimensions(t *testing.T) {
	body := imagePNG(t)
	binary.BigEndian.PutUint32(body[16:20], 100000)
	binary.BigEndian.PutUint32(body[20:24], 100000)
	binary.BigEndian.PutUint32(body[29:33], crc32.ChecksumIEEE(body[12:29]))
	if _, err := normalize(body); err == nil {
		t.Fatal("accepted oversized dimensions")
	}
}

func TestFetcherBoundsConcurrency(t *testing.T) {
	f := NewFetcher()
	for i := 0; i < cap(f.pending); i++ {
		f.pending <- struct{}{}
	}
	if _, err := f.Fetch(context.Background(), "https://images.example/picture"); err == nil {
		t.Fatal("exceeded concurrency limit")
	}
}

func TestFetcherRechecksPermissionAfterWaitingForCapacity(t *testing.T) {
	f := NewFetcher()
	for range cap(f.slots) {
		f.slots <- struct{}{}
	}
	f.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("request started after permission was revoked")
		return nil, ErrImage
	})
	var revoked atomic.Bool
	result := make(chan error, 1)
	go func() {
		_, err := f.FetchAuthorized(context.Background(), "https://images.example/a", func() error {
			if revoked.Load() {
				return ErrImage
			}
			return nil
		})
		result <- err
	}()
	deadline := time.Now().Add(time.Second)
	for len(f.pending) == 0 && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if len(f.pending) == 0 {
		t.Fatal("request did not enter the queue")
	}
	revoked.Store(true)
	<-f.slots
	if err := <-result; err == nil {
		t.Fatal("queued request ignored revoked permission")
	}
}
