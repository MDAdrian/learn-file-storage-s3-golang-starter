package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
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
