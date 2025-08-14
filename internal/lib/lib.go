package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Mediafile struct {
	Path   string
	IsOpus bool
}

/*
 * Various helper functions
 */

func CreateOutputDir(outputDir string) error {
	if outputDir != "" {
		err := os.MkdirAll(filepath.Dir(outputDir), 0775)
		return err
	}
	return nil
}

func CollectInputFiles(input string) ([]Mediafile, error) {
	inputInfo, err := os.Stat(input)
	if err != nil {
		return []Mediafile{}, err
	}

	var files []Mediafile
	if inputInfo.IsDir() {
		entries, err := os.ReadDir(input)
		if err != nil {
			return []Mediafile{}, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				file := Mediafile{
					Path:   filepath.Join(input, entry.Name()),
					IsOpus: strings.HasSuffix(entry.Name(), ".opus") || strings.HasSuffix(entry.Name(), ".ogg"),
				}
				files = append(files, file)
			}
		}
	} else {
		files = []Mediafile{Mediafile{
			Path:   input,
			IsOpus: strings.HasSuffix(input, ".opus") || strings.HasSuffix(input, ".ogg"),
		}}
	}

	return files, nil
}

func BuildOutputPath(file Mediafile, outputDir string) string {
	if outputDir == "" {
		ext := filepath.Ext(file.Path)
		return fmt.Sprintf("temp%s", ext)
	} else if filepath.Ext(outputDir) == "" {
		return filepath.Join(outputDir, filepath.Base(file.Path))
	}

	return outputDir
}

func RenameTempFile(file Mediafile, outpath string) error {
	tempfile := fmt.Sprintf("temp%s", filepath.Ext(outpath))
	if filepath.Base(outpath) == "temp"+filepath.Ext(outpath) {
		err := os.Rename(tempfile, file.Path)
		return fmt.Errorf("Failed to overwrite the original file for %s: %s\n", file.Path, err)
	}

	return nil
}
