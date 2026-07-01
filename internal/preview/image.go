package preview

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"charm.land/lipgloss/v2"
	"github.com/N1xev/mditor/internal/uict"
	"github.com/charmbracelet/x/mosaic"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/draw"
)

const (
	// Img sentinel pieces use control characters as glue so glamour's
	// word-wrap cannot break them across lines. The structure of the
	// emitted sentinel is the same regardless of the idx, so the regex
	// can recover it deterministically.
	ImgOpenPrefix  = "\x01mdt-img-o"
	ImgClosePrefix = "\x01mdt-img-c"
)

func ImgSentinelOpen(idx int) string  { return fmt.Sprintf("%s%d\x02", ImgOpenPrefix, idx) }
func ImgSentinelClose(idx int) string { return fmt.Sprintf("%s%d\x02", ImgClosePrefix, idx) }

var imgSentinelRE = regexp.MustCompile(
	`(?s)` + regexp.QuoteMeta(ImgOpenPrefix) + `(\d+)\x02(.*?)` + regexp.QuoteMeta(ImgClosePrefix) + `\d+\x02`,
)

type ImageSpec struct {
	ID   int
	Path string
	Alt  string
}

func PreprocessImages(src, baseDir string) (string, []ImageSpec) {
	var specs []ImageSpec
	var b strings.Builder
	parts := strings.Split(src, "```")
	for i, part := range parts {
		if i%2 == 0 {
			b.WriteString(transformImagesOutsideFence(part, &specs, baseDir))
		} else {
			b.WriteString(part)
		}
		if i < len(parts)-1 {
			b.WriteString("```")
		}
	}
	return b.String(), specs
}

var imgInlineRE = regexp.MustCompile(`!\[((?:[^\[\]\\]|\\.)*)\]\(([^\s)<>]+)(?:\s+"[^"]*")?\)`)

var imgRefDeclRE = regexp.MustCompile(`(?m)^\s{0,3}\[([^\]]+)\]:\s+(\S+)(?:\s+"[^"]*")?\s*$`)

func transformImagesOutsideFence(src string, specs *[]ImageSpec, baseDir string) string {
	refDefs := make(map[string]string)
	src = imgRefDeclRE.ReplaceAllStringFunc(src, func(m string) string {
		sm := imgRefDeclRE.FindStringSubmatch(m)
		if len(sm) >= 3 {
			refDefs[sm[1]] = sm[2]
		}
		return m
	})

	imgRefRE := regexp.MustCompile(`!\[((?:[^\[\]\\]|\\.)*)\]\[([^\]]+)\]`)
	src = imgRefRE.ReplaceAllStringFunc(src, func(m string) string {
		sm := imgRefRE.FindStringSubmatch(m)
		alt := sm[1]
		refID := sm[2]
		path, ok := refDefs[refID]
		if !ok {
			return m
		}
		idx := len(*specs)
		*specs = append(*specs, ImageSpec{ID: idx, Path: resolvePath(path, baseDir), Alt: alt})
		return ImgSentinelOpen(idx) + alt + ImgSentinelClose(idx)
	})
	src = imgInlineRE.ReplaceAllStringFunc(src, func(m string) string {
		sm := imgInlineRE.FindStringSubmatch(m)
		alt := sm[1]
		path := sm[2]
		idx := len(*specs)
		*specs = append(*specs, ImageSpec{ID: idx, Path: resolvePath(path, baseDir), Alt: alt})
		return ImgSentinelOpen(idx) + alt + ImgSentinelClose(idx)
	})
	return src
}

func resolvePath(p, baseDir string) string {
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") || strings.HasPrefix(p, "data:") {
		return p
	}
	if baseDir == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(baseDir, p)
}

func LoadImage(spec ImageSpec) (img image.Image, alt string, err error) {
	if spec.Path == "" {
		return nil, spec.Alt, fmt.Errorf("empty path")
	}
	rc, cleanup, openErr := openSource(spec)
	if openErr != nil {
		return nil, spec.Alt, openErr
	}
	defer cleanup()
	return decodeAny(rc, spec.Alt)
}

