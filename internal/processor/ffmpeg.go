package processor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

type Processor struct {
	ffmpegPath  string
	ffprobePath string
}

func New(ffmpegPath, ffprobePath string) *Processor {
	return &Processor{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}
}

// Segment takes an input video file and produces HLS segments in outputDir.
func (p *Processor) Segment(ctx context.Context, inputPath, outputDir string) error {
	outputFile := filepath.Join(outputDir, "index.m3u8")

	cmd := exec.CommandContext(ctx, p.ffmpegPath,
		"-i", inputPath,
		"-codec", "copy",
		"-start_number", "0",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(outputDir, "seg%03d.ts"),
		"-f", "hls",
		outputFile,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg segmentation failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// Probe returns the duration of the video file in seconds.
func (p *Processor) Probe(filepath string) (float64, error) {
	return ProbeFile(filepath, p.ffprobePath)
}
