# Audio Workbench

A simple CLI interface for a few things I do often with audio files. I used to use Audacity/Tenacity but wanted something simpler.

## Operations

- Normalize the loudness of an audio file.
- Convert an audio file from one format to another.
- Resample an audio file to a different sample rate

## Dependencies

This is built mainly on top of ffmpeg and ffprobe but also uses opusinfo and opustags for some operations on .opus/.ogg files.
