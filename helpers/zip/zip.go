package zip

import (
	stdzip "archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Zip struct{}

func NewZip() *Zip {
	return &Zip{}
}

func (Zip) CreateFromDir(sourceDir, destinationPath string, prefix string, extraFiles map[string][]byte) error {
	out, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer out.Close()

	writer := stdzip.NewWriter(out)
	defer writer.Close()

	cleanSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to resolve source directory: %w", err)
	}

	for name, content := range extraFiles {
		entry, err := writer.Create(filepath.ToSlash(name))
		if err != nil {
			return fmt.Errorf("failed to create zip entry %s: %w", name, err)
		}
		if _, err := entry.Write(content); err != nil {
			return fmt.Errorf("failed to write zip entry %s: %w", name, err)
		}
	}

	return filepath.WalkDir(cleanSource, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(cleanSource, filePath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("illegal file path detected: %s", filePath)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		entryName := filepath.ToSlash(filepath.Join(prefix, relPath))
		header, err := stdzip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header: %w", err)
		}
		header.Name = entryName
		header.Method = stdzip.Deflate

		entry, err := writer.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip entry %s: %w", entryName, err)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", filePath, err)
		}
		defer file.Close()

		if _, err := io.Copy(entry, file); err != nil {
			return fmt.Errorf("failed to write zip entry %s: %w", entryName, err)
		}
		return nil
	})
}
