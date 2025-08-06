package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
)

func main() {
	normalizeCmd := flag.NewFlagSet("normalize", flag.ExitOnError)
	normalizeTargetLoudness := normalizeCmd.Float64("lufs", -18.0, "Target loudness in LUFS")
	normalizeOutput := normalizeCmd.String("output", "", "Output file or directory")

	if len(os.Args) < 2 {
		writer := tabwriter.NewWriter(os.Stdout, 15, 2, 1, ' ', 0)
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
			fmt.Println("Failed to parse flags: ", err)
			os.Exit(1)
		}
		if normalizeCmd.Arg(0) == "" {
			fmt.Println("Fatal: You need to provide an input directory or file.")
			fmt.Println("Usage: audio-workbench normalize [<args>] <path>")
			normalizeCmd.PrintDefaults()
			os.Exit(1)
		}
		normalize(*normalizeTargetLoudness, normalizeCmd.Arg(0), *normalizeOutput)
	case "set-cover":
		if len(os.Args) < 4 {
			fmt.Println("Fatal: You need to provide a cover file and a file or directory of files to apply it to.")
			fmt.Println("Usage: audio-workbench set-cover <cover> <path>")
			os.Exit(1)
		}
		imgExtensions := []string{".jpeg", ".jpg", ".png"}
		if !slices.Contains(imgExtensions, filepath.Ext(os.Args[2])) {
			fmt.Println("Fatal: Provided cover is not a supported image format.")
			fmt.Println("Supported formats: ", strings.Join(imgExtensions, ", "))
			os.Exit(1)
		}
		file := Audiofile{
			Path:   os.Args[3],
			IsOpus: strings.HasSuffix(os.Args[3], ".opus") || strings.HasSuffix(os.Args[3], ".ogg"),
		}
		err := setCover(file, os.Args[2])
		if err != nil {
			fmt.Printf("Failed to set %s as cover for %s: %s\n", os.Args[2], os.Args[3], err)
			os.Exit(1)
		}
	case "extract-cover":
		if len(os.Args) < 3 {
			fmt.Println("Fatal: You need to provide a file from which to extract a cover.")
			fmt.Println("Usage: audio-workbench extract-cover <path>")
			os.Exit(1)
		}
		file := Audiofile{
			Path:   os.Args[2],
			IsOpus: strings.HasSuffix(os.Args[2], ".opus") || strings.HasSuffix(os.Args[2], ".ogg"),
		}
		hasCover, err := extractCover(file)
		if err != nil {
			fmt.Printf("Failed to extract the cover from %s: %s", os.Args[2], err)
			os.Exit(1)
		}
		if hasCover {
			fmt.Println("Extracted the cover to cover.jpg.")
		} else {
			fmt.Println("No cover could be extracted from ", os.Args[2])
		}
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func normalize(targetLoudness float64, input string, outputDir string) {
	if outputDir != "" {
		err := os.MkdirAll(outputDir, 0775)
		if err != nil {
			fmt.Println("Failed to create output directory: ", err)
			os.Exit(1)
		}
	}

	inputInfo, err := os.Stat(input)
	if err != nil {
		fmt.Println("Failed to get info in input: ", err)
		os.Exit(1)
	}

	var files []Audiofile
	if inputInfo.IsDir() {
		entries, err := os.ReadDir(input)
		if err != nil {
			fmt.Println("Failed to read input directory: ", err)
			os.Exit(1)
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

	for _, file := range files {
		fmt.Println("Processing ", file.Path)
		sampleRate, err := extractSampleRate(file.Path)
		if err != nil {
			fmt.Printf("Failed to extract the sample rate from %s: %s\n", file.Path, err)
			os.Exit(1)
		}
		bitrate, err := extractBitrate(file)
		if err != nil {
			fmt.Printf("Failed to extract the bitrate from %s: %s", file.Path, err)
			os.Exit(1)
		}
		loudnessInfo, err := extractLoudnessInfo(file.Path)
		if err != nil {
			fmt.Printf("Failed to extract the loudness from %s: %s", file.Path, err)
			os.Exit(1)
		}

		// if we are normalizing a .opus or .ogg file we need to handle the cover image seperately
		var hasCover bool
		if file.IsOpus {
			hasCover, err = extractCover(file)
			if err != nil {
				fmt.Printf("Failed to extract the cover from %s: %s\n", file.Path, err)
				os.Exit(1)
			}
		}

		// build the output path for the normalized file
		var outpath string
		if outputDir == "" {
			ext := filepath.Ext(file.Path)
			outpath = fmt.Sprintf("temp%s", ext)
		} else {
			outpath = filepath.Join(outputDir, filepath.Base(file.Path))
		}

		err = normalizeLoudness(file, outpath, targetLoudness, loudnessInfo, sampleRate, bitrate)
		if err != nil {
			fmt.Printf("Failed to normalize the loudness of %s: %s\n", file.Path, err)
			os.Exit(1)
		}

		if outputDir == "" {
			err := os.Rename(outpath, file.Path)
			if err != nil {
				fmt.Printf("Failed to overwrite the original file for %s: %s\n", file.Path, err)
			}
		} else {
			// update the path to the new file for all remaining operations
			file.Path = outpath
		}

		// reapply the cover image to the normalized file for .opus or .ogg files
		if file.IsOpus && hasCover {
			err := setCover(file, "cover.jpg")
			if err != nil {
				fmt.Printf("Failed to set cover for %s: %s\n", file.Path, err)
				os.Exit(1)
			}
			err = os.Remove("cover.jpg")
			if err != nil {
				fmt.Println("Unable to remove temporary cover.jpg file: ", err)
				os.Exit(1)
			}
		}
	}
}
