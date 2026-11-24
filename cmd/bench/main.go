package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	TotalDuration   time.Duration
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	StatusCodes     map[int]int64
	Latencies       []time.Duration
	BytesRead       int64
}

func main() {
	targetURL := flag.String("url", "http://127.0.0.1:8080/", "Target URL to benchmark")
	concurrency := flag.Int("c", 100, "Number of concurrent workers")
	duration := flag.Duration("d", 10*time.Second, "Duration of the benchmark")
	warmup := flag.Duration("warmup", 1*time.Second, "Warm-up duration before recording metrics")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Printf("   ProxyGO BENCHMARK ENGINE (High-Concurrency Load Generator)\n")
	fmt.Println("================================================================================")
	fmt.Printf("Target URL:    %s\n", *targetURL)
	fmt.Printf("Concurrency:   %d workers\n", *concurrency)
	fmt.Printf("Duration:      %v (+ %v warmup)\n", *duration, *warmup)
	fmt.Println("--------------------------------------------------------------------------------")

	// Custom optimized transport
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        *concurrency * 4,
		MaxIdleConnsPerHost: *concurrency * 4,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression: false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Warmup phase
	if *warmup > 0 {
		fmt.Printf("Warming up connections for %v...\n", *warmup)
		runPhase(client, *targetURL, *concurrency, *warmup, false)
		fmt.Println("Warmup completed. Running official benchmark...")
	}

	// Official benchmark phase
	result := runPhase(client, *targetURL, *concurrency, *duration, true)
	printReport(result)
}

func runPhase(client *http.Client, url string, concurrency int, duration time.Duration, collectLatencies bool) *Result {
	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	var totalReqs atomic.Int64
	var successReqs atomic.Int64
	var failedReqs atomic.Int64
	var totalBytes atomic.Int64

	var statusLock sync.Mutex
	statusCodes := make(map[int]int64)

	// Collect latencies slice per worker to avoid lock contention
	workerLatencies := make([][]time.Duration, concurrency)

	startTime := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		workerID := i
		if collectLatencies {
			workerLatencies[workerID] = make([]time.Duration, 0, 50000)
		}

		go func(id int) {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "ReverseProxy-Benchmark/1.0")

			localStatuses := make(map[int]int64)

			for {
				select {
				case <-stopCh:
					statusLock.Lock()
					for code, count := range localStatuses {
						statusCodes[code] += count
					}
					statusLock.Unlock()
					return
				default:
					t0 := time.Now()
					resp, err := client.Do(req)
					elapsed := time.Since(t0)

					totalReqs.Add(1)

					if err != nil {
						failedReqs.Add(1)
						localStatuses[0]++
						continue
					}

					localStatuses[resp.StatusCode]++
					if resp.StatusCode >= 200 && resp.StatusCode < 400 {
						successReqs.Add(1)
					} else {
						failedReqs.Add(1)
					}

					n, _ := io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
					totalBytes.Add(n)

					if collectLatencies {
						workerLatencies[id] = append(workerLatencies[id], elapsed)
					}
				}
			}
		}(workerID)
	}

	time.Sleep(duration)
	close(stopCh)
	wg.Wait()
	actualDuration := time.Since(startTime)

	var allLatencies []time.Duration
	if collectLatencies {
		totalCount := 0
		for _, l := range workerLatencies {
			totalCount += len(l)
		}
		allLatencies = make([]time.Duration, 0, totalCount)
		for _, l := range workerLatencies {
			allLatencies = append(allLatencies, l...)
		}
	}

	return &Result{
		TotalDuration:   actualDuration,
		TotalRequests:   totalReqs.Load(),
		SuccessRequests: successReqs.Load(),
		FailedRequests:  failedReqs.Load(),
		StatusCodes:     statusCodes,
		Latencies:       allLatencies,
		BytesRead:       totalBytes.Load(),
	}
}

func printReport(r *Result) {
	rps := float64(r.TotalRequests) / r.TotalDuration.Seconds()
	mbTransferred := float64(r.BytesRead) / (1024 * 1024)

	fmt.Println("================================================================================")
	fmt.Printf("   BENCHMARK RESULTS SUMMARY\n")
	fmt.Println("================================================================================")
	fmt.Printf("Total Elapsed Time:   %.2fs\n", r.TotalDuration.Seconds())
	fmt.Printf("Total Requests:       %d\n", r.TotalRequests)
	fmt.Printf("Successful Requests:  %d (%.2f%%)\n", r.SuccessRequests, float64(r.SuccessRequests)/float64(r.TotalRequests)*100)
	fmt.Printf("Failed Requests:      %d\n", r.FailedRequests)
	fmt.Printf("Data Transferred:     %.2f MB (%.2f MB/s)\n", mbTransferred, mbTransferred/r.TotalDuration.Seconds())
	fmt.Printf("\n>>> THROUGHPUT:       \033[1;32m%.2f req/s\033[0m <<<\n", rps)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("HTTP Status Codes:")
	for code, count := range r.StatusCodes {
		if code == 0 {
			fmt.Printf("  Connection Errors:  %d\n", count)
		} else {
			fmt.Printf("  HTTP %d:            %d\n", code, count)
		}
	}

	if len(r.Latencies) > 0 {
		sort.Slice(r.Latencies, func(i, j int) bool {
			return r.Latencies[i] < r.Latencies[j]
		})

		var sum time.Duration
		for _, lat := range r.Latencies {
			sum += lat
		}
		avg := sum / time.Duration(len(r.Latencies))
		min := r.Latencies[0]
		max := r.Latencies[len(r.Latencies)-1]
		p50 := quantile(r.Latencies, 0.50)
		p90 := quantile(r.Latencies, 0.90)
		p95 := quantile(r.Latencies, 0.95)
		p99 := quantile(r.Latencies, 0.99)
		p999 := quantile(r.Latencies, 0.999)

		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("Latency Distribution (Percentiles):")
		fmt.Printf("  Min:                %v\n", min)
		fmt.Printf("  Avg (Mean):         %v\n", avg)
		fmt.Printf("  p50 (Median):       \033[1;34m%v\033[0m\n", p50)
		fmt.Printf("  p90:                %v\n", p90)
		fmt.Printf("  p95:                \033[1;34m%v\033[0m\n", p95)
		fmt.Printf("  p99:                \033[1;32m%v\033[0m  (Target: < 3ms)\n", p99)
		fmt.Printf("  p99.9:              %v\n", p999)
		fmt.Printf("  Max:                %v\n", max)
	}
	fmt.Println("================================================================================")
}

func quantile(latencies []time.Duration, q float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	idx := int(float64(len(latencies)-1) * q)
	return latencies[idx]
}
