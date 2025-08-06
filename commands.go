package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LoudnessInfo struct {
	I      string `json:"input_i"`
	TP     string `json:"input_tp"`
	LRA    string `json:"input_lra"`
	Thresh string `json:"input_thresh"`
	Offset string `json:"target_offset"`
}

type Audiofile struct {
	Path   string
	IsOpus bool
}

/*
 * Commands used during loudness normalization
 */

// Extract the sample rate of an audio file.
func extractSampleRate(file string) (string, error) {
	args := []string{"ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=sample_rate", "-of", "default=noprint_wrappers=1:nokey=1", file}
	ffmpeg := exec.Command(args[0], args[1:]...)
	output, err := ffmpeg.Output()
	if err != nil {
		return "", err
	}
	return strings.Trim(string(output), "\n"), nil
}

// Extract the bitrate of an audio file.
func extractBitrate(file Audiofile) (string, error) {
	if file.IsOpus {
		// ignore the error here as one is thrown even if the bitrate is extracted successfully
		opusinfo := exec.Command("opusinfo", file.Path)
		output, _ := opusinfo.Output()
		lines := strings.SplitSeq(string(output), "\n")
		for line := range lines {
			if !strings.HasSuffix(line, "kbit/s") {
				continue
			}

			return fmt.Sprintf("%sk", strings.Split(strings.TrimSpace(line), " ")[6]), nil
		}
		return "", fmt.Errorf("No information about the files bitrate could be found.")
	}

	args := []string{"ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries", "format=bit_rate", "-of", "default=noprint_wrappers=1:nokey=1", file.Path}
	ffmpeg := exec.Command(args[0], args[1:]...)
	output, err := ffmpeg.Output()
	if err != nil {
		return "", err
	}
	return strings.Trim(string(output), "\n"), nil
}

// First pass with ffmpeg to analyze the loudness of an audio file.
func extractLoudnessInfo(file string) (LoudnessInfo, error) {
	ffmpegArgs := []string{"ffmpeg", "-i", file, "-af", "loudnorm=print_format=json", "-nostats", "-hide_banner", "-f", "null", "-"}
	ffmpeg := exec.Command(ffmpegArgs[0], ffmpegArgs[1:]...)
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
func normalizeLoudness(file Audiofile, outpath string, targetLoudness float64, loudnessInfo LoudnessInfo, sampleRate string, bitrate string) error {
	loudnorm := fmt.Sprintf("loudnorm=linear=true:I=%.2f:LRA=7.0:TP=-2.0:offset=%s:measured_I=%s:measured_TP=%s:measured_LRA=%s:measured_thresh=%s", targetLoudness, loudnessInfo.Offset, loudnessInfo.I, loudnessInfo.TP, loudnessInfo.LRA, loudnessInfo.Thresh)
	args := []string{"ffmpeg", "-i", file.Path, "-af", loudnorm, "-ar", sampleRate, "-b:a", bitrate}
	if !file.IsOpus {
		args = append(args, []string{"-map", "0", "-map_metadata", "0"}...)
	} else {
		args = append(args, []string{"-map_metadata", "0"}...)
	}
	args = append(args, outpath)
	ffmpeg := exec.Command(args[0], args[1:]...)
	err := ffmpeg.Run()
	return err
}

/*
 * Commands used during various operations
 */

// Extract the embedded cover.
func extractCover(file Audiofile) (bool, error) {
	if file.IsOpus {
		args := []string{"opustags", "--output-cover", "cover.jpg", file.Path, "-i"}
		opustags := exec.Command(args[0], args[1:]...)
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
func setCover(file Audiofile, cover string) error {
	if file.IsOpus {
		args := []string{"opustags", "--set-cover", cover, file.Path, "-i"}
		opustags := exec.Command(args[0], args[1:]...)
		err := opustags.Run()
		return err
	}

	ext := filepath.Ext(file.Path)
	tempfile := fmt.Sprintf("temp%s", ext)
	args := []string{"ffmpeg", "-i", file.Path, "-i", cover, "-map", "0", "-map 1", "-c", "copy", "-metadata:s:v", "title=\"Album cover\"", "-metadata:s:v", "comment=\"Cover (front)\"", tempfile}
	ffmpeg := exec.Command(args[0], args[1:]...)
	err := ffmpeg.Run()
	if err != nil {
		return err
	}

	err = os.Rename(tempfile, file.Path)
	return err
}