const remoteTimeout = 3 * time.Second

// httpUserAgent identifies mditor to remote image hosts. Some CDNs (notably
// Wikimedia) reject requests without a User-Agent header with HTTP 403.
const httpUserAgent = "mditor/1.0 (+https://github.com/N1xev/mditor)"

func openSource(spec ImageSpec) (io.ReadCloser, func(), error) {
	if strings.HasPrefix(spec.Path, "http://") || strings.HasPrefix(spec.Path, "https://") {
		ctx, cancel := context.WithTimeout(context.Background(), remoteTimeout)
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, spec.Path, nil)
		if reqErr != nil {
			cancel()
			return nil, func() {}, reqErr
		}
		req.Header.Set("User-Agent", httpUserAgent)
		req.Header.Set("Accept", "image/png,image/jpeg,image/gif,image/webp,image/*;q=0.8")
		resp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			cancel()
			return nil, func() {}, doErr
		}
		if resp.StatusCode/100 != 2 {
			resp.Body.Close()
			cancel()
			return nil, func() {}, fmt.Errorf("http %d", resp.StatusCode)
		}
		return resp.Body, func() {
			resp.Body.Close()
			cancel()
		}, nil
	}
	f, err := os.Open(spec.Path)
	if err != nil {
		return nil, func() {}, err
	}
	return f, func() { f.Close() }, nil
}

// gifMagic is the first 6 bytes of every GIF file.
var gifMagic = []byte{'G', 'I', 'F', '8', '?', 'a'}

func isGIFHeader(b []byte) bool {
	if len(b) < 6 {
		return false
	}
	return b[0] == gifMagic[0] && b[1] == gifMagic[1] && b[2] == gifMagic[2] &&
		b[3] == gifMagic[3] && (b[4] == '7' || b[4] == '9') && b[5] == 'a'
}

// svgTrimmedPrefix returns the first up-to-256 bytes of buf with any leading
// whitespace/BOM stripped, so SVG detection works on documents that begin with
// a UTF-8 BOM or a comment block. The full byte slice is what oksvg consumes.
func isSVG(b []byte) bool {
	trimmed := bytes.TrimLeft(b, " \t\r\n\xef\xbb\xbf")
	if len(trimmed) < 5 {
		return false
	}
	if bytes.HasPrefix(trimmed, []byte("<svg")) || bytes.HasPrefix(trimmed, []byte("<?xml")) {
		return true
	}
	if bytes.HasPrefix(trimmed, []byte("<!--")) {
		end := bytes.Index(trimmed, []byte("-->"))
		if end < 0 {
			return false
		}
		rest := bytes.TrimLeft(trimmed[end+3:], " \t\r\n")
		return bytes.HasPrefix(rest, []byte("<svg"))
	}
	return false
}

// decodeSVG rasterizes an SVG buffer using oksvg+rasterx into an RGBA image at
// the SVG's natural viewBox dimensions. Use WarnErrorMode so missing attributes
// don't fail the entire decode for partial / hand-written SVGs.
func decodeSVG(b []byte) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(b), oksvg.WarnErrorMode)
	if err != nil {
		return nil, err
	}
	w := int(icon.ViewBox.W)
	h := int(icon.ViewBox.H)
	if w < 1 || h < 1 {
		w, h = 256, 256
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.SetTarget(0, 0, float64(w), float64(h))
	icon.Draw(dasher, 1.0)
	return img, nil
}

// loadedImage carries everything the renderer needs about a decoded source:
// a sequence of frames (one for static images, many for animated GIFs) and
// the per-frame delay in 100ths of a second.
type loadedImage struct {
	frames   []image.Image
	delays   []int
	animated bool
	alt      string
}

func (l loadedImage) pick() image.Image {
	if len(l.frames) == 0 {
		return nil
	}
	return l.frames[0]
}

