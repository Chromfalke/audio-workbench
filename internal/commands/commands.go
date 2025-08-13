package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Chromfalke/audio-workbench/internal/lib"
)

type LoudnessInfo struct {
	I      string `json:"input_i"`
	TP     string `json:"input_tp"`
	LRA    string `json:"input_lra"`
	Thresh string `json:"input_thresh"`
	Offset string `json:"target_offset"`
}

/*
 * Commands used during loudness normalization
 */

// Extract the sample rate of an audio file.
func ExtractSampleRate(file string) (string, error) {
	args := []string{"-v", "error", "-select_streams", "a:0", "-show_entries", "stream=sample_rate", "-of", "default=noprint_wrappers=1:nokey=1", file}
	ffmpeg := exec.Command("ffprobe", args...)
	output, err := ffmpeg.Output()
	if err != nil {
		return "", err
	}
	return strings.Trim(string(output), "\n"), nil
}

// Extract the bitrate of an audio file.
func ExtractBitrate(file lib.Mediafile) (string, error) {
	if file.IsOpus {
		// return 128kbit/s as a good default for opus
		return "128000", nil
	}

	args := []string{"-v", "error", "-select_streams", "a:0", "-show_entries", "format=bit_rate", "-of", "default=noprint_wrappers=1:nokey=1", file.Path}
	ffmpeg := exec.Command("ffprobe", args...)
	output, err := ffmpeg.Output()
	if err != nil {
		return "", err
	}
	return strings.Trim(string(output), "\n"), nil
}

// First pass with ffmpeg to analyze the loudness of an audio file.
func ExtractLoudnessInfo(file string) (LoudnessInfo, error) {
	ffmpegArgs := []string{"-i", file, "-af", "loudnorm=print_format=json", "-nostats", "-hide_banner", "-f", "null", "-"}
	ffmpeg := exec.Command("ffmpeg", ffmpegArgs...)
	output, err := ffmpeg.CombinedOutput()
	if err != nil {
		return LoudnessInfo{}, err
	}
	start := strings.Index(string(output), "{")
	end := strings.Index(string(output), "}")

	var loudnessInfo LoudnessInfo
	err = json.Unmarshal([]byte(string(output)[start:end+1]), &loudnessInfo)
	if err != nil {
		return LoudnessInfo{}, err
	}

	return loudnessInfo, nil
}

// Second pass with ffmpeg to normalize the loudness.
func NormalizeLoudness(file lib.Mediafile, outpath string, targetLoudness float64, loudnessInfo LoudnessInfo, sampleRate string, bitrate string) error {
	loudnorm := fmt.Sprintf("loudnorm=linear=true:I=%.2f:LRA=7.0:TP=-2.0:offset=%s:measured_I=%s:measured_TP=%s:measured_LRA=%s:measured_thresh=%s", targetLoudness, loudnessInfo.Offset, loudnessInfo.I, loudnessInfo.TP, loudnessInfo.LRA, loudnessInfo.Thresh)
	args := []string{"-i", file.Path, "-af", loudnorm, "-ar", sampleRate, "-b:a", bitrate}
	if !file.IsOpus {
		args = append(args, []string{"-map", "0", "-map_metadata", "0", outpath}...)
	} else {
		args = append(args, []string{"-map_metadata", "0", outpath}...)
	}
	ffmpeg := exec.Command("ffmpeg", args...)
	err := ffmpeg.Run()
	if err != nil {
		return err
	}

	return lib.RenameTempFile(file, outpath)
}

/*
 * Commands used during conversion
 */

// Reformat the audio file
func Convert(file lib.Mediafile, outpath string, sampleRate string, bitrate string) error {
	args := []string{"-i", file.Path, "-ar", sampleRate, "-b:a", bitrate}
	if !file.IsOpus {
		args = append(args, []string{"-map", "0", "-map_metadata", "0", outpath}...)
	} else {
		args = append(args, []string{"-map_metadata", "0", outpath}...)
	}
	ffmpeg := exec.Command("ffmpeg", args...)
	err := ffmpeg.Run()
	if err != nil {
		return err
	}

	return lib.RenameTempFile(file, outpath)
}

/*
 * Commands used during resampling
 */

// Resample an audio file
func Resample(file lib.Mediafile, outpath string, targetSampleRate int, bitrate string) error {
	args := []string{"-i", file.Path, "-ar", fmt.Sprintf("%d", targetSampleRate), "-b:a", bitrate}
	if !file.IsOpus {
		args = append(args, []string{"-map", "0", "-map_metadata", "0", outpath}...)
	} else {
		args = append(args, []string{"-map_metadata", "0", outpath}...)
	}
	ffmpeg := exec.Command("ffmpeg", args...)
	err := ffmpeg.Run()
	if err != nil {
		return err
	}

	return lib.RenameTempFile(file, outpath)
}

/*
 * Commands used during various operations
 */

// Extract the embedded cover.
func ExtractCover(file lib.Mediafile) (bool, error) {
	if file.IsOpus {
		opustags := exec.Command("opustags", "--output-cover", "cover.jpg", file.Path, "-i")
		err := opustags.Run()
		if err != nil {
			return false, err
		}
	} else {
		ffmpeg := exec.Command("ffmpeg", "-i", file.Path, "-an", "-c:v", "copy", "cover.jpg")
		err := ffmpeg.Run()
		if err != nil {
			return false, err
		}
	}

	_, err := os.Stat("cover.jpg")
	if err != nil {
		// assume that if no cover was extracted and no error was thrown that no embedded cover exists
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

// Embed a given image as a cover.
func SetCover(file lib.Mediafile, cover string) error {
	if file.IsOpus {
		opustags := exec.Command("opustags", "--set-cover", cover, file.Path, "-i")
		err := opustags.Run()
		return err
	}

	tempfile := fmt.Sprintf("temp%s", filepath.Ext(file.Path))
	args := []string{"-i", file.Path, "-i", cover, "-map", "0", "-map", "1", "-c", "copy", "-metadata:s:v", `title="Album cover"`, "-metadata:s:v", `comment="Cover (front)"`, tempfile}
	ffmpeg := exec.Command("ffmpeg", args...)
	err := ffmpeg.Run()
	if err != nil {
		return err
	}

	err = os.Rename(tempfile, file.Path)
	return err
}
