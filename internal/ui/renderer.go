package ui

import (
	"path/filepath"
	"strings"

	"github.com/N1xev/mditor/internal/preview"
)

func (m *Model) refreshRenderer() {
	if m.Cur() == nil {
		return
	}
	w := m.previewWidth()
	if w == m.LastPreviewW && m.Renderer != nil {
		return
	}
	if r, err := preview.NewRenderer(w); err == nil {
		m.Renderer = r
		m.LastPreviewW = w
	}
}

func (m *Model) refreshPreview() ([]string, error) {
	if m.Cur() == nil {
		m.Preview = ""
		m.Vp.SetContent("")
		m.CodeBlockContents = nil
		return nil, nil
	}
	w := m.previewWidth()
	if m.Renderer == nil {
		if r, err := preview.NewRenderer(w); err == nil {
			m.Renderer = r
		} else {
			return nil, err
		}
	}
	currentValue := m.Cur().TA.Value()
	t := m.Cur()
	if t.PreviewCache != "" && t.PreviewCacheSrc == currentValue && t.PreviewCacheW == w {
		m.Preview = t.PreviewCache
		m.CodeBlockContents = t.PreviewCodeBlocks
		m.Vp.SetContent(m.Preview)
		var pending []string
		for _, p := range t.PreviewImageData {
			pending = append(pending, p.Data)
		}
		return pending, nil
	}
	if t.EmittedImagePaths == nil {
		t.EmittedImagePaths = make(map[string]bool)
	}
	if t.PreviewCache == "" || t.PreviewCacheSrc != currentValue || t.PreviewCacheW != w {
		t.EmittedImagePaths = make(map[string]bool)
	}
	baseDir := ""
	if fn := t.Filename; fn != "" {
		baseDir = filepath.Dir(fn)
	}
	out, blocks, allPayloads := m.Renderer.RenderWithCodeBlocksBase(currentValue, baseDir, w, m.Height-3, w)
	var contents []string
	if len(blocks) > 0 {
		contents = make([]string, len(blocks))
		for _, b := range blocks {
			if b.BlockID < len(contents) {
				contents[b.BlockID] = b.Code
			}
		}
	}
	var newPayloads []preview.ImagePayload
	var pending []string
	for _, p := range allPayloads {
		if !t.EmittedImagePaths[p.Spec.Path] {
			t.EmittedImagePaths[p.Spec.Path] = true
			newPayloads = append(newPayloads, p)
			pending = append(pending, p.Data)
		}
	}
	t.PreviewCache = out
	t.PreviewCacheSrc = currentValue
	t.PreviewCacheW = w
	t.PreviewCodeBlocks = contents
	t.PreviewImageData = newPayloads
	m.Preview = out
	m.CodeBlockContents = contents
	m.Vp.SetContent(m.Preview)
	return pending, nil
}

func (m *Model) syncEditorToPreview() {
	if m.Cur() == nil {
		return
	}
	taTotal := m.Cur().TA.LineCount()
	if taTotal <= 0 {
		return
	}
	taLine := max(m.Cur().TA.Line(), m.Cur().TA.ScrollYOffset())

	previewMaxOffset := m.previewMaxYOffset()
	if previewMaxOffset <= 0 {
		return
	}
	target := (taLine * previewMaxOffset) / max(taTotal-1, 1)
	target = max(0, min(target, previewMaxOffset))
	m.Vp.SetYOffset(target)
}

func (m *Model) syncPreviewToEditor() {
	if m.Cur() == nil {
		return
	}
	previewMaxOffset := m.previewMaxYOffset()
	previewProgress := m.Vp.YOffset()
	taTotal := m.Cur().TA.LineCount()
	if taTotal <= 0 {
		return
	}
	var target int
	if previewMaxOffset <= 0 {
		target = 0
	} else {
		target = (previewProgress * max(taTotal-1, 1)) / previewMaxOffset
	}
	target = max(0, min(target, taTotal-1))
	cur := m.Cur().TA.Line()
	if cur > target {
		m.Cur().TA.MoveToBegin()
		cur = 0
	}
	ta := &m.Cur().TA
	for cur < target {
		ta.CursorDown()
		cur++
	}
}

func (m *Model) previewMaxYOffset() int {
	total := strings.Count(m.Preview, "\n") + 1
	return max(0, total-m.Vp.Height())
}
