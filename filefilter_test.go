package filefilter

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/HansK-p/go-utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type FileSpecification struct {
	Folder   string
	Name     string
	Modified time.Time
	Size     int64
}

type Filter struct {
	Configurations []Configuration `yaml:"filters"`
}

func prepare(fileSpecifications []FileSpecification) (tmpDir string, err error) {
	tmpDir, err = os.MkdirTemp("", "testfilefilter")
	if err != nil {
		return tmpDir, fmt.Errorf("unable to create the temp folder for testing: %w", err)
	}
	for idx := range fileSpecifications {
		fileSpecification := &fileSpecifications[idx]
		folderPath := path.Join(tmpDir, fileSpecification.Folder)
		if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
			return tmpDir, fmt.Errorf("when creating folder '%s': %s", folderPath, err)
		}
		filePath := path.Join(folderPath, fileSpecification.Name)
		file, err := os.Create(filePath)
		if err != nil {
			return tmpDir, fmt.Errorf("when creating file '%s': %s", filePath, err)
		}
		defer file.Close()
		for idx := 0; idx < int(fileSpecification.Size); idx++ {
			if n, err := file.Write([]byte("a")); err != nil {
				return tmpDir, fmt.Errorf("when writing to file '%s': %s", filePath, err)
			} else if n != 1 {
				return tmpDir, fmt.Errorf("writing one byte to file '%s' succedded, but '%d' byte(s) was written", filePath, n)
			}
		}
		if err := file.Sync(); err != nil {
			return tmpDir, fmt.Errorf("when doing file synd of file '%s': %s", filePath, err)
		}
		if !fileSpecification.Modified.IsZero() {
			if err := os.Chtimes(filePath, fileSpecification.Modified, fileSpecification.Modified); err != nil {
				return tmpDir, fmt.Errorf("when changing modification time of '%s' to %v", filePath, fileSpecification.Modified)
			}
		}
	}
	return tmpDir, nil
}

func TestFileFilter(t *testing.T) {
	// Prepare the test
	now := time.Now()
	fileSpecifications := []FileSpecification{
		{Folder: "readdir", Name: "test_1hourago_10b_size.txt", Modified: now.Add(-time.Hour), Size: 10},
	}
	tmpDir, err := prepare(fileSpecifications)
	if err != nil {
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
		t.Fatalf("when initializing the folder structure: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	// Perform tests
	logger := utils.NewLogger().WithFields(log.Fields{})
	yamlText := []byte(`---
filters:
- pattern: ^.*\.txt$
  min_age: 30m
  min_size: 11
`)
	filter := Filter{}
	if err := yaml.Unmarshal(yamlText, &filter); err != nil {
		t.Fatalf("when unmarshaling '%s': %s", string(yamlText), err)
	}
	t.Logf("Filter: %#v", filter)
	if filterMatches, err := ReadDirMatches(logger, filter.Configurations, path.Join(tmpDir, "readdir")); err != nil {
		t.Errorf("received error: %s", err)
	} else if len(filterMatches) != 0 {
		t.Errorf("Expected 1 match, but got %d (%#v)", len(filterMatches), filterMatches)
	}
}