// LoadImageFull returns the full frame set so animated GIFs can be rendered
// in motion. For non-GIF sources, it returns a single-frame loadedImage.
func LoadImageFull(spec ImageSpec) (loadedImage, error) {
	if spec.Path == "" {
		return loadedImage{alt: spec.Alt}, fmt.Errorf("empty path")
	}
	rc, cleanup, err := openSource(spec)
	if err != nil {
		return loadedImage{alt: spec.Alt}, err
	}
	defer cleanup()
	return decodeAnyFull(rc, spec.Alt)
}

func decodeAny(r io.Reader, alt string) (image.Image, string, error) {
	out, err := decodeAnyFull(r, alt)
	if err != nil {
		return nil, alt, err
	}
	return out.pick(), out.alt, nil
}

func decodeAnyFull(r io.Reader, alt string) (loadedImage, error) {
	buf, err := io.ReadAll(io.LimitReader(r, 50<<20))
	if err != nil {
		return loadedImage{alt: alt}, err
	}
	if isGIFHeader(buf) {
		g, decErr := gif.DecodeAll(bytes.NewReader(buf))
		if decErr != nil {
			return loadedImage{alt: alt}, decErr
		}
		frames := compositeGIF(g)
		if len(frames) == 0 {
			return loadedImage{alt: alt}, fmt.Errorf("empty gif")
		}
		return loadedImage{
			frames:   frames,
			delays:   g.Delay,
			animated: len(frames) > 1,
			alt:      alt,
		}, nil
	}
	if isSVG(buf) {
		img, decErr := decodeSVG(buf)
		if decErr != nil {
			return loadedImage{alt: alt}, decErr
		}
		return loadedImage{
			frames:   []image.Image{img},
			alt:      alt,
			animated: false,
		}, nil
	}
	img, _, decErr := image.Decode(bytes.NewReader(buf))
	if decErr != nil {
		return loadedImage{alt: alt}, decErr
	}
	return loadedImage{
		frames:   []image.Image{img},
		alt:      alt,
		animated: false,
	}, nil
}

