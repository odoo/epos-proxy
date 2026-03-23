package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ZipLogs(sourceDir, targetZip string) error {
	zipFile, err := os.Create(targetZip)
	if err != nil {
		return fmt.Errorf("failed to create zip file %s: %w", targetZip, err)
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", sourceDir, err)
	}
	for _, file := range files {
		if err := zipEntry(zipWriter, sourceDir, file); err != nil {
			return fmt.Errorf("failed to zip entry %s: %w", file.Name(), err)
		}
	}
	return nil
}

func zipEntry(zipWriter *zip.Writer, sourceDir string, file os.DirEntry) error {
	path := filepath.Join(sourceDir, file.Name())
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer src.Close()
	w, err := zipWriter.Create(file.Name())
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}
	if _, err = io.Copy(w, src); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}
	return nil
}
