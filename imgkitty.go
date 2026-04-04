package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blacktop/go-termimg"
)

// kittyImageState holds pre-rendered kitty image data.
type kittyImageState struct {
	path     string
	format   string
	imgW     int
	imgH     int
	rendered string // raw kitty escape sequence output
	renderW  int    // rendered at this width
	renderH  int    // rendered at this height
}

// newKittyImage loads and renders an image via kitty protocol.
func newKittyImage(path string, width, height int) (*kittyImageState, error) {
	img, err := termimg.Open(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	format := ext[1:]

	// Compute height from width preserving aspect ratio.
	// Terminal cells are ~2:1 (height:width in pixels), so divide by 2.
	imgW := img.Bounds.Dx()
	imgH := img.Bounds.Dy()
	fitH := int(float64(width) * float64(imgH) / float64(imgW) / 2.0)
	if fitH < 1 {
		fitH = 1
	}
	if fitH > height {
		fitH = height
	}

	rendered, err := img.
		Width(width).
		Height(fitH).
		Protocol(termimg.Kitty).
		Render()
	if err != nil {
		return nil, err
	}

	return &kittyImageState{
		path:     path,
		format:   format,
		imgW:     img.Bounds.Dx(),
		imgH:     img.Bounds.Dy(),
		rendered: rendered,
		renderW:  width,
		renderH:  fitH,
	}, nil
}

// OverlayString returns escape sequences that:
// 1. Save cursor
// 2. Move to absolute position (row, col)
// 3. Clear kitty images in that area
// 4. Write the image
// 5. Restore cursor
// This string should be appended AFTER hardClip so it's never processed by lipgloss/ansi.Truncate.
func (k *kittyImageState) OverlayString(row, col int) string {
	if k.rendered == "" {
		return ""
	}
	var out strings.Builder
	// Save cursor
	out.WriteString("\033[s")
	// Move to target position
	out.WriteString(fmt.Sprintf("\033[%d;%dH", row, col))
	// Write image
	out.WriteString(k.rendered)
	// Restore cursor
	out.WriteString("\033[u")
	return out.String()
}

// Header returns the text header line for the preview.
func (k *kittyImageState) Header() string {
	return fmt.Sprintf("  %s  %dx%d  (kitty)", k.format, k.imgW, k.imgH)
}

var useKitty = detectKitty()

func detectKitty() bool {
	proto := termimg.DetectProtocol()
	return proto == termimg.Kitty
}

// isKittyAvailable checks if kitty graphics protocol is available.
func isKittyAvailable() bool {
	return useKitty
}

// kittyClearImages returns the escape sequence to delete all kitty images.
func kittyClearImages() string {
	return termimg.ClearAllString()
}
