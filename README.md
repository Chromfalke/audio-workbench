# Audio Workbench

A simple CLI interface for a few things I do often with audio files. I used to use Audacity/Tenacity but wanted something less bulky.

## Operations

- Normalize the loudness of an audio file.
- Convert an audio file from one format to another.
- Resample an audio file to a different sample rate

Each of the operations supports bulk processing by passing in a folder instead of individual files.

## Dependencies

This is built mainly on top of ffmpeg and ffprobe but also uses opustags for working with the cover images of .opus/.ogg files.
