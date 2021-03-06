package filefilter

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/HansK-p/go-customtypes"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	Pattern   *customtypes.Regexp `yaml:"pattern"`
	MinAge    time.Duration       `yaml:"min_age"`
	MaxAge    time.Duration       `yaml:"max_age"`
	MinSize   int64               `yaml:"min_size"`
	MaxSize   int64               `yaml:"max_size"`
	MinSizeMB float64             `yaml:"min_size_mb"`
	MaxSizeMB float64             `yaml:"max_size_mb"`
}

func PassesFilter(logger *log.Entry, config *Configuration, fileInfo fs.FileInfo) (bool, string, error) {
	logger = logger.WithFields(log.Fields{"Package": "filefilter", "Function": "passesFilter", "Filter": config})
	if config.Pattern != nil && !config.Pattern.Match([]byte(fileInfo.Name())) {
		logger.Debugf("File did not pass the pattern filter")
		return false, "pattern", nil
	}
	if config.MaxAge.Milliseconds() != 0 && fileInfo.ModTime().Add(config.MaxAge).Before(time.Now()) {
		logger.Debugf("File did not pass the max_age filter")
		return false, "max_age", nil
	}
	if config.MinAge.Milliseconds() != 0 && fileInfo.ModTime().Add(config.MinAge).After(time.Now()) {
		logger.Debugf("File did not pass the min_age filter")
		return false, "min_age", nil
	}
	if config.MinSize != 0 && fileInfo.Size() < config.MinSize {
		logger.Debugf("File did not pass the min_size filter")
		return false, "min_size", nil
	}
	if config.MaxSize != 0 && fileInfo.Size() > config.MaxSize {
		logger.Debugf("File did not pass the max_size filter")
		return false, "max_size", nil
	}
	if config.MinSizeMB != 0 && float64(fileInfo.Size()/1024/1024) < config.MinSizeMB {
		logger.Debugf("File did not pass the min_size_mb filter")
		return false, "min_size_mb", nil
	}
	if config.MaxSizeMB != 0 && float64(fileInfo.Size()/1024/1024) > config.MaxSizeMB {
		logger.Debugf("File did not pass the max_size_mb filter")
		return false, "max_size_mb", nil
	}
	return true, "", nil
}

func ReadDir(logger *log.Entry, config *Configuration, dirPath string) (files []fs.FileInfo, err error) {
	allFiles, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("when listing files in the folder '%s': %w", dirPath, err)
	}

	for _, fileInfo := range allFiles {
		logger := logger.WithFields(log.Fields{"Filename": fileInfo.Name()})
		passes, condition, err := PassesFilter(logger, config, fileInfo)
		if err != nil {
			return nil, fmt.Errorf("when using the filter on the file '%s': %w", fileInfo.Name(), err)
		}
		if !passes {
			logger.Debugf("The file did not pass the '%s' condition", condition)
		} else {
			files = append(files, fileInfo)
		}
	}

	return files, nil
}

func WalkDir(logger *log.Entry, config *Configuration, dirPath string) (files map[string]fs.FileInfo, err error) {
	files = make(map[string]fs.FileInfo)
	if err := filepath.Walk(dirPath,
		func(filePath string, fileInfo os.FileInfo, err error) error {
			logger := logger.WithFields(log.Fields{"FilePath": filePath})
			if err != nil {
				return err
			}
			if fileInfo.IsDir() {
				return nil // Traverse into the folder
			}
			passes, condition, err := PassesFilter(logger, config, fileInfo)
			if err != nil {
				return fmt.Errorf("when using the filter on the file '%s': %w", fileInfo.Name(), err)
			}
			if !passes {
				logger.Debugf("The file did not pass the '%s' condition", condition)
			} else {
				files[filePath] = fileInfo
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("when walking the file system: %w", err)
	}
	return files, nil
}

type FilterMatch struct {
	FileInfo fs.FileInfo
	FilePath string
	Config   *Configuration
}

func ReadDirMatches(logger *log.Entry, configs []Configuration, dirPath string) (filterMatches []FilterMatch, err error) {
	for idx := range configs {
		config := &configs[idx]
		files, err := ReadDir(logger, config, dirPath)
		if err != nil {
			return nil, fmt.Errorf("when applying filter configuration '%#v': %w", config, err)
		}
		for _, file := range files {
			filterMatches = append(filterMatches, FilterMatch{
				FileInfo: file,
				FilePath: path.Join(dirPath, file.Name()),
				Config:   config,
			})
		}
	}
	return filterMatches, err
}

func WalkDirMatches(logger *log.Entry, configs []Configuration, dirPath string) (filterMatches []FilterMatch, err error) {
	for idx := range configs {
		config := &configs[idx]
		files, err := WalkDir(logger, config, dirPath)
		if err != nil {
			return nil, fmt.Errorf("when applying filter configuration '%#v': %w", config, err)
		}
		for filePath, file := range files {
			filterMatches = append(filterMatches, FilterMatch{
				FileInfo: file,
				FilePath: filePath,
				Config:   config,
			})
		}
	}
	return filterMatches, err
}
