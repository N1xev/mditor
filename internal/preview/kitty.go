package preview

import (
	"image"
	"image/draw"
	"image/gif"
	"io"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// kittyPlaceholder is the Unicode Private Use Area glyph (U+10EEEE) that the
// kitty graphics protocol uses as a virtual placement anchor. Each cell of
// the placeholder grid carries this rune; kitty renders the image over it.
// Kept byte-for-byte in sync with charmbracelet/x/ansi/kitty.Placeholder.
const kittyPlaceholder = '\U0010EEEE'

// Diacritics from the kitty protocol spec (rowcolumn-diacritics.txt). Each
// rune maps to an offset in the placeholder grid. The first one is the
// "row" of the first cell; the second is the "column" of the first cell.
// Kept byte-for-byte in sync with charmbracelet/x/ansi/kitty.Diacritic.
var diacritics = []rune{
	'̅', '̍', '̎', '̐', '̒', '̽', '̾',
	'̿', '͆', '͊', '͋', '͌', '͐', '͑',
	'͒', '͗', '͘', '͙', '͚', 'ͣ', 'ͤ',
	'ͥ', 'ͦ', 'ͧ', 'ͨ', 'ͩ', 'ͪ', 'ͫ',
	'ͬ', 'ͭ', 'ͮ', 'ͯ', '҃', '҄', '҅',
	'҆', '҇', '֑', '֒', '֓', '֔', '֕',
	'֖', '֗', '֘', '֙', '֚', '֛', '֜',
	'֝', '֞', '֟', '֠', '֡', '֢', '֣',
	'֤', '֥', '֦', '֧', '֨', '֩', '֪',
	'֫', '֬', '֭', '֮', '֯', 'ְ', 'ֱ',
	'ֲ', 'ֳ', 'ִ', 'ֵ', 'ֶ', 'ַ', 'ָ',
	'ֹ', 'ֺ', 'ֻ', 'ּ', 'ֽ', 'ֿ', 'ׁ',
	'ׂ', 'ׄ', 'ׅ', 'ׇ', 'ؐ', 'ؑ', 'ؒ',
	'ؓ', 'ؔ', 'ؕ', 'ؖ', 'ؗ', 'ؘ', 'ؙ',
	'ؚ', 'ً', 'ٌ', 'ٍ', 'َ', 'ُ', 'ِ',
	'ّ', 'ْ', 'ٓ', 'ٔ', 'ٕ', 'ٖ', 'ٗ',
	'٘', 'ٙ', 'ٚ', 'ٛ', 'ٜ', 'ٝ', 'ٞ',
	'ٟ', 'ٰ', 'ۖ', 'ۗ', 'ۘ', 'ۙ', 'ۚ',
	'ۛ', 'ۜ', '۟', '۠', 'ۡ', 'ۢ', 'ۣ',
	'ۤ', 'ۧ', 'ۨ', '۪', '۫', '۬', 'ۭ',
	'܉', '܊', '܋', '܌', '܍', '܏', 'ܐ',
	'ܑ', 'ܒ', 'ܓ', 'ܔ', 'ܕ', 'ܖ', 'ܗ',
	'ܘ', 'ܙ', 'ܚ', 'ܛ', 'ܜ', 'ܝ', 'ܞ',
	'ܟ', 'ܠ', 'ܡ', 'ܢ', 'ܣ', 'ܤ', 'ܥ',
	'ܦ', 'ܧ', 'ܨ', 'ܩ', 'ܪ', 'ܫ', 'ܬ',
	'ܭ', 'ܮ', 'ܯ', 'ܰ', 'ܱ', 'ܲ', 'ܳ',
	'ܴ', 'ܵ', 'ܶ', 'ܷ', 'ܸ', 'ܹ', 'ܺ',
	'ܻ', 'ܼ', 'ܽ', 'ܾ', 'ܿ', '݀', '݁',
	'݂', '݃', '݄', '݅', '݆', '݇', '݈',
	'݉', '݊', 'ަ', 'ާ', 'ި', 'ީ', 'ު',
	'ޫ', 'ެ', 'ޭ', 'ޮ', 'ޯ', 'ް', 'ޱ',
	'޲', '޳', '޴', '޵', '޶', '޷', '޸',
	'޹', '޺', '޻', '޼', '޽', '޾', '޿',
	'߀', '߁', '߂', '߃', '߄', '߅', '߆',
	'߇', '߈', '߉', 'ߊ', 'ߋ', 'ߌ', 'ߍ',
	'ߎ', 'ߏ', 'ߐ', 'ߑ', 'ߒ', 'ߓ', 'ߔ',
	'ߕ', 'ߖ', 'ߗ', 'ߘ', 'ߙ', 'ߚ', 'ߛ',
	'ߜ', 'ߝ', 'ߞ', 'ߟ', 'ߠ', 'ߡ', 'ߢ',
	'ߣ', 'ߤ', 'ߥ', 'ߦ', 'ߧ', 'ߨ', 'ߩ',
	'ߪ', '߫', '߬', '߭', '߮', '߯', '߰',
	'߱', '߲', '߳', 'ߴ', 'ߵ', '߶', '߷',
	'߸', '߹', 'ߺ', '߻', '߼', '߽', '߾',
	'߿', 'ऀ', 'ँ', 'ं',
}

// kittySupported detects whether the host terminal speaks the kitty graphics
// protocol. Matches the heuristic rasterm uses.
func kittySupported() bool {
	tg := strings.ToLower(os.Getenv("TERM_GRAPHICS"))
	switch tg {
	case "kitty":
		return true
	case "none", "iterm", "sixel":
		return false
	}
	if os.Getenv("KITTY_PID") != "" && os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if term == "xterm-kitty" {
		return true
	}
	if strings.ToLower(os.Getenv("TERM_PROGRAM")) == "ghostty" {
		return true
	}
	return false
}

// scaleToPixelSize returns the image scaled to the given exact pixel
// dimensions. Used by both static and animated frames.
func scaleToPixelSize(src image.Image, wPx, hPx int) image.Image {
	if src.Bounds().Dx() == wPx && src.Bounds().Dy() == hPx {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, wPx, hPx))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// encodeKittyStatic writes a single-frame image using the kitty graphics
// protocol with virtual placement. The keys follow the canonical charmbracelet
// x/ansi/kitty.EncodeGraphics approach: f=32 (RGBA), U=1, C=1, plus c=cols,
// r=rows so the terminal knows the placeholder grid size. Without c/r the
// virtual placement has no anchor geometry and the placeholders are ignored.
//
// Per the kitty graphics protocol docs, each U+10EEEE placeholder occupies one
// cell, so c= must equal the number of placeholders emitted horizontally (cols)
// to make the image scale to fit the placeholder grid. If c differs from the
// emitted placeholder count, kitty only displays the matching portion and the
// rest of the image overflows or is clipped.
func encodeKittyStatic(w io.Writer, img image.Image, wPx, hPx int, id uint32, cols, rows int) (uint32, error) {
	if wPx < 1 || hPx < 1 {
		return 0, errInvalidSize
	}
	scaled := scaleToPixelSize(img, wPx, hPx)
	rgba := image.NewRGBA(image.Rect(0, 0, wPx, hPx))
	draw.Draw(rgba, rgba.Bounds(), scaled, scaled.Bounds().Min, draw.Src)
	raw := rgba.Pix

	keys := []string{
		"q=2",
		"a=T",
		"C=1",
		"U=1",
		"f=32",
		"s=" + itoa(wPx),
		"v=" + itoa(hPx),
		"c=" + itoa(cols),
		"r=" + itoa(rows),
		"i=" + itoa(int(id)),
	}
	if err := writeRawChunks(w, strings.Join(keys, ","), raw); err != nil {
		return 0, err
	}
	return id, nil
}

// encodeKittyAnimated writes an animated GIF using the kitty graphics
// animation mode. The first frame carries U=1 + c/r + animation marker;
// subsequent frames reference the same image id and only update pixels.
func encodeKittyAnimated(w io.Writer, frames []image.Image, delays []int, wPx, hPx int, id uint32, cols, rows int) (uint32, error) {
	if len(frames) == 0 {
		return 0, errNoFrames
	}
	if wPx < 1 || hPx < 1 {
		return 0, errInvalidSize
	}
	for i, f := range frames {
		scaled := scaleToPixelSize(f, wPx, hPx)
		rgba := image.NewRGBA(image.Rect(0, 0, wPx, hPx))
		draw.Draw(rgba, rgba.Bounds(), scaled, scaled.Bounds().Min, draw.Src)
		raw := rgba.Pix
		ms := delayToMs(delays, i)
		var keys string
		if i == 0 {
			keys = strings.Join([]string{
				"q=2",
				"a=T",
				"C=1",
				"U=1",
				"f=32",
				"s=" + itoa(wPx),
				"v=" + itoa(hPx),
				"c=" + itoa(cols),
				"r=" + itoa(rows),
				"z=" + itoa(ms),
				"i=" + itoa(int(id)),
			}, ",")
		} else {
			keys = strings.Join([]string{
				"q=2",
				"a=f",
				"z=" + itoa(ms),
				"i=" + itoa(int(id)),
			}, ",")
		}
		if err := writeRawChunks(w, keys, raw); err != nil {
			return 0, err
		}
	}
	return id, nil
}

var (
	errInvalidSize = stringError("invalid pixel size")
	errNoFrames    = stringError("no frames")
)

type stringError string

func (e stringError) Error() string { return string(e) }

// writeRawChunks emits a base64-encoded RGBA payload as one or more kitty
// graphics protocol chunks (canonical 4096-char base64 chunks). Wraps each
// chunk in DCS passthrough when running inside tmux.
func writeRawChunks(w io.Writer, keys string, raw []byte) error {
	encoded := base64Encode(raw)
	total := len(encoded)
	if total == 0 {
		return stringError("empty payload")
	}
	const chunkSize = 4096
	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		more := end < total
		var seq string
		if i == 0 {
			if more {
				seq = "\x1b_G" + keys + ",m=1;" + encoded[i:end] + "\x1b\\"
			} else {
				seq = "\x1b_G" + keys + ",m=0;" + encoded[i:end] + "\x1b\\"
			}
		} else {
			mFlag := "0"
			if more {
				mFlag = "1"
			}
			seq = "\x1b_Gm=" + mFlag + ";" + encoded[i:end] + "\x1b\\"
		}
		if err := writeRawOrPassthrough(w, seq); err != nil {
			return err
		}
	}
	return nil
}

