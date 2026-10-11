package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestWorkerSuccess verifies that a worker successfully scrapes a live HTTP server response
func TestWorkerSuccess(t *testing.T) {
	// 1. Spin up a local mock HTTP server so tests don't depend on the internet
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	jobs := make(chan Job, 1)
	results := make(chan Result, 1)

	// 2. Enqueue our test job pointing to the local mock server
	jobs <- Job{ID: 1, URL: ts.URL}
	close(jobs)

	// 3. Run the worker directly
	go worker(1, jobs, results)

	// 4. Collect and verify the result
	select {
	case res := <-results:
		if res.ID != 1 {
			t.Errorf("Expected Job ID 1, got %d", res.ID)
		}
		if res.Status != "Success" {
			t.Errorf("Expected status 'Success', got '%s'", res.Status)
		}
		if res.Code != http.StatusOK {
			t.Errorf("Expected HTTP status 200, got %d", res.Code)
		}
		if res.Error != nil {
			t.Errorf("Expected no error, got %v", res.Error)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out waiting for worker result")
	}
}

// TestWorkerFailure verifies that a worker handles bad URLs or connection refused gracefully
func TestWorkerFailure(t *testing.T) {
	jobs := make(chan Job, 1)
	results := make(chan Result, 1)

	// Pass an invalid/unreachable URL port
	jobs <- Job{ID: 2, URL: "http://127.0.0.1:1"}
	close(jobs)

	go worker(1, jobs, results)

	select {
	case res := <-results:
		if res.ID != 2 {
			t.Errorf("Expected Job ID 2, got %d", res.ID)
		}
		if res.Status != "Failed" {
			t.Errorf("Expected status 'Failed', got '%s'", res.Status)
		}
		if res.Error == nil {
			t.Error("Expected an error for unreachable port, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Test timed out waiting for failure response")
	}
}

// TestConcurrencyPipeline verifies that multiple workers process multiple jobs concurrently
func TestConcurrencyPipeline(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	numJobs := 5
	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)

	// Enqueue multiple jobs
	for i := 0; i < numJobs; i++ {
		jobs <- Job{ID: i, URL: ts.URL}
	}
	close(jobs)

	// Spawn 2 concurrent workers
	numWorkers := 2
	for w := 1; w <= numWorkers; w++ {
		go worker(w, jobs, results)
	}

	// Collect all results and make sure we got exactly numJobs back
	receivedCount := 0
	for i := 0; i < numJobs; i++ {
		select {
		case res := <-results:
			if res.Status == "Success" {
				receivedCount++
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("Timeout waiting for result %d", i)
		}
	}

	if receivedCount != numJobs {
		t.Errorf("Expected %d successful results, got %d", numJobs, receivedCount)
	}
}
