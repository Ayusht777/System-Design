package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorRed    = "\033[31m"
	colorBold   = "\033[1m"
)

func main() {
	// ── CLI flags ──────────────────────────────────────────────
	totalRequests := flag.Int("n", 10000, "Total number of requests to send")
	concurrency := flag.Int("c", 1000, "Number of concurrent workers")
	targetURL := flag.String("url", "http://localhost:8080/", "Load balancer URL")
	flag.Parse()

	fmt.Printf("\n%s%s══════════════════════════════════════════%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s   Load Balancer — Stress Tester%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s══════════════════════════════════════════%s\n\n", colorBold, colorCyan, colorReset)
	fmt.Printf("  Target URL    : %s\n", *targetURL)
	fmt.Printf("  Total Requests: %d\n", *totalRequests)
	fmt.Printf("  Concurrency   : %d\n\n", *concurrency)

	// ── Shared counters ────────────────────────────────────────
	var (
		successCount  int64
		failCount     int64
		completedReqs int64
		// per-second counter (reset every tick)
		perSecondCount int64
	)

	// Collect all latencies for percentile calculation
	latencies := make([]time.Duration, 0, *totalRequests)
	var latMu sync.Mutex

	// ── HTTP client with sensible timeouts ─────────────────────
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// ── Live ticker — prints req/sec every second ──────────────
	ticker := time.NewTicker(1 * time.Second)
	startTime := time.Now()
	stopTicker := make(chan struct{})

	go func() {
		seconds := 0
		for {
			select {
			case <-ticker.C:
				seconds++
				count := atomic.SwapInt64(&perSecondCount, 0)
				done := atomic.LoadInt64(&completedReqs)
				elapsed := time.Since(startTime).Seconds()
				overallRPS := float64(done) / elapsed
				fmt.Printf("  [%4ds]  req/sec: %s%-5d%s  |  completed: %-6d  |  avg req/sec: %s%.1f%s\n",
					seconds, colorGreen, count, colorReset, done, colorYellow, overallRPS, colorReset)
			case <-stopTicker:
				return
			}
		}
	}()

	// ── Worker pool ────────────────────────────────────────────
	jobs := make(chan int, *totalRequests)
	var wg sync.WaitGroup

	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				reqStart := time.Now()

				resp, err := client.Get(*targetURL)
				latency := time.Since(reqStart)

				if err != nil {
					atomic.AddInt64(&failCount, 1)
				} else {
					// drain and close to allow connection reuse
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					atomic.AddInt64(&successCount, 1)
				}

				latMu.Lock()
				latencies = append(latencies, latency)
				latMu.Unlock()

				atomic.AddInt64(&completedReqs, 1)
				atomic.AddInt64(&perSecondCount, 1)
			}
		}()
	}

	// Enqueue all requests
	for i := 0; i < *totalRequests; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for every request to finish
	wg.Wait()
	totalDuration := time.Since(startTime)

	// Stop the live ticker
	ticker.Stop()
	close(stopTicker)

	// ── Compute stats ──────────────────────────────────────────
	slices.Sort(latencies)

	min := latencies[0]
	max := latencies[len(latencies)-1]
	p50 := latencies[len(latencies)*50/100]
	p90 := latencies[len(latencies)*90/100]
	p99 := latencies[len(latencies)*99/100]

	var totalLatency time.Duration
	for _, l := range latencies {
		totalLatency += l
	}
	avg := totalLatency / time.Duration(len(latencies))

	overallRPS := float64(completedReqs) / totalDuration.Seconds()
	overallRPM := overallRPS * 60

	// ── Final report ───────────────────────────────────────────
	fmt.Printf("\n%s%s══════════════════════════════════════════%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s          LOAD TEST RESULTS%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s══════════════════════════════════════════%s\n\n", colorBold, colorCyan, colorReset)

	fmt.Printf("  %sTotal Requests  :%s  %d\n", colorBold, colorReset, completedReqs)
	fmt.Printf("  %sSuccessful      :%s  %s%d%s\n", colorBold, colorReset, colorGreen, successCount, colorReset)
	fmt.Printf("  %sFailed          :%s  %s%d%s\n", colorBold, colorReset, colorRed, failCount, colorReset)
	fmt.Printf("  %sTotal Duration  :%s  %s\n", colorBold, colorReset, totalDuration.Round(time.Millisecond))
	fmt.Println()
	fmt.Printf("  %sThroughput%s\n", colorBold, colorReset)
	fmt.Printf("    Requests/sec  :  %s%.2f%s\n", colorYellow, overallRPS, colorReset)
	fmt.Printf("    Requests/min  :  %s%.0f%s\n", colorYellow, overallRPM, colorReset)
	fmt.Println()
	fmt.Printf("  %sLatency%s\n", colorBold, colorReset)
	fmt.Printf("    Min           :  %s\n", min.Round(time.Microsecond))
	fmt.Printf("    Avg           :  %s\n", avg.Round(time.Microsecond))
	fmt.Printf("    P50 (median)  :  %s\n", p50.Round(time.Microsecond))
	fmt.Printf("    P90           :  %s\n", p90.Round(time.Microsecond))
	fmt.Printf("    P99           :  %s\n", p99.Round(time.Microsecond))
	fmt.Printf("    Max           :  %s\n", max.Round(time.Microsecond))

	fmt.Printf("\n%s%s══════════════════════════════════════════%s\n\n", colorBold, colorCyan, colorReset)

	if failCount > 0 {
		os.Exit(1)
	}
}
