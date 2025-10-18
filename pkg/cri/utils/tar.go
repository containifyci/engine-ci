package utils

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// extractTar extracts a tar archive from the provided io.Reader to the target directory.
func ExtractTar(r io.Reader, targetDir string) error {
	// Create a tar reader from the input reader
	tarReader := tar.NewReader(r)

	// Iterate over the files in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// End of archive
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar archive: %w", err)
		}

		// Determine the target file path
		// Resolve the target path and ensure it is within the target directory
		targetPath := filepath.Join(targetDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in tar archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Ensure the directory for the file exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for file: %w", err)
			}

			// Create the file on disk
			file, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			// Copy file contents from tar to disk
			if _, err := io.Copy(file, tarReader); err != nil {
				return fmt.Errorf("failed to copy file contents: %w", err)
			}
		default:
			// Handle other file types if necessary
			fmt.Printf("Unknown file type in tar: %c in file %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}
