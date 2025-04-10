package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

// BenchmarkResult represents performance data of file uploads
type BenchmarkResult struct {
	Timestamp    time.Time
	FileName     string
	FileSize     int64
	Duration     time.Duration
	TransferRate float64 // bytes per second
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

	// Create upload endpoint
	router.POST("/upload", func(c *gin.Context) {
		startTime := time.Now()
		logger.Info("upload request received", "client_ip", c.ClientIP())

		// Get destination path from query parameter
		destPath := c.Query("dest")
		if destPath == "" {
			logger.Error("destination path not provided")
			c.JSON(http.StatusBadRequest, gin.H{"error": "destination path is required"})
			return
		}

		logger.Info("processing upload", "destination", destPath)

		// Ensure destination directory exists
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			logger.Error("failed to create directory", "error", err, "path", destDir)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create directory: %v", err)})
			return
		}

		// Get file from request
		fileHeader, err := c.FormFile("file")
		if err != nil {
			logger.Error("failed to get file from request", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to get file: %v", err)})
			return
		}

		fileName := fileHeader.Filename
		fileSize := fileHeader.Size
		logger.Info("file received", "name", fileName, "size", fileSize)

		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			logger.Error("failed to open uploaded file", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to open file: %v", err)})
			return
		}
		defer file.Close()

		// Create destination file
		dst, err := os.Create(destPath)
		if err != nil {
			logger.Error("failed to create destination file", "error", err, "path", destPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create destination file: %v", err)})
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
			return
		}
		
		copyDuration := time.Since(copyStartTime)
		totalDuration := time.Since(startTime)
		
		// Calculate transfer rate in bytes per second
		transferRate := float64(written) / copyDuration.Seconds()

		logger.Info("file saved successfully",
			"name", fileName,
			"size", written,
			"destination", destPath,
			"copy_duration", copyDuration,
			"total_duration", totalDuration,
			"transfer_rate_mbps", transferRate/1024/1024,
		)

		// Create benchmark result
		result := BenchmarkResult{
			Timestamp:    time.Now(),
			FileName:     fileName,
			FileSize:     written,
			Duration:     copyDuration,
			TransferRate: transferRate,
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
				"[%s] File: %s, Size: %d bytes, Duration: %v, Transfer Rate: %.2f MB/s\n",
				r.Timestamp.Format(time.RFC3339),
				r.FileName,
				r.FileSize,
				r.Duration,
				r.TransferRate/1024/1024,
			)

			if _, err := f.WriteString(benchmarkLine); err != nil {
				logger.Error("failed to write to benchmark file", "error", err)
			}
		}(result)

		c.JSON(http.StatusOK, gin.H{
			"status":        "success",
			"destination":   destPath,
			"size":          written,
			"duration_ms":   copyDuration.Milliseconds(),
			"transfer_rate": fmt.Sprintf("%.2f MB/s", transferRate/1024/1024),
		})
	})

	// Run the server
	logger.Info("server starting", "address", ":8080")
	if err := router.Run(":8080"); err != nil {
		logger.Fatal("server failed to start", "error", err)
	}
}
