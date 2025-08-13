package processors

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Chromfalke/audio-workbench/internal/commands"
	"github.com/Chromfalke/audio-workbench/internal/lib"
)

type Processor interface {
	Run(file lib.Audiofile, outpath string) error
}

// Processor to normalize the loudness of an audio file
type Normalizer struct {
	TargetLoudness float64
}

func (normalizer Normalizer) Run(file lib.Audiofile, outpath string) error {
	sampleRate, err := commands.ExtractSampleRate(file.Path)
	if err != nil {
		return fmt.Errorf("Failed to extract the sample rate from %s: %s\n", file.Path, err)
	}
	bitrate, err := commands.ExtractBitrate(file)
	if err != nil {
		return fmt.Errorf("Failed to extract the bitrate from %s: %s", file.Path, err)
	}
	loudnessInfo, err := commands.ExtractLoudnessInfo(file.Path)
	if err != nil {
		return fmt.Errorf("Failed to extract the loudness from %s: %s", file.Path, err)
	}

	err = commands.NormalizeLoudness(file, outpath, normalizer.TargetLoudness, loudnessInfo, sampleRate, bitrate)
	if err != nil {
		return fmt.Errorf("Failed to normalize the loudness of %s: %s\n", file.Path, err)
	}

	return nil
}

// Processor to convert the audio file to a different format
type Converter struct {
	Format string
}

func (converter Converter) Run(file lib.Audiofile, outpath string) error {
	ext := filepath.Ext(outpath)
	outpath = strings.TrimRight(outpath, ext) + "." + converter.Format

	sampleRate, err := commands.ExtractSampleRate(file.Path)
	if err != nil {
		return fmt.Errorf("Failed to extract the sample rate from %s: %s\n", file.Path, err)
	}
	bitrate, err := commands.ExtractBitrate(file)
	if err != nil {
		return fmt.Errorf("Failed to extract the bitrate from %s: %s", file.Path, err)
	}
	err = commands.Convert(file, outpath, sampleRate, bitrate)
	if err != nil {
		return fmt.Errorf("Failed to convert %s to %s: %s", file.Path, converter.Format, err)
	}

	return nil
}

// Processor to resample an audio file
type Resampler struct {
	SampleRate int
}

func (resampler Resampler) Run(file lib.Audiofile, outpath string) error {
	bitrate, err := commands.ExtractBitrate(file)
	if err != nil {
		return fmt.Errorf("Failed to extract the bitrate from %s: %s", file.Path, err)
	}
	err = commands.Resample(file, outpath, resampler.SampleRate, bitrate)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Failed to resample the %s: %s", file.Path, err)
	}

	return nil
}
