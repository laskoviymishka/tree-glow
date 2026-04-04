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

// renderImage renders a static image using Unicode half-blocks.
// Kitty protocol is handled separately via kittyImageState in model.go.
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

	// Composite onto opaque background for transparency
	opaque := image.NewRGBA(bounds)
	draw.Draw(opaque, bounds, image.NewUniform(image.Black), image.Point{}, draw.Src)
	draw.Draw(opaque, bounds, img, bounds.Min, draw.Over)

	header := fmt.Sprintf("  %s  %dx%d", format, bounds.Dx(), bounds.Dy())
	return dimStyle.Render(header) + "\n\n" + renderHalfBlocks(opaque, width, height)
}

// renderHalfBlocks scales an image and renders it as Unicode half-block characters.
// Each terminal row represents 2 pixel rows using '▀' with fg=top, bg=bottom.
func renderHalfBlocks(img image.Image, width, height int) string {
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
	for y := 0; y < newH-1; y += 2 {
		out.WriteString(padding)
		for x := 0; x < newW; x++ {
			topR, topG, topB, _ := resized.At(x, y).RGBA()
			botR, botG, botB, _ := resized.At(x, y+1).RGBA()
			out.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
				topR>>8, topG>>8, topB>>8, botR>>8, botG>>8, botB>>8))
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

// --- Animated GIF ---

type animatedGif struct {
	frames  []string
	delays  []time.Duration
	frame   int
	width   int
	height  int
	imgW    int
	imgH    int
	nFrames int
}

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

	canvas := image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
	draw.Draw(canvas, canvas.Bounds(), image.Black, image.Point{}, draw.Src)

	for i, frame := range g.Image {
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		// Snapshot the canvas for this frame
		snapshot := image.NewRGBA(canvas.Bounds())
		draw.Draw(snapshot, snapshot.Bounds(), canvas, canvas.Bounds().Min, draw.Src)

		header := fmt.Sprintf("  gif  %dx%d  frame %d/%d", g.Config.Width, g.Config.Height, i+1, len(g.Image))
		rendered := dimStyle.Render(header) + "\n\n" + renderHalfBlocks(snapshot, width, height)
		ag.frames = append(ag.frames, rendered)

		delay := 100 * time.Millisecond
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

// --- File type helpers ---

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
