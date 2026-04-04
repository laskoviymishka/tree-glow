package main

import (
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// renderImage loads an image file and renders it as colored Unicode half-blocks.
// Each terminal row represents 2 pixel rows using '▀' with fg=top, bg=bottom.
func renderImage(path string, width, height int) string {
	f, err := os.Open(path)
	if err != nil {
		return dimStyle.Render("  Cannot open image: " + err.Error())
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return dimStyle.Render("  Cannot decode image: " + err.Error())
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	// Draw onto opaque dark background to handle GIF/PNG transparency
	opaque := image.NewRGBA(bounds)
	draw.Draw(opaque, bounds, image.NewUniform(image.Black), image.Point{}, draw.Src)
	draw.Draw(opaque, bounds, img, bounds.Min, draw.Over)
	img = opaque

	// Fit to preview panel. Each terminal row = 2 pixel rows.
	pixelH := height * 2
	pixelW := width

	// Maintain aspect ratio
	scaleW := float64(pixelW) / float64(imgW)
	scaleH := float64(pixelH) / float64(imgH)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}

	newW := int(float64(imgW) * scale)
	newH := int(float64(imgH) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	// Resize using high-quality interpolation
	resized := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	// Center horizontally
	padLeft := (width - newW) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padding := strings.Repeat(" ", padLeft)

	// Header
	var out strings.Builder
	header := fmt.Sprintf("  %s  %dx%d", format, imgW, imgH)
	out.WriteString(dimStyle.Render(header))
	out.WriteString("\n\n")

	// Render using half-block characters
	// Each row of output = 2 rows of pixels
	for y := 0; y < newH-1; y += 2 {
		out.WriteString(padding)
		for x := 0; x < newW; x++ {
			topR, topG, topB, _ := resized.At(x, y).RGBA()
			botR, botG, botB, _ := resized.At(x, y+1).RGBA()

			// Convert from 16-bit to 8-bit
			tr, tg, tb := topR>>8, topG>>8, topB>>8
			br, bg, bb := botR>>8, botG>>8, botB>>8

			// ▀ with fg=top color, bg=bottom color
			out.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
				tr, tg, tb, br, bg, bb))
		}
		out.WriteString("\n")
	}

	// Handle odd height — last row only has top pixel
	if newH%2 == 1 {
		out.WriteString(padding)
		for x := 0; x < newW; x++ {
			r, g, b, _ := resized.At(x, newH-1).RGBA()
			out.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm▀\x1b[0m", r>>8, g>>8, b>>8))
		}
		out.WriteString("\n")
	}

	return out.String()
}

// animatedGif holds pre-decoded and pre-rendered GIF frames.
type animatedGif struct {
	frames []string // pre-rendered terminal strings per frame
	delays []time.Duration
	frame  int // current frame index
	width  int
	height int
	imgW   int
	imgH   int
	nFrames int
}

// loadAnimatedGif decodes all frames of a GIF and pre-renders them.
func loadAnimatedGif(path string, width, height int) *animatedGif {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil || len(g.Image) == 0 {
		return nil
	}

	ag := &animatedGif{
		width:   width,
		height:  height,
		imgW:    g.Config.Width,
		imgH:    g.Config.Height,
		nFrames: len(g.Image),
	}

	// Build a canvas and composite each frame (GIF disposal handling)
	canvas := image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
	// Fill with black background
	draw.Draw(canvas, canvas.Bounds(), image.Black, image.Point{}, draw.Src)

	for i, frame := range g.Image {
		// Draw frame onto canvas
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		// Render this composited state
		rendered := renderImageFromRGBA(canvas, width, height, "gif", g.Config.Width, g.Config.Height, i+1, len(g.Image))
		ag.frames = append(ag.frames, rendered)

		// Delay (GIF delay is in 100ths of a second)
		delay := 100 * time.Millisecond // default
		if i < len(g.Delay) && g.Delay[i] > 0 {
			delay = time.Duration(g.Delay[i]) * 10 * time.Millisecond
		}
		ag.delays = append(ag.delays, delay)
	}

	return ag
}

func (ag *animatedGif) CurrentFrame() string {
	if ag == nil || len(ag.frames) == 0 {
		return ""
	}
	return ag.frames[ag.frame]
}

func (ag *animatedGif) Advance() time.Duration {
	if ag == nil || len(ag.frames) <= 1 {
		return 0
	}
	delay := ag.delays[ag.frame]
	ag.frame = (ag.frame + 1) % len(ag.frames)
	return delay
}

// renderImageFromRGBA renders an already-decoded RGBA image to terminal half-blocks.
func renderImageFromRGBA(img *image.RGBA, width, height int, format string, imgW, imgH, frameNum, totalFrames int) string {
	bounds := img.Bounds()

	pixelH := height * 2
	pixelW := width

	scaleW := float64(pixelW) / float64(bounds.Dx())
	scaleH := float64(pixelH) / float64(bounds.Dy())
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}

	newW := int(float64(bounds.Dx()) * scale)
	newH := int(float64(bounds.Dy()) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	resized := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	padLeft := (width - newW) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padding := strings.Repeat(" ", padLeft)

	var out strings.Builder
	header := fmt.Sprintf("  %s  %dx%d  frame %d/%d", format, imgW, imgH, frameNum, totalFrames)
	out.WriteString(dimStyle.Render(header))
	out.WriteString("\n\n")

	for y := 0; y < newH-1; y += 2 {
		out.WriteString(padding)
		for x := 0; x < newW; x++ {
			topR, topG, topB, _ := resized.At(x, y).RGBA()
			botR, botG, botB, _ := resized.At(x, y+1).RGBA()
			tr, tg, tb := topR>>8, topG>>8, topB>>8
			br, bg, bb := botR>>8, botG>>8, botB>>8
			out.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
				tr, tg, tb, br, bg, bb))
		}
		out.WriteString("\n")
	}

	if newH%2 == 1 {
		out.WriteString(padding)
		for x := 0; x < newW; x++ {
			r, g, b, _ := resized.At(x, newH-1).RGBA()
			out.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm▀\x1b[0m", r>>8, g>>8, b>>8))
		}
		out.WriteString("\n")
	}

	return out.String()
}

// isImageFile checks if a file extension is a supported image format.
func isImageFile(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	}
	return false
}

func isGifFile(ext string) bool {
	return ext == ".gif"
}
