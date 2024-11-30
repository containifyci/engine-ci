package filesystem

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FileCache struct {
	cache       bool
	fileName    string
	FileResults map[string]FileResult `yaml:"file_results"`
}

type FileResult struct {
	Query string   `yaml:"query"`
	Paths []string `yaml:"paths"`
}

func NewFileCache(fileName string) *FileCache {
	dir := os.TempDir()
	slog.Info("Creating file cache", "file", dir+fileName)
	fc := &FileCache{
		cache:       false,
		fileName:    dir + fileName,
		FileResults: make(map[string]FileResult)}

	if fc.cache {
		_, err := fc.LoadResultsFromYAML()
		if err != nil {
			slog.Error("Error loading results from YAML", "error", err)
			os.Exit(1)
		}
	}

	return fc
}

func (fc *FileCache) LoadResultsFromYAML() (map[string]FileResult, error) {
	var results map[string]FileResult

	data, err := os.ReadFile(fc.fileName)
	if err != nil {
		return results, err
	}

	err = yaml.Unmarshal(data, &results)
	if err != nil {
		return results, err
	}

	fc.FileResults = results

	return results, nil
}

func (fc *FileCache) FindFilesBySuffix(root, fileSuffix string) ([]string, error) {
	if fc.FileResults == nil {
		fc.FileResults = make(map[string]FileResult)
	}

	if result, ok := fc.FileResults[fileSuffix]; ok {
		return result.Paths, nil
	}

	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), fileSuffix) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		slog.Error("Error finding files by suffix", "error", err, "suffix", fileSuffix)
		return nil, err
	}

	if fc.cache {
		err = fc.SaveResultsAsYAML(fileSuffix, files)
		if err != nil {
			slog.Error("Error saving results as YAML", "error", err)
			return nil, err
		}
	}

	return files, err
}

func (fc *FileCache) SaveResultsAsYAML(fileSuffix string, files []string) error {
	results := FileResult{Query: fileSuffix, Paths: files}
	fc.FileResults[fileSuffix] = results

	data, err := yaml.Marshal(&fc.FileResults)
	if err != nil {
		return err
	}

	err = os.WriteFile(fc.fileName, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Error getting home directory", "error", err)
		os.Exit(1)
	}
	return home
}
