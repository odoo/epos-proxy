package util

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"epos-proxy/logger"
)

func ZipLogs(sourceDir, targetZip string) error {

	zipFile, err := os.Create(targetZip)
	if err != nil {
		logger.Log.Errorf("Failed to create zip file %s: %v", targetZip, err)
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		logger.Log.Errorf("Failed to read source directory %s: %v", sourceDir, err)
		return err
	}

	for _, file := range files {

		path := filepath.Join(sourceDir, file.Name())

		src, err := os.Open(path)
		if err != nil {
			logger.Log.Errorf("Failed to open file %s: %v", path, err)
			return err
		}
		defer src.Close()

		w, err := zipWriter.Create(file.Name())
		if err != nil {
			logger.Log.Errorf("Failed to create entry %s in zip: %v", file.Name(), err)
			return err
		}

		_, err = io.Copy(w, src)
		if err != nil {
			logger.Log.Errorf("Failed to copy file %s to zip: %v", path, err)
			return err
		}
	}

	return nil
}