// cellDims returns the (cols, rows) of placeholder glyphs for an image of
// natural pixel size (natW, natH) bounded by the given max cell counts,
// preserving aspect ratio. Each placeholder glyph U+10EEEE occupies one
// terminal cell per the kitty graphics protocol, so the returned w/h are
// literal cell counts that map 1:1 to ESC_G c= and r= values.
func cellDims(natW, natH, maxCellsW, maxCellsH int) (w, h int) {
	if natW <= 0 || natH <= 0 {
		return 0, 0
	}
	if maxCellsW <= 0 {
		maxCellsW = 80
	}
	if maxCellsH <= 0 {
		maxCellsH = 20
	}
	w = maxCellsW
	h = (natH * w) / natW
	if h > maxCellsH {
		h = maxCellsH
		w = (natW * h) / natH
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func scaleToCells(src image.Image, wPx, hPx int) image.Image {
	if wPx == src.Bounds().Dx() && hPx == src.Bounds().Dy() {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, wPx, hPx))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

type protocolKind int

const (
	protoNative protocolKind = iota
	protoMosaic
	protoAltText
)

var (
	protocolProbeOnce bool
	protocolProbeKind protocolKind
	inTmux            = os.Getenv("TMUX") != ""
)

func initProtoProbe() {
	if protocolProbeOnce {
		return
	}
	protocolProbeOnce = true
	if kittySupported() {
		protocolProbeKind = protoNative
		return
	}
	protocolProbeKind = protoMosaic
}

// ImagePayload pairs encoded image data with the spec it was produced from.
// Callers use the spec path to dedupe re-emits across preview refreshes.
type ImagePayload struct {
	Spec ImageSpec
	Data string
}

func RenderImages(rendered string, specs []ImageSpec, cellW, cellH int) (string, []ImagePayload) {
	if !strings.Contains(rendered, ImgOpenPrefix) {
		return rendered, nil
	}
	initProtoProbe()
	pathID := make(map[string]uint32, len(specs))
	var nextID uint32 = 1
	for _, sp := range specs {
		if _, ok := pathID[sp.Path]; !ok {
			pathID[sp.Path] = nextID
			nextID++
		}
	}
	var payloads []ImagePayload
	out := imgSentinelRE.ReplaceAllStringFunc(rendered, func(match string) string {
		sm := imgSentinelRE.FindStringSubmatch(match)
		idx, parseErr := strconv.Atoi(sm[1])
		if parseErr != nil || idx < 0 || idx >= len(specs) {
			return match
		}
		spec := specs[idx]
		id := pathID[spec.Path]
		switch protocolProbeKind {
		case protoNative:
			data, err := EncodeKittyImageData(spec, id, cellW, cellH)
			if err != nil {
				return placeholderBox(spec.Alt, spec.Path, max(cellW-2, 12), 5)
			}
			if len(data) > 0 {
				payloads = append(payloads, ImagePayload{Spec: spec, Data: string(data)})
			}
			return renderPlaceholder(spec, id, cellW, cellH)
		default:
			return renderMosaicTile(spec, cellW, cellH)
		}
	})
	return out, payloads
}

// EncodeKittyImageData returns the raw ESC_G payload for a spec, wrapped in
// DCS passthrough when running inside tmux. The returned bytes must be sent
// to the terminal BEFORE the placeholder cells are redrawn so kitty can
// associate the image id with the placeholder positions. Sending this through
// Bubble Tea's cell renderer doesn't work because ultraviolet's printString
// overwrites cell.Content on the next printable rune, dropping DCS bytes.
func EncodeKittyImageData(spec ImageSpec, id uint32, cellW, cellH int) ([]byte, error) {
	loaded, err := LoadImageFull(spec)
	if err != nil || len(loaded.frames) == 0 {
		return nil, err
	}
	maxCellsH := maxAvailableImageRows(cellH)
	maxCellsW := max((cellW-4)/2, 4)
	cw, ch := cellDims(
		loaded.frames[0].Bounds().Dx(),
		loaded.frames[0].Bounds().Dy(),
		maxCellsW, maxCellsH,
	)
	const phPxW, phPxH = 16, 16
	wPx := cw * phPxW
	hPx := ch * phPxH
	ensureTmuxPassthrough()
	var buf bytes.Buffer
	if loaded.animated {
		if _, encErr := encodeKittyAnimated(&buf, loaded.frames, loaded.delays, wPx, hPx, id, cw, ch); encErr != nil {
			return nil, encErr
		}
	} else {
		if _, encErr := encodeKittyStatic(&buf, loaded.frames[0], wPx, hPx, id, cw, ch); encErr != nil {
			return nil, encErr
		}
	}
	return buf.Bytes(), nil
}

// renderPlaceholder returns the cell-renderer-safe block that anchors the
// image at the sentinel position. The FG color encodes the image id; the
// diacritics encode each cell's (row, col) within the area so kitty can place
// the image at the correct virtual position. A Charple border frames the
// grid so the image boundary is visible.
func renderPlaceholder(spec ImageSpec, id uint32, cellW, cellH int) string {
	maxCellsH := maxAvailableImageRows(cellH)
	maxCellsW := max((cellW-4)/2, 4)
	nw, nh := loadFirstFrameDims(spec, maxCellsW, maxCellsH)
	cw, ch := cellDims(nw, nh, maxCellsW, maxCellsH)
	title := spec.Alt
	if title == "" {
		title = filepath.Base(spec.Path)
	}
	var buf bytes.Buffer
	EmitPlaceholderBlock(&buf, id, cw, ch, title)
	return buf.String()
}

// maxAvailableImageRows caps how tall an image can be in cells given the
// available pane height. We reserve a few rows for padding and borders so
// the image doesn't slam into the status bar or header.
func maxAvailableImageRows(cellH int) int {
	if cellH <= 0 {
		return 20
	}
	return min(max(cellH-6, 5), 40)
}

// MaxAvailableImageRowsForTest exposes maxAvailableImageRows for diagnostic
// tools outside the package. The Max* naming convention matches the rest of
// the preview API.
func MaxAvailableImageRowsForTest(cellH int) int { return maxAvailableImageRows(cellH) }

// CellDimsForTest exposes cellDims for diagnostic tools.
func CellDimsForTest(natW, natH, maxCellsW, maxCellsH int) (int, int) {
	return cellDims(natW, natH, maxCellsW, maxCellsH)
}

// loadFirstFrameDims returns the natural (width, height) of the first frame
// for the spec, falling back to (maxCellsW, maxCellsH) when the source can't
// be loaded — the placeholder still needs a non-zero size so the layout is
// predictable.
func loadFirstFrameDims(spec ImageSpec, maxCellsW, maxCellsH int) (int, int) {
	loaded, err := LoadImageFull(spec)
	if err != nil || len(loaded.frames) == 0 {
		return maxCellsW, maxCellsH
	}
	b := loaded.frames[0].Bounds()
	return b.Dx(), b.Dy()
}

// renderMosaicTile is the fallback for terminals without kitty graphics.
// Mosaic tiles fit naturally inside a single cell's content so they go
// through the cell renderer without needing the side channel.
func renderMosaicTile(spec ImageSpec, cellW, cellH int) string {
	loaded, err := LoadImageFull(spec)
	if err != nil || len(loaded.frames) == 0 {
		return placeholderBox(spec.Alt, spec.Path, max(cellW-2, 12), 5)
	}
	maxCellsH := maxAvailableImageRows(cellH)
	maxCellsW := max((cellW-2)/2, 8)
	cw, ch := cellDims(
		loaded.frames[0].Bounds().Dx(),
		loaded.frames[0].Bounds().Dy(),
		maxCellsW, maxCellsH,
	)
	img := loaded.frames[0]
	wPxM, hPxM := cellDims(img.Bounds().Dx(), img.Bounds().Dy(), cw, ch)
	scaled := scaleToCells(img, wPxM, hPxM)
	m := mosaic.New().Width(wPxM).Height(hPxM / 2)
	var buf bytes.Buffer
	buf.WriteString(m.Render(scaled))
	return buf.String()
}

// EmitPlaceholderBlock writes the unicode placeholder grid that the
// terminal interprets via the kitty virtual-placement scheme, surrounded by a
// Charple-colored box-drawing border so the image boundary is visible before
// kitty renders anything. The id is encoded into the FG color so kitty can
// associate each placeholder cell with the correct image; every cell carries
// both the row and column diacritic so kitty can anchor each pixel of the
// image to the correct cell. The optional third diacritic carries the high
// byte of the image id when ids exceed 255 (the indexed-color ceiling). IDs
// <= 255 use an indexed color slot, IDs > 255 use a truecolor RGB. The
// title (typically the alt text or path basename) is rendered inside the
// bottom-left corner of the border so users can identify the image before
// it loads. The trailing reset keeps style state from leaking past the
// grid.
//
// The placeholder grid itself remains exactly width × height cells — the
// border and inner padding are emitted around it and do not consume any of
// the kitty virtual-placement cells. The total visual width is
// width*2 + 4 (border + 1-cell padding each side).
func EmitPlaceholderBlock(w io.Writer, id uint32, width, height int, title string) {
	if width < 1 || height < 1 {
		return
	}
	extra := int((id >> 24) & 0xff)
	r := int((id >> 16) & 0xff)
	g := int((id >> 8) & 0xff)
	b := int(id & 0xff)

	var fgStyle string
	if id <= 255 {
		fgStyle = fmt.Sprintf("\x1b[38;5;%dm", b)
	} else {
		fgStyle = fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	}

	borderStyle := "\x1b[38;5;245m"
	titleStyle := "\x1b[38;5;245m\x1b[3m"
	resetStyle := "\x1b[0m"
	ph := string(kittyPlaceholder)
	innerW := width * 2

	// Top border: ┌────────────┐
	io.WriteString(w, borderStyle)
	io.WriteString(w, "┌")
	io.WriteString(w, strings.Repeat("─", innerW+2))
	io.WriteString(w, "┐\n")
	io.WriteString(w, resetStyle)

	for y := range height {
		rowDiac := string(diacritics[y%len(diacritics)])
		io.WriteString(w, borderStyle)
		io.WriteString(w, "│ ")
		io.WriteString(w, fgStyle)
		for x := range width {
			io.WriteString(w, ph)
			io.WriteString(w, rowDiac)
			diacriticWrite(w, x)
			if extra > 0 {
				diacriticWrite(w, extra)
			}
		}
		io.WriteString(w, resetStyle)
		io.WriteString(w, borderStyle)
		io.WriteString(w, " │\n")
		io.WriteString(w, resetStyle)
	}
	// Bottom border with title: └─[title]──────┘
	truncated := title
	if lipgloss.Width(truncated) > innerW {
		truncated = truncated[:max(innerW-1, 1)] + "…"
	}
	pad := max(innerW-lipgloss.Width(truncated), 0)
	io.WriteString(w, borderStyle)
	io.WriteString(w, "└─")
	io.WriteString(w, titleStyle)
	io.WriteString(w, truncated)
	io.WriteString(w, resetStyle)
	io.WriteString(w, borderStyle)
	io.WriteString(w, strings.Repeat("─", pad))
	io.WriteString(w, "┘\n")
	io.WriteString(w, resetStyle)
}

func diacriticWrite(w io.Writer, i int) {
	if i < len(diacritics) {
		fmt.Fprintf(w, "%c", diacritics[i])
		return
	}
	fmt.Fprintf(w, "%c", diacritics[0])
}

func ensureTmuxPassthrough() {
	if !inTmux {
		return
	}
	// Best-effort: ask tmux to enable allow-passthrough on the current pane.
	// Errors are ignored — the user may not have tmux in PATH, may be on
	// an old version, or may have already enabled it.
	_ = exec.Command("tmux", "set", "-p", "allow-passthrough", "on").Run()
}

// placeholderBox renders a compact "image not loaded" panel sized to fit
// inside a preview pane without overlapping the bottom title row. The src
// argument is the raw URL or filesystem path so users can see what failed.
func placeholderBox(alt, src string, w, h int) string {
	if alt == "" {
		alt = "image"
	}
	if w < 6 {
		w = 6
	}
	if h < 3 {
		h = 3
	}
	innerW := w - 2

	label := "[ " + alt + " ]"
	if lipgloss.Width(label) > innerW {
		label = label[:max(innerW-3, 1)] + "…"
	}

	srcLine := src
	if srcLine == "" {
		srcLine = "(no source)"
	}
	if lipgloss.Width(srcLine) > innerW {
		srcLine = srcLine[:max(innerW-1, 1)] + "…"
	}

	labelPad := (innerW - lipgloss.Width(label)) / 2
	labelLeftPad := strings.Repeat(" ", max(labelPad, 0))
	labelRightPad := strings.Repeat(" ", max(innerW-labelPad-lipgloss.Width(label), 0))

	srcPad := (innerW - lipgloss.Width(srcLine)) / 2
	srcLeftPad := strings.Repeat(" ", max(srcPad, 0))
	srcRightPad := strings.Repeat(" ", max(innerW-srcPad-lipgloss.Width(srcLine), 0))

	top := lipgloss.NewStyle().Foreground(uict.Sriracha).Render("┌" + strings.Repeat("─", innerW) + "┐")
	bot := lipgloss.NewStyle().Foreground(uict.Sriracha).Render("└" + strings.Repeat("─", innerW) + "┘")
	borders := lipgloss.NewStyle().Foreground(uict.Sriracha)
	labelLine := borders.Render("│") +
		labelLeftPad +
		lipgloss.NewStyle().Foreground(uict.Sriracha).Bold(true).Render(label) +
		labelRightPad +
		borders.Render("│")
	srcLineRendered := borders.Render("│") +
		srcLeftPad +
		lipgloss.NewStyle().Foreground(uict.Squid).Render(srcLine) +
		srcRightPad +
		borders.Render("│")
	emptyMid := borders.Render("│" + strings.Repeat(" ", innerW) + "│\n")

	midCount := max(h-4, 0)
	mid := strings.Repeat(emptyMid, midCount)
	return top + "\n" + mid + labelLine + "\n" + srcLineRendered + "\n" + bot + "\n"
}
