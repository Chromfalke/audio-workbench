package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"github.com/Chromfalke/audio-workbench/internal/lib"
	"github.com/Chromfalke/audio-workbench/internal/processors"
)

func main() {
	normalizeCmd := pflag.NewFlagSet("normalize", pflag.ExitOnError)
	normalizeCmd.SetOutput(os.Stderr)
	targetLoudness := normalizeCmd.Float64P("lufs", "l", -18.0, "Target loudness in LUFS")

	convertCmd := pflag.NewFlagSet("convert", pflag.ExitOnError)
	convertCmd.SetOutput(os.Stderr)
	conversionFormat := convertCmd.StringP("format", "f", "mp3", "Output format")

	resampleCmd := pflag.NewFlagSet("resample", pflag.ExitOnError)
	resampleCmd.SetOutput(os.Stderr)
	resampleRate := resampleCmd.IntP("samplerate", "r", 48000, "Target sample rate")

	imgExtractCmd := pflag.NewFlagSet("extract-cover", pflag.ExitOnError)
	imgExtractCmd.SetOutput(os.Stderr)
	imgFormat := imgExtractCmd.StringP("format", "f", "jpg", "Output format")

	audioExtractCmd := pflag.NewFlagSet("extract-audio", pflag.ExitOnError)
	audioExtractCmd.SetOutput(os.Stderr)
	audioFormat := audioExtractCmd.StringP("format", "f", "mp3", "Output format")
	audioExtractCopyCover := audioExtractCmd.BoolP("copy-cover", "c", false, "Copy the cover from the video")
	audioExtractCoverTimestamp := audioExtractCmd.StringP("cover-timestamp", "t", "00:00:10", "The timestamp in the video to extract the cover from")

	if len(os.Args) < 2 || os.Args[1] == "help" {
		writer := tabwriter.NewWriter(os.Stderr, 15, 2, 1, ' ', 0)
		fmt.Fprintln(writer, "Usage: audio-workbench <command> [<args>]")
		fmt.Fprintln(writer, "Commands:")
		fmt.Fprintln(writer, "  normalize\tNormalize the loudness of an audio file")
		fmt.Fprintln(writer, "  convert\tConvert from one audio codec to another")
		fmt.Fprintln(writer, "  resample\tResample the audio to a different sample rate")
		fmt.Fprintln(writer, "  set-cover\tSet the cover image for an audio file")
		fmt.Fprintln(writer, "  extract-cover\tExtract the cover image from a media file")
		fmt.Fprintln(writer, "  extract-audio\tExtract the audio from a video")
		fmt.Fprintln(writer, "  help\tPrints this help message")
		writer.Flush()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "normalize":
		err := normalizeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalln("Failed to parse flags: ", err)
		}
		if normalizeCmd.Arg(0) == "" {
			log.Println("Usage: audio-workbench normalize [<args>] <path> [<outpath>]")
			normalizeCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		runner(normalizeCmd.Arg(0), normalizeCmd.Arg(1), processors.Normalizer{TargetLoudness: *targetLoudness})
	case "convert":
		err := convertCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalln("Failed to parse flags: ", err)
		}
		if convertCmd.Arg(0) == "" {
			log.Println("Usage: audio-workbench convert [<args>] <path> [<outpath>]")
			convertCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		validFormats := []string{"flac", "mp3", "opus", "wav"}
		if !slices.Contains(validFormats, *conversionFormat) {
			log.Println("Supported formats are: ", strings.Join(validFormats, ", "))
			log.Fatalf("Fatal: Invalid format %s\n", *conversionFormat)
		}

		runner(convertCmd.Arg(0), convertCmd.Arg(1), processors.Converter{Format: *conversionFormat})
	case "resample":
		err := resampleCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalln("Failed to parse flags: ", err)
		}
		if resampleCmd.Arg(0) == "" {
			log.Println("Usage: audio-workbench resample [<args>] <path>")
			resampleCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		if *resampleRate > 192000 || *resampleRate < 8000 {
			log.Fatalf("Unable to resample to %d Hz.\n", *resampleRate)
		}
		validOpusRates := []int{48000, 24000, 16000, 12000, 8000}
		validOpusStrings := []string{"48000", "24000", "16000", "12000", "8000"}
		if strings.HasSuffix(resampleCmd.Arg(0), ".opus") || strings.HasSuffix(resampleCmd.Arg(0), ".ogg") {
			if !slices.Contains(validOpusRates, *resampleRate) {
				log.Println("Supported sample rates for opus are: ", strings.Join(validOpusStrings, ", "))
				log.Fatalf("Fatal: Invalid sample rate for opus %d\n", *resampleRate)
			}
		}

		runner(resampleCmd.Arg(0), resampleCmd.Arg(1), processors.Resampler{SampleRate: *resampleRate})
	case "set-cover":
		if len(os.Args) < 4 {
			log.Println("Usage: audio-workbench set-cover <cover> <path>")
			log.Fatalln("Fatal: You need to provide a cover file and a file or directory of files to apply it to.")
		}
		imgExtensions := []string{".jpeg", ".jpg", ".png"}
		if !slices.Contains(imgExtensions, filepath.Ext(os.Args[2])) {
			log.Println("Supported image types: ", strings.Join(imgExtensions, ", "))
			log.Fatalf("Fatal: Provided cover format %s is not a supported image format.\n", filepath.Ext(os.Args[2]))
		}

		runner(os.Args[3], "", processors.CoverImageSetter{CoverImage: os.Args[2]})
	case "extract-cover":
		err := imgExtractCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalln("Failed to parse flags: ", err)
		}
		if imgExtractCmd.Arg(0) == "" {
			log.Println("Usage: audio-workbench extract-cover [<args>] <path> <outpath>")
			imgExtractCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		imgExtensions := []string{"jpeg", "jpg", "png"}
		var usedFormat string
		if filepath.Ext(imgExtractCmd.Arg(1)) != "" {
			usedFormat = filepath.Ext(imgExtractCmd.Arg(1))
			usedFormat = strings.ReplaceAll(usedFormat, ".", "")
		} else {
			usedFormat = *imgFormat
		}
		if !slices.Contains(imgExtensions, usedFormat) {
			log.Println(usedFormat)
			log.Println(*imgFormat)
			log.Println("Supported formats: ", strings.Join(imgExtensions, ", "))
			log.Fatalf("Fatal: Extracting a cover with format %s is not a supported.\n", usedFormat)
		}

		runner(imgExtractCmd.Arg(0), imgExtractCmd.Arg(1), processors.CoverImageExtractor{ImageFormat: "." + usedFormat})
	case "extract-audio":
		err := audioExtractCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalln("Failed to parse flags: ", err)
		}
		if audioExtractCmd.Arg(0) == "" {
			log.Println("Usage: audio-workbench extract-audio [<args>] <path>")
			audioExtractCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		validFormats := []string{"flac", "mp3", "opus", "wav"}
		if !slices.Contains(validFormats, *audioFormat) {
			log.Println("Supported formats are: ", strings.Join(validFormats, ", "))
			log.Fatalf("Fatal: Invalid format %s\n", *audioFormat)
		}

		if *audioExtractCopyCover {
			matches, err := regexp.MatchString("([0-5][0-9]|60):([0-5][0-9]|60):([0-5][0-9]|60)", *audioExtractCoverTimestamp)
			if err != nil {
				log.Fatalln("Failed to check timestamp: ", err)
			}
			if !matches {
				log.Fatalf("Fatal: The provided timestamp %s does not follow the schema HH:MM:SS.\n", *audioExtractCoverTimestamp)
			}
		}

		runner(audioExtractCmd.Arg(0), audioExtractCmd.Arg(1), processors.AudioExtractor{AudioFormat: "." + *audioFormat, CopyCover: *audioExtractCopyCover, VideoTimestamp: *audioExtractCoverTimestamp})
	default:
		log.Fatalln("Unknown command:", os.Args[1])
	}
}

func runner(input string, outputDir string, processor processors.Processor) {
	err := lib.CreateOutputDir(outputDir)
	if err != nil {
		log.Fatalln("Failed to create output directory: ", err)
	}

	files, err := lib.CollectInputFiles(input)
	if err != nil {
		log.Fatalln("Failed to collect input files: ", err)
	}

	for _, file := range files {
		log.Println("Processing ", file.Path)
		outpath := lib.BuildOutputPath(file, outputDir)
		processor.Run(file, outpath)
	}
}
