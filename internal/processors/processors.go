package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chromfalke/audio-workbench/internal/commands"
	"github.com/Chromfalke/audio-workbench/internal/lib"
)

type Processor interface {
	Run(file lib.Mediafile, outpath string) error
}

// Processor to normalize the loudness of an audio file
type Normalizer struct {
	TargetLoudness float64
}

func (normalizer Normalizer) Run(file lib.Mediafile, outpath string) error {
	var hasCover bool
	var err error
	if file.IsOpus {
		hasCover, err = commands.ExtractCover(file, "cover.jpg", "")
		if err != nil {
			return fmt.Errorf("Failed to extract the cover from %s: %s\n", file.Path, err)
		}
	}

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

	if file.IsOpus && hasCover {
		err := commands.SetCover(file, "cover.jpg")
		if err != nil {
			return fmt.Errorf("Failed to set cover for %s: %s\n", file.Path, err)
		}
		err = os.Remove("cover.jpg")
		if err != nil {
			return fmt.Errorf("Unable to remove temporary cover.jpg file: %s", err)
		}
	}

	return nil
}

// Processor to convert the audio file to a different format
type Converter struct {
	Format string
}

func (converter Converter) Run(file lib.Mediafile, outpath string) error {
	var hasCover bool
	var err error
	if file.IsOpus {
		hasCover, err = commands.ExtractCover(file, "cover.jpg", "")
		if err != nil {
			return fmt.Errorf("Failed to extract the cover from %s: %s\n", file.Path, err)
		}
	}

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

	if file.IsOpus && hasCover {
		err := commands.SetCover(file, "cover.jpg")
		if err != nil {
			return fmt.Errorf("Failed to set cover for %s: %s\n", file.Path, err)
		}
		err = os.Remove("cover.jpg")
		if err != nil {
			return fmt.Errorf("Unable to remove temporary cover.jpg file: %s", err)
		}
	}

	return nil
}

// Processor to resample an audio file
type Resampler struct {
	SampleRate int
}

func (resampler Resampler) Run(file lib.Mediafile, outpath string) error {
	var hasCover bool
	var err error
	if file.IsOpus {
		hasCover, err = commands.ExtractCover(file, "cover.jpg", "")
		if err != nil {
			return fmt.Errorf("Failed to extract the cover from %s: %s\n", file.Path, err)
		}
	}

	bitrate, err := commands.ExtractBitrate(file)
	if err != nil {
		return fmt.Errorf("Failed to extract the bitrate from %s: %s", file.Path, err)
	}
	err = commands.Resample(file, outpath, resampler.SampleRate, bitrate)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Failed to resample the %s: %s", file.Path, err)
	}

	if file.IsOpus && hasCover {
		err := commands.SetCover(file, "cover.jpg")
		if err != nil {
			return fmt.Errorf("Failed to set cover for %s: %s\n", file.Path, err)
		}
		err = os.Remove("cover.jpg")
		if err != nil {
			return fmt.Errorf("Unable to remove temporary cover.jpg file: %s", err)
		}
	}

	return nil
}

// Processor to extract the cover image
type CoverImageExtractor struct {
	ImageFormat string
}

func (extractor CoverImageExtractor) Run(file lib.Mediafile, outpath string) error {
	if filepath.Ext(file.Path) == ".wav" {
		// skip .wav files since they don't have a cover
		return nil
	}

	var imagePath string
	if outpath == fmt.Sprintf("temp%s", filepath.Ext(file.Path)) {
		imagePath = strings.ReplaceAll(file.Path, filepath.Ext(file.Path), extractor.ImageFormat)
	} else {
		imagePath = strings.ReplaceAll(outpath, filepath.Ext(outpath), extractor.ImageFormat)
	}

	hasCover, err := commands.ExtractCover(file, imagePath, "")
	if err != nil {
		return fmt.Errorf("Failed to extract the cover from %s: %s", file.Path, err)
	}
	if hasCover {
		fmt.Printf("Extracted the cover to %s.\n", imagePath)
	} else {
		fmt.Println("No cover could be extracted from ", file.Path)
	}

	return nil
}

// Processor to set the cover image
type CoverImageSetter struct {
	CoverImage string
}

func (setter CoverImageSetter) Run(file lib.Mediafile, outpath string) error {
	if filepath.Ext(file.Path) == ".wav" {
		// skip .wav files since they don't have a cover
		return nil
	}

	err := commands.SetCover(file, setter.CoverImage)
	if err != nil {
		return fmt.Errorf("Failed to set %s as cover for %s: %s", setter.CoverImage, file.Path, err)
	}

	return nil
}

// Processor to extract the audio from a video
type AudioExtractor struct {
	AudioFormat    string
	CopyCover      bool
	VideoTimestamp string
}

func (extractor AudioExtractor) Run(file lib.Mediafile, outpath string) error {
	if !file.IsVideo {
		return nil
	}

	var audioPath string
	if outpath == fmt.Sprintf("temp%s", filepath.Ext(file.Path)) {
		audioPath = strings.ReplaceAll(file.Path, filepath.Ext(file.Path), extractor.AudioFormat)
	} else {
		audioPath = strings.ReplaceAll(outpath, filepath.Ext(outpath), extractor.AudioFormat)
	}

	err := commands.ExtractAudio(file, audioPath)
	if err != nil {
		return fmt.Errorf("Failed to extract the audio from %s: %s", file.Path, err)
	}

	if extractor.CopyCover {
		hasCover, err := commands.ExtractCover(file, "cover.jpg", extractor.VideoTimestamp)
		if err != nil {
			return fmt.Errorf("Failed to extract the cover from %s: %s", file.Path, err)
		}

		if hasCover {
			audioFile := lib.Mediafile{
				Path:    audioPath,
				IsOpus:  extractor.AudioFormat == ".opus",
				IsVideo: false,
			}
			err := commands.SetCover(audioFile, "cover.jpg")
			if err != nil {
				return fmt.Errorf("Failed to set cover for %s: %s\n", file.Path, err)
			}
			err = os.Remove("cover.jpg")
			if err != nil {
				return fmt.Errorf("Unable to remove temporary cover.jpg file: %s", err)
			}
		} else {
			fmt.Println("No cover could be extracted from ", file.Path)
		}
	}

	return nil
}
