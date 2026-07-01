package ui

func (m *Model) relayout() {
	statusH := 1
	headerH := 2
	paneOuterH := max(5, m.Height-headerH-statusH)
	contentH := paneOuterH - 2

	availW := m.Width
	if m.ShowSidebar {
		availW -= sidebarWidth
	}
	if m.ShowSidebar {
		m.FileList.SetWidth(sidebarWidth - 2)
		m.FileList.SetHeight(max(0, contentH-1))
	}

	if m.Cur() == nil {
		return
	}

	m.invalidateTAView()
	m.DispMap = nil

	boxW := max(8, availW-2)

	m.Cur().TA.MaxHeight = 0
	m.Cur().TA.MaxWidth = 0

	switch m.Mode {
	case modeEdit:
		m.Cur().TA.SetHeight(contentH)
		m.Cur().TA.SetWidth(boxW)
		m.Vp.SetWidth(boxW)
		m.Vp.SetHeight(contentH)
	case modeMixed:
		halfBox := max(8, availW/2)
		m.Cur().TA.SetHeight(contentH)
		m.Cur().TA.SetWidth(halfBox)
		m.Vp.SetWidth(halfBox)
		m.Vp.SetHeight(contentH)
	case modeView:
		m.Vp.SetWidth(boxW)
		m.Vp.SetHeight(contentH)
	}
}

func (m Model) contentWidth() int {
	w := m.Width
	if m.ShowSidebar {
		w -= sidebarWidth
	}
	return w
}

func (m Model) previewPaneStartX() int {
	paneLeft := 1
	if m.ShowSidebar {
		paneLeft = sidebarWidth + 1
	}
	if m.Mode == modeMixed {
		return paneLeft + m.contentWidth()/2 + 1
	}
	return paneLeft + 1
}

func (m Model) textStartX() int {
	paneLeft := 1
	if m.ShowSidebar {
		paneLeft = sidebarWidth + 1
	}
	return paneLeft + 1 + m.numDigitsForGutter() + 2
}

func (m *Model) previewWidth() int {
	availW := m.Width
	if m.ShowSidebar {
		availW -= sidebarWidth
	}
	if m.Mode == modeMixed {
		half := (availW - 2) / 2
		return max(20, half-2)
	}
	return max(20, availW-2)
}

func (m Model) numDigitsForGutter() int {
	if m.Cur() == nil {
		return 1
	}
	n := m.Cur().TA.MaxHeight
	if n <= 0 {
		return 1
	}
	d := 0
	for n > 0 {
		d++
		n /= 10
	}
	return d
}

func (m Model) contentOffsetY() int { return 3 }

func (m Model) inPreviewPane(x, y int) bool {
	if m.Mode == modeView || m.Cur() == nil {
		return y >= m.contentOffsetY()
	}
	if m.Mode == modeMixed {
		paneX := 1
		if m.ShowSidebar {
			paneX = sidebarWidth + 1
		}
		paneX += m.contentWidth() / 2
		return x >= paneX && y >= m.contentOffsetY()
	}
	return false
}
