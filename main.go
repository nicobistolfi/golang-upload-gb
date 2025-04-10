package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create a Gin router with default middleware
	router := gin.Default()

	// Increase the max multipart memory
	router.MaxMultipartMemory = 8 << 20 // 8 MiB, just for initial buffer

	// Create upload endpoint
	router.POST("/upload", func(c *gin.Context) {
		// Get destination path from query parameter
		destPath := c.Query("dest")
		if destPath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "destination path is required"})
			return
		}

		// Ensure destination directory exists
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create directory: %v", err)})
			return
		}

		// Get file from request
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to get file: %v", err)})
			return
		}

		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to open file: %v", err)})
			return
		}
		defer file.Close()

		// Create destination file
		dst, err := os.Create(destPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create destination file: %v", err)})
			return
		}
		defer dst.Close()

		// Stream the file to disk using io.Copy
		written, err := io.Copy(dst, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save file: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":      "success",
			"destination": destPath,
			"size":        written,
		})
	})

	// Run the server
	log.Println("Server started on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
