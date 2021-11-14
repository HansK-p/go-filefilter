package filefilter

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"time"

	"github.com/HansK-p/go-customtypes"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	MinAge  time.Duration       `yaml:"min_age"`
	MaxAge  time.Duration       `yaml:"max_age"`
	Pattern *customtypes.Regexp `yaml:"pattern"`
}

func PassesFilter(logger *log.Entry, fileInfo fs.FileInfo, config *Configuration) (bool, string, error) {
	logger = logger.WithFields(log.Fields{"Package": "filefilter", "Function": "passesFilter", "Filter": config})
	if config.Pattern != nil && !config.Pattern.Match([]byte(fileInfo.Name())) {
		logger.Debugf("File did not pass the pattern filter")
		return false, "pattern", nil
	}
	if config.MaxAge.Milliseconds() != 0 && fileInfo.ModTime().Add(config.MaxAge).Before(time.Now()) {
		logger.Debugf("File did not pass the maxAge filter")
		return false, "max_age", nil
	}
	if config.MinAge.Milliseconds() != 0 && fileInfo.ModTime().Add(config.MinAge).After(time.Now()) {
		logger.Debugf("File did not pass the minAge filter")
		return false, "min_age", nil
	}
	return true, "", nil
}

func ReadDir(logger *log.Entry, dirPath string, config *Configuration) ([]fs.FileInfo, error) {
	allFiles, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("when listing files in the folder '%s': %w", dirPath, err)
	}

	files := []fs.FileInfo{}
	for _, fileInfo := range allFiles {
		logger := logger.WithFields(log.Fields{"Filename": fileInfo.Name()})
		passes, condition, err := PassesFilter(logger, fileInfo, config)
		if err != nil {
			return nil, fmt.Errorf("when using the filter on the file '%s': %w", fileInfo.Name(), err)
		}
		if !passes {
			logger.Debugf("The file did not pass the '%s' condition", condition)
		}
		files = append(files, fileInfo)
	}

	return files, nil
}
