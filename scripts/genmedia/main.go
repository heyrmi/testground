// Command genmedia writes the fixture media the playground embeds.
//
// The media challenges need real files -- a browser will not decode a
// placeholder -- but binary assets checked in by hand rot silently: nobody can
// tell whether the committed PNG is still the one the challenge describes. So
// the assets are generated from this program instead, and regenerating them
// must produce byte-identical output. Every pixel here is a pure function of
// its coordinates, with no clock, no randomness and no encoder defaults left to
// chance, which is what lets `make media` act as a check rather than a change.
//
// Run it with `make media` from the repository root. ffmpeg is required for the
// two video clips and is deliberately not a build dependency: the outputs are
// committed, so only someone changing them needs it installed.
//
//go:generate go run .
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

// outDir is under web/static, which embed.go already embeds wholesale, so a new
// file here reaches the binary without touching the embed directives.
const outDir = "web/static/media"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genmedia:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// The three widths are one image at three sizes, so a srcset challenge can
	// assert which one the browser actually chose from its rendered width.
	for _, width := range []int{320, 640, 1280} {
		if err := writePNG(fmt.Sprintf("photo-%d.png", width), photo(width, width*3/4)); err != nil {
			return err
		}
	}
	if err := writeJPEG("photo.jpg", photo(640, 480)); err != nil {
		return err
	}
	if err := writePNG("poster.png", poster(640, 360)); err != nil {
		return err
	}
	if err := writePNG("tall.png", photo(400, 900)); err != nil {
		return err
	}

	if err := writeGIF("spinner.gif", spinner()); err != nil {
		return err
	}
	if err := writeSVG("sprite.svg", sprite()); err != nil {
		return err
	}
	if err := writeSVG("decorative.svg", decorative()); err != nil {
		return err
	}
	if err := writeBrokenPNG("broken.png"); err != nil {
		return err
	}
	if err := writeWAV("tone.wav"); err != nil {
		return err
	}
	return writeClips()
}

// photo is the ordinary image: a smooth gradient crossed by hard edges, which
// gives a screenshot comparison something to fail on and a scaler something to
// resample. It carries no text, so it survives translation challenges.
func photo(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			fx := float64(x) / float64(w)
			fy := float64(y) / float64(h)
			// The band term is what stops a resize from being invisible: it
			// puts a hard edge where interpolation has to make a decision.
			band := 0.0
			if (x/max(w/16, 1)+y/max(h/16, 1))%2 == 0 {
				band = 0.18
			}
			img.Set(x, y, color.RGBA{
				R: channel(0.15 + 0.55*fx + band),
				G: channel(0.25 + 0.45*fy),
				B: channel(0.70 - 0.35*fx - band),
				A: 0xff,
			})
		}
	}
	return img
}

