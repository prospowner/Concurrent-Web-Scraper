package main

import (
	"fmt"
	"net/http"
	"time"
)

// Job represent a single url to scrape
type Job struct {
	ID  int
	URL string
}

// Result represent the result of a scrape
type Result struct {
	ID     int
	URL    string
	Status string
	Code   int
	Error  error
}

// A worker func tht will read from jobs channel and writes to result channel
func worker(id int, jobs <-chan Job, results chan<- Result) {
	for job := range jobs {

		// setup client w a reasonable timeout
		client := http.Client{
			Timeout: 3 * time.Second,
		}

		//perform the http get request
		resp, err := client.Get(job.URL)

		if err != nil {
			results <- Result{
				ID:     job.ID,
				URL:    job.URL,
				Status: "Failed",
				Code:   0,
				Error:  err,
			}
			continue
		}
		resp.Body.Close()

		//Send result back if it succeeds
		results <- Result{
			ID:     job.ID,
			URL:    job.URL,
			Status: "Success",
			Code:   resp.StatusCode,
			Error:  nil,
		}
		//sleep so status changes stay visible for a bit
		time.Sleep(500 * time.Millisecond)
	}
}

// Wiring up the concurreny now
func main() {
	urls := []string{
		"https://golang.org",
		"https://github.com",
		"https://news.ycombinator.com",
		"https://pkg.go.dev",
	}

	numJobs := len(urls)
	jobs := make(chan Job, numJobs)
	results := make(chan Result, numJobs)

	//Spawning 3 concurrent workers
	numWorkers := 3
	for w := 1; w <= numWorkers; w++ {
		go worker(w, jobs, results)
	}

	//Enqueue the jobs
	for i, u := range urls {
		jobs <- Job{ID: i, URL: u}
	}
	close(jobs) //tell workers tht no more jobs are coming and they can rest

	//Collect all results
	for a := 1; a <= numJobs; a++ {
		res := <-results
		if res.Error != nil {
			fmt.Printf("[%d] %s -> Error: %v\n", res.ID, res.URL, res.Error)
		} else {
			fmt.Printf("[%d] %s -> Status: %s (HTTP %d)\n", res.ID, res.URL, res.Status, res.Code)
		}
	}
}