// itoa converts a non-negative int to its decimal string representation.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

// base64Encode returns the standard base64 encoding of src.
func base64Encode(src []byte) string {
	const tbl = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	srcLen := len(src)
	outLen := ((srcLen + 2) / 3) * 4
	out := make([]byte, 0, outLen)
	var i int
	for ; i+3 <= srcLen; i += 3 {
		v := uint32(src[i])<<16 | uint32(src[i+1])<<8 | uint32(src[i+2])
		out = append(out, tbl[(v>>18)&0x3f], tbl[(v>>12)&0x3f], tbl[(v>>6)&0x3f], tbl[v&0x3f])
	}
	switch srcLen - i {
	case 1:
		v := uint32(src[i]) << 16
		out = append(out, tbl[(v>>18)&0x3f], tbl[(v>>12)&0x3f], '=', '=')
	case 2:
		v := uint32(src[i])<<16 | uint32(src[i+1])<<8
		out = append(out, tbl[(v>>18)&0x3f], tbl[(v>>12)&0x3f], tbl[(v>>6)&0x3f], '=')
	}
	return string(out)
}

// writeRawOrPassthrough emits the escape sequence directly when not in tmux
// and wraps it in a tmux DCS passthrough when running inside tmux.
func writeRawOrPassthrough(w io.Writer, seq string) error {
	if !inTmux {
		_, err := io.WriteString(w, seq)
		return err
	}
	var b strings.Builder
	b.Grow(len(seq) + 8)
	b.WriteString("\x1bPtmux;")
	for i := 0; i < len(seq); i++ {
		c := seq[i]
		b.WriteByte(c)
		if c == 0x1b {
			b.WriteByte(0x1b)
		}
	}
	b.WriteString("\x1b\\")
	_, err := io.WriteString(w, b.String())
	return err
}

func delayToMs(delays []int, idx int) int {
	if idx >= len(delays) {
		return 100
	}
	d := delays[idx]
	if d < 2 {
		d = 10
	}
	return d * 10
}

func compositeGIF(g *gif.GIF) []image.Image {
	if len(g.Image) == 0 {
		return nil
	}
	w, h := g.Config.Width, g.Config.Height
	frames := make([]image.Image, len(g.Image))
	prev := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, src := range g.Image {
		disposal := byte(0)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		frame := *prev
		frame.Pix = make([]uint8, len(prev.Pix))
		copy(frame.Pix, prev.Pix)
		draw.Draw(&frame, frame.Bounds(), src, src.Bounds().Min, draw.Over)
		frames[i] = &frame
		switch disposal {
		case 0, 1:
			prev = cloneRGBA(&frame)
		case 2, 3:
			prev = image.NewRGBA(image.Rect(0, 0, w, h))
		}
	}
	return frames
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}