// poster is flat and high-contrast so a test can tell the poster frame from the
// first decoded video frame, which the gradient would make ambiguous.
func poster(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := color.RGBA{R: 0x11, G: 0x1b, B: 0x2e, A: 0xff}
			if (x+y)%64 < 6 {
				c = color.RGBA{R: 0xf2, G: 0xc0, B: 0x4c, A: 0xff}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// spinner loops forever by design: a CSS-free animation that never settles is
// the visual-regression hazard the R category needs, and an animated GIF is the
// one form of it that no reduced-motion preference will stop.
func spinner() *gif.GIF {
	const (
		size   = 64
		frames = 12
	)
	palette := color.Palette{
		color.RGBA{0, 0, 0, 0},
		color.RGBA{0x2b, 0x6c, 0xb0, 0xff},
		color.RGBA{0xcb, 0xd5, 0xe0, 0xff},
	}

	out := &gif.GIF{LoopCount: 0}
	for f := range frames {
		frame := image.NewPaletted(image.Rect(0, 0, size, size), palette)
		lead := float64(f) / frames * 2 * math.Pi
		for y := range size {
			for x := range size {
				dx, dy := float64(x-size/2), float64(y-size/2)
				r := math.Hypot(dx, dy)
				if r < 22 || r > 30 {
					continue
				}
				// Distance around the ring from the leading edge, so the trail
				// fades behind the head rather than blinking on and off.
				delta := math.Mod(math.Atan2(dy, dx)-lead+4*math.Pi, 2*math.Pi)
				if delta < math.Pi {
					frame.SetColorIndex(x, y, 1)
				} else {
					frame.SetColorIndex(x, y, 2)
				}
			}
		}
		out.Image = append(out.Image, frame)
		out.Delay = append(out.Delay, 8) // hundredths of a second
		out.Disposal = append(out.Disposal, gif.DisposalBackground)
	}
	return out
}

func sprite() string {
	// A sprite sheet is referenced by fragment, so the icon a test is looking
	// for is never an element of its own -- only a <use> pointing into this.
	return `<svg xmlns="http://www.w3.org/2000/svg" style="display:none">
  <symbol id="icon-check" viewBox="0 0 24 24">
    <path d="M20 6 9 17l-5-5" fill="none" stroke="currentColor" stroke-width="2"/>
  </symbol>
  <symbol id="icon-cross" viewBox="0 0 24 24">
    <path d="M6 6l12 12M18 6L6 18" fill="none" stroke="currentColor" stroke-width="2"/>
  </symbol>
  <symbol id="icon-warning" viewBox="0 0 24 24">
    <path d="M12 3 2 20h20L12 3zm0 6v6m0 3v.5" fill="none" stroke="currentColor" stroke-width="2"/>
  </symbol>
</svg>
`
}

func decorative() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="24" viewBox="0 0 120 24">
  <rect width="120" height="24" fill="none"/>
  <path d="M0 12h120" stroke="#cbd5e0" stroke-width="2"/>
  <circle cx="60" cy="12" r="6" fill="#2b6cb0"/>
</svg>
`
}

// writeBrokenPNG emits a file that announces itself as a PNG and then is not
// one. A file that is merely absent produces a 404 and a different failure path
// in the browser; the challenge needs a resource served with a 200 that fails
// to decode, which is what fires the element's error event.
//
// Truncating a real PNG is the obvious approach and the wrong one: browsers
// decode progressively, so a half file paints half an image and may still
// report success. Keeping the signature but corrupting the header chunk fails
// at the structural parse, before any decoder has pixels to be lenient about.
func writeBrokenPNG(name string) error {
	broken := append([]byte("\x89PNG\r\n\x1a\n"), []byte("not an image, and deliberately so")...)
	return write(name, broken)
}

// writeWAV emits 16-bit mono PCM by hand. Go has no WAV encoder in the standard
// library and the format is small enough that adding a dependency for it would
// spend one of the ten the PRD allows.
func writeWAV(name string) error {
	const (
		rate    = 8000
		seconds = 2
		freq    = 440.0
	)
	samples := rate * seconds

	var body bytes.Buffer
	for i := range samples {
		// The envelope ramps down so the clip ends in silence rather than in a
		// click, which would show up as a spike in any waveform assertion.
		envelope := 1 - float64(i)/float64(samples)
		v := math.Sin(2*math.Pi*freq*float64(i)/rate) * envelope * 0.6
		binary.Write(&body, binary.LittleEndian, int16(v*math.MaxInt16))
	}

	var out bytes.Buffer
	out.WriteString("RIFF")
	binary.Write(&out, binary.LittleEndian, uint32(36+body.Len()))
	out.WriteString("WAVEfmt ")
	binary.Write(&out, binary.LittleEndian, uint32(16))   // PCM header size
	binary.Write(&out, binary.LittleEndian, uint16(1))    // PCM, uncompressed
	binary.Write(&out, binary.LittleEndian, uint16(1))    // mono
	binary.Write(&out, binary.LittleEndian, uint32(rate)) // sample rate
	binary.Write(&out, binary.LittleEndian, uint32(rate*2))
	binary.Write(&out, binary.LittleEndian, uint16(2))  // block align
	binary.Write(&out, binary.LittleEndian, uint16(16)) // bits per sample
	out.WriteString("data")
	binary.Write(&out, binary.LittleEndian, uint32(body.Len()))
	out.Write(body.Bytes())

	return write(name, out.Bytes())
}

// writeClips renders the video fixtures through ffmpeg. Each second of the clip
// is a flat, distinct colour, so a test can assert that playback actually
// advanced by sampling a pixel -- currentTime alone moves even when decoding
// has stalled.
func writeClips() error {
	frames, err := os.MkdirTemp("", "genmedia-frames-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(frames)

	const (
		fps      = 10
		seconds  = 4
		w, h     = 320, 180
		filename = "frame-%03d.png"
	)
	shades := []color.RGBA{
		{0x1f, 0x3a, 0x5f, 0xff},
		{0x2f, 0x7a, 0x4f, 0xff},
		{0x8a, 0x5a, 0x1f, 0xff},
		{0x6b, 0x2f, 0x5f, 0xff},
	}

	for i := range fps * seconds {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		base := shades[(i/fps)%len(shades)]
		for y := range h {
			for x := range w {
				c := base
				// A bar that sweeps left to right within each second gives
				// sub-second progress something to show.
				if x < (i%fps+1)*w/fps && y > h-24 {
					c = color.RGBA{0xf5, 0xf5, 0xf5, 0xff}
				}
				img.Set(x, y, c)
			}
		}
		path := filepath.Join(frames, fmt.Sprintf(filename, i))
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	// -bitexact and the stripped metadata are what make a regenerated clip
	// byte-identical; without them ffmpeg stamps the encoder version and the
	// current time into the container and every run produces a new file.
	common := []string{
		"-y", "-nostdin", "-loglevel", "error",
		"-framerate", fmt.Sprint(fps),
		"-i", filepath.Join(frames, filename),
		"-pix_fmt", "yuv420p",
		"-map_metadata", "-1",
		"-fflags", "+bitexact", "-flags:v", "+bitexact",
	}
	clips := []struct {
		name string
		args []string
	}{
		{"clip.webm", []string{"-c:v", "libvpx-vp9", "-b:v", "0", "-crf", "40"}},
		{"clip.mp4", []string{"-c:v", "libx264", "-crf", "28", "-preset", "veryfast", "-movflags", "+faststart"}},
	}

	for _, clip := range clips {
		args := append(append([]string{}, common...), clip.args...)
		args = append(args, filepath.Join(outDir, clip.name))
		cmd := exec.Command("ffmpeg", args...)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("encoding %s: %w", clip.name, err)
		}
		fmt.Println("wrote", filepath.Join(outDir, clip.name))
	}
	return nil
}

func writePNG(name string, img image.Image) error {
	var buf bytes.Buffer
	// The default compression level is part of the output bytes, so it is
	// pinned rather than inherited.
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return err
	}
	return write(name, buf.Bytes())
}

func writeJPEG(name string, img image.Image) error {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		return err
	}
	return write(name, buf.Bytes())
}

func writeGIF(name string, g *gif.GIF) error {
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return err
	}
	return write(name, buf.Bytes())
}

func writeSVG(name, body string) error { return write(name, []byte(body)) }

func write(name string, body []byte) error {
	path := filepath.Join(outDir, name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	return nil
}

func channel(v float64) uint8 {
	return uint8(math.Round(math.Min(math.Max(v, 0), 1) * 255))
}
