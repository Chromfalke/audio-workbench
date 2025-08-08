package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Audiofile struct {
	Path   string
	IsOpus bool
}

/*
 * Various helper functions
 */

func createOutputDir(outputDir string) error {
	if outputDir != "" {
		err := os.MkdirAll(outputDir, 0775)
		return err
	}
	return nil
}

func collectInputFiles(input string) ([]Audiofile, error) {
	inputInfo, err := os.Stat(input)
	if err != nil {
		return []Audiofile{}, err
	}

	var files []Audiofile
	if inputInfo.IsDir() {
		entries, err := os.ReadDir(input)
		if err != nil {
			return []Audiofile{}, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				file := Audiofile{
					Path:   filepath.Join(input, entry.Name()),
					IsOpus: strings.HasSuffix(entry.Name(), ".opus") || strings.HasSuffix(entry.Name(), ".ogg"),
				}
				files = append(files, file)
			}
		}
	} else {
		files = []Audiofile{Audiofile{
			Path:   input,
			IsOpus: strings.HasSuffix(input, ".opus") || strings.HasSuffix(input, ".ogg"),
		}}
	}

	return files, nil
}

func buildOutputPath(file Audiofile, outputDir string) string {
	if outputDir == "" {
		ext := filepath.Ext(file.Path)
		return fmt.Sprintf("temp%s", ext)
	} else {
		return filepath.Join(outputDir, filepath.Base(file.Path))
	}
}
