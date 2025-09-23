package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	// Run the command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe failed: %w; stderr: %s", err, errBuf.String())
	}

	// Define minimal structs matching the parts of ffprobe's JSON we need
	type stream struct {
		CodecType string `json:"codec_type"` // "video", "audio", etc.
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	type ffprobeOutput struct {
		Streams []stream `json:"streams"`
	}

	// Unmarshal from the byte's buffer
	var info ffprobeOutput
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		return "", fmt.Errorf("failed to parse ffprobe to JSON: %w", err)
	}

	// Find the first video stream with height and width
	var w, h int
	for _, s := range info.Streams {
		if s.CodecType == "video" && s.Width > 0 && s.Height > 0 {
			w, h = s.Width, s.Height
			break
		}
	}
	if w == 0 || h == 0 {
		return "", fmt.Errorf("no valid video stream found with width and height")
	}

	// Compute aspect ratio
	const (
		target169 = 16.0 / 9.0
		target916 = 9.0 / 16.0
		eps = 0.02 // 2% tolerance
	)

	r := float64(w) / float64(h)

	switch {
	case math.Abs(r - target169) < eps:
		return "16:9", nil
	case math.Abs(r - target916) < eps:
		return "9:16", nil
	default:
		return "other", nil
	}
	
}

// processVideoForFastStart takes a path to a local (temp) file and produces a new MP4
// with the "faststart" flag (moov atom moved to the front). It returns the new file path.
func processVideoForFastStart(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("empty input file path")
	}
	// Create output path (simple convention: append ".processing")
	outPath := filePath + ".processing"

	// ffmpeg -i <in> -c copy -movflags faststart -f mp4 <out>
	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outPath,
	)

	// If you want logs, you can wire these up:
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg faststart failed: %w", err)
	}
	// Basic sanity check that output exists and is non-zero
	info, err := os.Stat(outPath)
	if err != nil {
		return "", fmt.Errorf("processed file missing: %w", err)
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("processed file is empty")
	}

	return outPath, nil
}