package generator

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// FilePattern defines the pattern of the file content
type FilePattern string

const (
	// RandomPattern generates completely random data
	RandomPattern FilePattern = "random"
	// ZeroPattern generates files filled with zeros
	ZeroPattern FilePattern = "zeros"
	// RepeatingPattern generates files with repeating patterns
	RepeatingPattern FilePattern = "repeating"
)

// FileGenerator handles creation of test files
type FileGenerator struct {
	tempDir    string
	fileSize   int64
	fileCount  int
	filePrefix string
	pattern    FilePattern
}

// NewFileGenerator creates a new file generator
func NewFileGenerator(fileSize int64, fileCount int, pattern FilePattern) (*FileGenerator, error) {
	tempDir, err := os.MkdirTemp("", "s3perftest")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &FileGenerator{
		tempDir:    tempDir,
		fileSize:   fileSize,
		fileCount:  fileCount,
		filePrefix: fmt.Sprintf("test-%s", time.Now().Format("20060102-150405")),
		pattern:    pattern,
	}, nil
}

// GenerateFiles creates test files and returns their paths
func (g *FileGenerator) GenerateFiles() ([]string, error) {
	filePaths := make([]string, 0, g.fileCount)

	for i := 0; i < g.fileCount; i++ {
		fileName := fmt.Sprintf("%s-%03d.dat", g.filePrefix, i)
		filePath := filepath.Join(g.tempDir, fileName)

		if err := g.createFile(filePath); err != nil {
			// Clean up any created files
			g.CleanupFiles()
			return nil, fmt.Errorf("failed to create file %s: %w", fileName, err)
		}

		filePaths = append(filePaths, filePath)
	}

	return filePaths, nil
}

// createFile creates a single test file with the specified pattern
func (g *FileGenerator) createFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	switch g.pattern {
	case RandomPattern:
		if _, err := io.CopyN(file, rand.Reader, g.fileSize); err != nil {
			return err
		}
	case ZeroPattern:
		zeroBuf := make([]byte, 8192) // 8KB buffer for writing zeros
		remaining := g.fileSize

		for remaining > 0 {
			writeSize := remaining
			if writeSize > int64(len(zeroBuf)) {
				writeSize = int64(len(zeroBuf))
			}

			if _, err := file.Write(zeroBuf[:writeSize]); err != nil {
				return err
			}

			remaining -= writeSize
		}
	case RepeatingPattern:
		// Create a pattern like "abcdefghijklmnopqrstuvwxyz0123456789"
		pattern := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
		patternBuf := bytes.NewReader(bytes.Repeat(pattern, int(g.fileSize/int64(len(pattern))+1)))

		if _, err := io.CopyN(file, patternBuf, g.fileSize); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported file pattern: %s", g.pattern)
	}

	return nil
}

// CleanupFiles removes all generated files and the temporary directory
func (g *FileGenerator) CleanupFiles() error {
	return os.RemoveAll(g.tempDir)
}

// GetFileContent returns the content of a file at the given path for testing
func GetFileContent(filePath string, limit int64) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var buffer bytes.Buffer
	if limit > 0 {
		_, err = io.CopyN(&buffer, file, limit)
	} else {
		_, err = io.Copy(&buffer, file)
	}

	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return buffer.Bytes(), nil
}
