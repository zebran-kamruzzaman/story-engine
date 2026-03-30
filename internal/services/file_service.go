package services

import (
	"fmt"
	"os"
)

// FileService handles all reads and writes of .md scene files.
// No other service or package is permitted to call os.ReadFile or os.WriteFile
// on scene files. All file access flows through this service.
type FileService struct {
	projectDir string
}

// NewFileService creates a FileService rooted at the given project directory.
func NewFileService(projectDir string) *FileService {
	return &FileService{projectDir: projectDir}
}

// ReadScene reads the raw content of a .md file at the given absolute path.
// It returns the content as a UTF-8 string.
func (s *FileService) ReadScene(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("file: read %q: %w", filePath, err)
	}
	return string(data), nil
}

// WriteScene writes content to a .md file using an atomic temp-file-then-rename
// pattern. If the write succeeds but the process crashes before rename, the original
// file is preserved. If rename succeeds, the new content is guaranteed on disk.
func (s *FileService) WriteScene(filePath string, content string) error {
	tmp := filePath + ".tmp"

	// Write to temp file first.
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return fmt.Errorf("file: write tmp %q: %w", tmp, err)
	}

	// Atomic rename to the real path.
	// On the same filesystem this is guaranteed atomic on Windows, macOS, and Linux.
	if err := os.Rename(tmp, filePath); err != nil {
		// Clean up the tmp file if rename fails.
		_ = os.Remove(tmp)
		return fmt.Errorf("file: rename %q → %q: %w", tmp, filePath, err)
	}

	return nil
}

// EnsureDir creates the given directory and all parents if they do not exist.
// It is idempotent — calling it on an existing directory is a no-op.
func (s *FileService) EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("file: ensure dir %q: %w", path, err)
	}
	return nil
}

// ListSceneFiles returns the absolute paths of all .md files inside the scenes directory.
func (s *FileService) ListSceneFiles(scenesDir string) ([]string, error) {
	entries, err := os.ReadDir(scenesDir)
	if err != nil {
		return nil, fmt.Errorf("file: list scenes dir %q: %w", scenesDir, err)
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 3 && name[len(name)-3:] == ".md" {
			paths = append(paths, scenesDir+string(os.PathSeparator)+name)
		}
	}
	return paths, nil
}
