package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	defaultTotalIDsForMetrics = 100000
	testMachineID      int64  = 1023
	testMaxCounter     int64  = 4095
)

var (
	testCounter int64 = 0
	testLock    sync.Mutex
)

// Run with:
// go run . test
// go run . test <totalIDs> <workers>
func init() {
	if len(os.Args) < 2 || os.Args[1] != "test" {
		return
	}

	totalIDs := defaultTotalIDsForMetrics
	workers := runtime.NumCPU()

	if len(os.Args) >= 3 {
		if parsed, err := strconv.Atoi(os.Args[2]); err == nil && parsed > 0 {
			totalIDs = parsed
		} else {
			fmt.Printf("Invalid totalIDs '%s', using default: %d\n", os.Args[2], totalIDs)
		}
	}

	if len(os.Args) >= 4 {
		if parsed, err := strconv.Atoi(os.Args[3]); err == nil && parsed > 0 {
			workers = parsed
		} else {
			fmt.Printf("Invalid workers '%s', using default: %d\n", os.Args[3], workers)
		}
	}

	printSnowflakeMetrics(totalIDs, workers)
	os.Exit(0)
}

func generateSnowflakeIdForMetrics() int64 {
	testLock.Lock()
	defer testLock.Unlock()

	currentMillis := time.Now().UnixMilli()

	if testCounter < testMaxCounter {
		testCounter++
	} else {
		testCounter = 0
	}

	id := (currentMillis << 22) | (testMachineID << 12) | testCounter
	return id
}

func printSnowflakeMetrics(totalIDs int, workers int) {
	if totalIDs <= 0 {
		fmt.Println("totalIDs must be > 0")
		return
	}
	if workers <= 0 {
		workers = 1
	}
	if workers > totalIDs {
		workers = totalIDs
	}

	testLock.Lock()
	testCounter = 0
	testLock.Unlock()

	idStream := make(chan int64, 4096)
	var wg sync.WaitGroup

	base := totalIDs / workers
	extra := totalIDs % workers

	start := time.Now()

	for i := 0; i < workers; i++ {
		workCount := base
		if i < extra {
			workCount++
		}

		wg.Add(1)
		go func(count int) {
			defer wg.Done()
			for j := 0; j < count; j++ {
				idStream <- generateSnowflakeIdForMetrics()
			}
		}(workCount)
	}

	go func() {
		wg.Wait()
		close(idStream)
	}()

	seen := make(map[int64]struct{}, totalIDs)
	totalGenerated := 0
	collisions := 0

	for id := range idStream {
		totalGenerated++
		if _, exists := seen[id]; exists {
			collisions++
		} else {
			seen[id] = struct{}{}
		}
	}

	totalDuration := time.Since(start)
	uniqueIDs := len(seen)

	idsPerSecond := 0.0
	if totalDuration.Seconds() > 0 {
		idsPerSecond = float64(totalGenerated) / totalDuration.Seconds()
	}

	avgPerID := time.Duration(0)
	if totalGenerated > 0 {
		avgPerID = totalDuration / time.Duration(totalGenerated)
	}

	collisionRate := 0.0
	if totalGenerated > 0 {
		collisionRate = (float64(collisions) / float64(totalGenerated)) * 100
	}

	fmt.Println("========== Snowflake ID Metrics ==========")
	fmt.Printf("Total IDs requested : %d\n", totalIDs)
	fmt.Printf("Total IDs generated : %d\n", totalGenerated)
	fmt.Printf("Unique IDs          : %d\n", uniqueIDs)
	fmt.Printf("Collisions          : %d\n", collisions)
	fmt.Printf("Collision rate       : %.6f%%\n", collisionRate)
	fmt.Printf("Workers              : %d\n", workers)
	fmt.Printf("Total time           : %s\n", totalDuration)
	fmt.Printf("Generation speed     : %.2f IDs/sec\n", idsPerSecond)
	fmt.Printf("Average time per ID  : %s\n", avgPerID)
	fmt.Println("==========================================")
}
