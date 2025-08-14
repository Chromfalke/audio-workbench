package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/Chromfalke/audio-workbench/internal/lib"
	"github.com/Chromfalke/audio-workbench/internal/processors"
)

func main() {
	normalizeCmd := flag.NewFlagSet("normalize", flag.ExitOnError)
	normalizeCmd.SetOutput(os.Stderr)
	normalizeTargetLoudness := normalizeCmd.Float64("lufs", -18.0, "Target loudness in LUFS")
	normalizeOutput := normalizeCmd.String("output", "", "Output file or directory")

	convertCmd := flag.NewFlagSet("convert", flag.ExitOnError)
	convertCmd.SetOutput(os.Stderr)
	convertFormat := convertCmd.String("format", "mp3", "Output format")
	convertOutput := convertCmd.String("output", "", "Output file or directory")

	resampleCmd := flag.NewFlagSet("resample", flag.ExitOnError)
	resampleCmd.SetOutput(os.Stderr)
	resampleRate := resampleCmd.Int("samplerate", 48000, "Target sample rate")
	resampleOutput := resampleCmd.String("output", "", "Output file or directory")

	imgExtractCmd := flag.NewFlagSet("extract-cover", flag.ExitOnError)
	imgExtractCmd.SetOutput(os.Stderr)
	imgExtractFormat := imgExtractCmd.String("format", "jpg", "Output format")
	imgExtractOutput := imgExtractCmd.String("output", "", "Output directory")

	if len(os.Args) < 2 {
		writer := tabwriter.NewWriter(os.Stderr, 15, 2, 1, ' ', 0)
		fmt.Fprintln(writer, "Usage: audio-workbench <command> [<args>]")
		fmt.Fprintln(writer, "Commands:")
		fmt.Fprintln(writer, "  normalize\tNormalize the loudness of an audio file")
		fmt.Fprintln(writer, "  convert\tConvert from one audio codec to another")
		fmt.Fprintln(writer, "  resample\tResample the audio to a different sample rate")
		fmt.Fprintln(writer, "  set-cover\tSet the cover image for an audio file")
		fmt.Fprintln(writer, "  extract-cover\tExtract the cover image from an audio file")
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
			log.Println("Usage: audio-workbench normalize [<args>] <path>")
			normalizeCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		runner(normalizeCmd.Arg(0), *normalizeOutput, processors.Normalizer{TargetLoudness: *normalizeTargetLoudness})
	case "convert":
		err := convertCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalln("Failed to parse flags: ", err)
		}
		if convertCmd.Arg(0) == "" {
			log.Println("Usage: audio-workbench convert [<args>] <path>")
			convertCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		validFormats := []string{"flac", "mp3", "opus", "wav"}
		if !slices.Contains(validFormats, *convertFormat) {
			log.Println("Supported formats are: ", strings.Join(validFormats, ", "))
			log.Fatalf("Fatal: Invalid format %s\n", *convertFormat)
		}

		runner(convertCmd.Arg(0), *convertOutput, processors.Converter{Format: *convertFormat})
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

		runner(resampleCmd.Arg(0), *resampleOutput, processors.Resampler{SampleRate: *resampleRate})
	case "set-cover":
		if len(os.Args) < 4 {
			log.Println("Usage: audio-workbench set-cover <cover> <path>")
			log.Fatalln("Fatal: You need to provide a cover file and a file or directory of files to apply it to.")
		}
		imgExtensions := []string{".jpeg", ".jpg", ".png"}
		if !slices.Contains(imgExtensions, filepath.Ext(os.Args[2])) {
			log.Println("Supported formats: ", strings.Join(imgExtensions, ", "))
			log.Fatalf("Fatal: Provided cover format %s is not a supported image format.\n", filepath.Ext(os.Args[2]))
		}

		runner(os.Args[3], "", processors.CoverImageSetter{CoverImage: os.Args[2]})
	case "extract-cover":
		err := imgExtractCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalln("Failed to parse flags: ", err)
		}
		if imgExtractCmd.Arg(0) == "" {
			log.Println("Usage: audio-workbench extract-cover [<args>] <path>")
			imgExtractCmd.PrintDefaults()
			log.Fatalln("Fatal: You need to provide an input directory or file.")
		}

		imgExtensions := []string{".jpeg", ".jpg", ".png"}
		var usedFormat string
		if filepath.Ext(*imgExtractOutput) != "" {
			usedFormat = filepath.Ext(*imgExtractOutput)
		} else {
			usedFormat = *imgExtractFormat
		}
		if !slices.Contains(imgExtensions, usedFormat) {
			log.Println("Supported formats: ", strings.Join(imgExtensions, ", "))
			log.Fatalf("Fatal: Extracting a cover with format %s is not a supported.\n", usedFormat)
		}

		runner(imgExtractCmd.Arg(0), *imgExtractOutput, processors.CoverImageExtractor{ImageFormat: usedFormat})
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
