package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

// BenchmarkResult represents performance data of file uploads
type BenchmarkResult struct {
	Timestamp     time.Time
	FileName      string
	FileSize      int64
	Duration      time.Duration
	TransferRate  float64 // bytes per second
	MemoryUsage   uint64  // bytes
	CPUUsage      float64 // percentage of CPU(s) used (may exceed 100% on multi-core)
	NumGoroutines int     // number of goroutines
}

// getMemoryUsage returns current memory usage statistics
func getMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// monitorPerformance captures CPU and memory usage during a function execution
func monitorPerformance(logger *log.Logger, interval time.Duration, done <-chan struct{}) (maxMem uint64, avgCPU float64) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var m runtime.MemStats
	maxMem = 0
	cpuSamples := 0
	totalCPU := 0.0

	// Initial CPU time as baseline
	var startNumCPU = runtime.NumCPU()
	var prevCPUTime time.Time = time.Now()

	for {
		select {
		case <-ticker.C:
			// Memory stats
			runtime.ReadMemStats(&m)
			if m.Alloc > maxMem {
				maxMem = m.Alloc
			}

			// CPU approximation based on time elapsed and goroutines
			// This is a very rough approximation as Go doesn't expose precise CPU usage per process
			now := time.Now()
			elapsed := now.Sub(prevCPUTime).Seconds()
			numGoroutines := float64(runtime.NumGoroutine())

			// Estimate CPU usage as goroutines per CPU core
			// This is not accurate but provides a relative measure
			cpuEstimate := (numGoroutines / float64(startNumCPU)) * 100

			totalCPU += cpuEstimate
			cpuSamples++

			prevCPUTime = now

			logger.Debug("performance snapshot",
				"mem_alloc_mb", float64(m.Alloc)/1024/1024,
				"goroutines", runtime.NumGoroutine(),
				"cpu_estimate", cpuEstimate,
				"elapsed", elapsed,
			)

		case <-done:
			if cpuSamples > 0 {
				avgCPU = totalCPU / float64(cpuSamples)
			}
			return maxMem, avgCPU
		}
	}
}

func main() {
	// Configure charmbracelet/log
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
	})

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create a Gin router with default middleware
	router := gin.Default()

	// Increase the max multipart memory
	router.MaxMultipartMemory = 8 << 20 // 8 MiB, just for initial buffer

	// Create a mutex to protect benchmark file writing
	var benchmarkMutex sync.Mutex

	// Log system info on startup
	logger.Info("server starting",
		"address", ":8080",
		"go_version", runtime.Version(),
		"cpu_cores", runtime.NumCPU(),
		"gomaxprocs", runtime.GOMAXPROCS(0),
	)

	// Create upload endpoint
	router.POST("/upload", func(c *gin.Context) {
		startTime := time.Now()
		initialMemory := getMemoryUsage()

		// Start performance monitoring
		perfDone := make(chan struct{})
		var maxMem uint64
		var avgCPU float64

		go func() {
			maxMem, avgCPU = monitorPerformance(logger, 200*time.Millisecond, perfDone)
		}()

		logger.Info("upload request received",
			"client_ip", c.ClientIP(),
			"initial_mem_mb", float64(initialMemory)/1024/1024,
		)

		// Get destination path from query parameter
		destPath := c.Query("dest")
		if destPath == "" {
			logger.Error("destination path not provided")
			c.JSON(http.StatusBadRequest, gin.H{"error": "destination path is required"})
			close(perfDone)
			return
		}

		logger.Info("processing upload", "destination", destPath)

		// Ensure destination directory exists
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			logger.Error("failed to create directory", "error", err, "path", destDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create directory: %v", err)})
			close(perfDone)
			return
		}

		// Get file from request
		fileHeader, err := c.FormFile("file")
		if err != nil {
			logger.Error("failed to get file from request", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to get file: %v", err)})
			close(perfDone)
			return
		}

		fileName := fileHeader.Filename
		fileSize := fileHeader.Size
		logger.Info("file received",
			"name", fileName,
			"size", fileSize,
			"current_mem_mb", float64(getMemoryUsage())/1024/1024,
		)

		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			logger.Error("failed to open uploaded file", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to open file: %v", err)})
			close(perfDone)
			return
		}
		defer file.Close()

		// Create destination file
		dst, err := os.Create(destPath)
		if err != nil {
			logger.Error("failed to create destination file", "error", err, "path", destPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create destination file: %v", err)})
			close(perfDone)
			return
		}
		defer dst.Close()

		// Measure file copy time
		copyStartTime := time.Now()

		// Stream the file to disk using io.Copy
		written, err := io.Copy(dst, file)
		if err != nil {
			logger.Error("failed to save file", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file: %v", err)})
			close(perfDone)
			return
		}

		copyDuration := time.Since(copyStartTime)
		totalDuration := time.Since(startTime)

		// Stop performance monitoring
		close(perfDone)

		// Calculate transfer rate in bytes per second
		transferRate := float64(written) / copyDuration.Seconds()

		// Calculate memory usage
		memoryUsed := maxMem - initialMemory

		numGoroutines := runtime.NumGoroutine()

		logger.Info("file saved successfully",
			"name", fileName,
			"size", written,
			"destination", destPath,
			"copy_duration", copyDuration,
			"total_duration", totalDuration,
			"transfer_rate_mbps", transferRate/1024/1024,
			"max_memory_mb", float64(maxMem)/1024/1024,
			"memory_used_mb", float64(memoryUsed)/1024/1024,
			"cpu_usage_pct", avgCPU,
			"goroutines", numGoroutines,
		)

		// Create benchmark result
		result := BenchmarkResult{
			Timestamp:     time.Now(),
			FileName:      fileName,
			FileSize:      written,
			Duration:      copyDuration,
			TransferRate:  transferRate,
			MemoryUsage:   memoryUsed,
			CPUUsage:      avgCPU,
			NumGoroutines: numGoroutines,
		}

		// Append benchmark to file
		go func(r BenchmarkResult) {
			benchmarkMutex.Lock()
			defer benchmarkMutex.Unlock()

			f, err := os.OpenFile("benchmark.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Error("failed to open benchmark file", "error", err)
				return
			}
			defer f.Close()

			benchmarkLine := fmt.Sprintf(
				"[%s] File: %s, Size: %d bytes, Duration: %v, "+
					"Transfer Rate: %.2f MB/s, Memory Used: %.2f MB, CPU Usage: %.1f%%, Goroutines: %d\n",
				r.Timestamp.Format(time.RFC3339),
				r.FileName,
				r.FileSize,
				r.Duration,
				r.TransferRate/1024/1024,
				float64(r.MemoryUsage)/1024/1024,
				r.CPUUsage,
				r.NumGoroutines,
			)

			if _, err := f.WriteString(benchmarkLine); err != nil {
				logger.Error("failed to write to benchmark file", "error", err)
			}
		}(result)

		c.JSON(http.StatusOK, gin.H{
			"status":         "success",
			"destination":    destPath,
			"size":           written,
			"duration_ms":    copyDuration.Milliseconds(),
			"transfer_rate":  fmt.Sprintf("%.2f MB/s", transferRate/1024/1024),
			"memory_used_mb": fmt.Sprintf("%.2f MB", float64(memoryUsed)/1024/1024),
			"cpu_usage":      fmt.Sprintf("%.1f%%", avgCPU),
		})
	})

	// Run the server
	if err := router.Run(":8080"); err != nil {
		logger.Fatal("server failed to start", "error", err)
	}
}
