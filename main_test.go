package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// startSlowServer serves a handler that signals once a request arrives and
// then blocks until release is closed.
func startSlowServer(t *testing.T, ctx context.Context, drain time.Duration, release <-chan struct{}) (string, <-chan struct{}, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	arrived := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(arrived)
		<-release
		_, _ = io.WriteString(w, "done")
	})}
	done := make(chan error, 1)
	go func() { done <- serveUntilDone(ctx, srv, listener, drain) }()
	return "http://" + listener.Addr().String(), arrived, done
}

func TestServeUntilDoneDrainsInFlightRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})
	url, arrived, done := startSlowServer(t, ctx, 5*time.Second, release)

	type result struct {
		body string
		err  error
	}
	response := make(chan result, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			response <- result{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		response <- result{string(body), err}
	}()
	<-arrived

	// The signal arrives while the request is still running.
	cancel()
	select {
	case err := <-done:
		t.Fatalf("serveUntilDone returned %v before the in-flight request finished", err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := http.Get(url); err == nil {
		t.Error("expected new connections to be refused once shutdown began")
	}

	close(release)
	if got := <-response; got.err != nil || got.body != "done" {
		t.Errorf("in-flight request = %q, %v; want it to complete", got.body, got.err)
	}
	if err := <-done; err != nil {
		t.Errorf("serveUntilDone = %v, want nil after a clean shutdown", err)
	}
}

func TestServeUntilDoneGivesUpAfterDrainTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})
	defer close(release)
	url, arrived, done := startSlowServer(t, ctx, 50*time.Millisecond, release)

	go func() {
		if resp, err := http.Get(url); err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-arrived
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected an error when requests outlast the drain timeout")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveUntilDone did not return after the drain timeout")
	}
}
