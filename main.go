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
)

type Process interface {
	Run(file Audiofile, outpath string) error
}

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

		runner(normalizeCmd.Arg(0), *normalizeOutput, Normalizer{TargetLoudness: *normalizeTargetLoudness})
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

		validFormats := []string{"flac", "mp3", "ogg", "opus", "wav"}
		if !slices.Contains(validFormats, *convertFormat) {
			log.Println("Supported formats are: ", strings.Join(validFormats, ", "))
			log.Fatalf("Fatal: Invalid format %s\n", *convertFormat)
		}

		runner(convertCmd.Arg(0), *convertOutput, Converter{Format: *convertFormat})
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
				log.Fatalf("Fatal: Invalid sample rate %d\n", *resampleRate)
			}
		}

		runner(resampleCmd.Arg(0), *resampleOutput, Resampler{SampleRate: *resampleRate})
	case "set-cover":
		if len(os.Args) < 4 {
			log.Println("Usage: audio-workbench set-cover <cover> <path>")
			log.Fatalln("Fatal: You need to provide a cover file and a file or directory of files to apply it to.")
		}
		imgExtensions := []string{".jpeg", ".jpg", ".png"}
		if !slices.Contains(imgExtensions, filepath.Ext(os.Args[2])) {
			log.Println("Supported formats: ", strings.Join(imgExtensions, ", "))
			log.Fatalln("Fatal: Provided cover is not a supported image format.")
		}
		file := Audiofile{
			Path:   os.Args[3],
			IsOpus: strings.HasSuffix(os.Args[3], ".opus") || strings.HasSuffix(os.Args[3], ".ogg"),
		}
		err := setCover(file, os.Args[2])
		if err != nil {
			log.Fatalf("Failed to set %s as cover for %s: %s\n", os.Args[2], os.Args[3], err)
		}
	case "extract-cover":
		if len(os.Args) < 3 {
			fmt.Println("Usage: audio-workbench extract-cover <path>")
			log.Fatalln("Fatal: You need to provide a file from which to extract a cover.")
		}
		file := Audiofile{
			Path:   os.Args[2],
			IsOpus: strings.HasSuffix(os.Args[2], ".opus") || strings.HasSuffix(os.Args[2], ".ogg"),
		}
		hasCover, err := extractCover(file)
		if err != nil {
			log.Fatalf("Failed to extract the cover from %s: %s", os.Args[2], err)
		}
		if hasCover {
			fmt.Println("Extracted the cover to cover.jpg.")
		} else {
			fmt.Println("No cover could be extracted from ", os.Args[2])
		}
	default:
		log.Fatalln("Unknown command:", os.Args[1])
	}
}

func runner(input string, outputDir string, process Process) {
	err := createOutputDir(outputDir)
	if err != nil {
		log.Fatalln("Failed to create output directory: ", err)
	}

	files, err := collectInputFiles(input)
	if err != nil {
		log.Fatalln("Failed to collect input files: ", err)
	}

	for _, file := range files {
		fmt.Println("Processing ", file.Path)
		var hasCover bool
		if file.IsOpus {
			hasCover, err = extractCover(file)
			if err != nil {
				log.Fatalf("Failed to extract the cover from %s: %s\n", file.Path, err)
			}
		}

		outpath := buildOutputPath(file, outputDir)

		process.Run(file, outpath)

		if outputDir == "" {
			err := os.Rename(outpath, file.Path)
			if err != nil {
				log.Fatalf("Failed to overwrite the original file for %s: %s\n", file.Path, err)
			}
		} else {
			// update the path to the new file for all remaining operations
			file.Path = outpath
		}

		if file.IsOpus && hasCover {
			err := setCover(file, "cover.jpg")
			if err != nil {
				log.Fatalf("Failed to set cover for %s: %s\n", file.Path, err)
			}
			err = os.Remove("cover.jpg")
			if err != nil {
				log.Fatalln("Unable to remove temporary cover.jpg file: ", err)
			}
		}
	}
}

// Process to normalize the loudness of an audio file
type Normalizer struct {
	TargetLoudness float64
}

func (normalizer Normalizer) Run(file Audiofile, outpath string) error {
	sampleRate, err := extractSampleRate(file.Path)
	if err != nil {
		return fmt.Errorf("Failed to extract the sample rate from %s: %s\n", file.Path, err)
	}
	bitrate, err := extractBitrate(file)
	if err != nil {
		return fmt.Errorf("Failed to extract the bitrate from %s: %s", file.Path, err)
	}
	loudnessInfo, err := extractLoudnessInfo(file.Path)
	if err != nil {
		return fmt.Errorf("Failed to extract the loudness from %s: %s", file.Path, err)
	}

	err = normalizeLoudness(file, outpath, normalizer.TargetLoudness, loudnessInfo, sampleRate, bitrate)
	if err != nil {
		return fmt.Errorf("Failed to normalize the loudness of %s: %s\n", file.Path, err)
	}

	return nil
}

// Process to convert the audio file to a different format
type Converter struct {
	Format string
}

func (converter Converter) Run(file Audiofile, outpath string) error {
	ext := filepath.Ext(outpath)
	outpath = strings.TrimRight(outpath, ext) + "." + converter.Format

	sampleRate, err := extractSampleRate(file.Path)
	if err != nil {
		return fmt.Errorf("Failed to extract the sample rate from %s: %s\n", file.Path, err)
	}
	bitrate, err := extractBitrate(file)
	if err != nil {
		return fmt.Errorf("Failed to extract the bitrate from %s: %s", file.Path, err)
	}
	err = convert(file, outpath, sampleRate, bitrate)
	if err != nil {
		return fmt.Errorf("Failed to convert %s to %s: %s", file.Path, converter.Format, err)
	}

	return nil
}

// Process to resample an audio file
type Resampler struct {
	SampleRate int
}

func (resampler Resampler) Run(file Audiofile, outpath string) error {
	bitrate, err := extractBitrate(file)
	if err != nil {
		return fmt.Errorf("Failed to extract the bitrate from %s: %s", file.Path, err)
	}
	err = resample(file, outpath, resampler.SampleRate, bitrate)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Failed to resample the %s: %s", file.Path, err)
	}

	return nil
}
