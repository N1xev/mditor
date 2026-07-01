package ui

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	"github.com/charmbracelet/x/ansi"

	"github.com/N1xev/mditor/internal/edit"
)

type sidebarDelegate struct {
	inner       list.DefaultDelegate
	activePath  string
	focusOnSelf bool
}

func (d *sidebarDelegate) setActive(path string, focused bool) {
	d.activePath = path
	d.focusOnSelf = focused
}

func (d *sidebarDelegate) Height() int  { return d.inner.Height() }
func (d *sidebarDelegate) Spacing() int { return d.inner.Spacing() }

func (d *sidebarDelegate) Update(msg teaMsg, m *list.Model) teaCmd {
	return d.inner.Update(msg, m)
}

func (d *sidebarDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if index == m.Index() && d.focusOnSelf {
		d.inner.Render(w, m, index, item)
		return
	}
	if sf, ok := item.(edit.SideFile); ok && sf.Path == d.activePath {
		d.renderActive(w, m, sf)
		return
	}
	d.renderNormal(w, m, index, item)
}

func (d *sidebarDelegate) renderActive(w io.Writer, m list.Model, sf edit.SideFile) {
	titleStyle := sidebarActiveTitle
	descStyle := sidebarActiveDesc
	title := ansi.Truncate(sf.Title(), m.Width()-titleStyle.GetPaddingLeft()-titleStyle.GetPaddingRight(), "…")
	if d.inner.ShowDescription {
		desc := ansi.Truncate(sf.Description(), m.Width()-descStyle.GetPaddingLeft()-descStyle.GetPaddingRight(), "…")
		_, _ = fmt.Fprintf(w, "%s\n%s", titleStyle.Render(title), descStyle.Render(desc))
		return
	}
	_, _ = fmt.Fprint(w, titleStyle.Render(title))
}

func (d *sidebarDelegate) renderNormal(w io.Writer, m list.Model, index int, item list.Item) {
	sf, ok := item.(edit.SideFile)
	if !ok {
		d.inner.Render(w, m, index, item)
		return
	}
	ns := d.inner.Styles.NormalTitle
	ds := d.inner.Styles.NormalDesc
	title := ansi.Truncate(sf.Title(), m.Width()-ns.GetPaddingLeft()-ns.GetPaddingRight(), "…")
	if d.inner.ShowDescription {
		desc := ansi.Truncate(sf.Description(), m.Width()-ds.GetPaddingLeft()-ds.GetPaddingRight(), "…")
		_, _ = fmt.Fprintf(w, "%s\n%s", ns.Render(title), ds.Render(desc))
		return
	}
	_, _ = fmt.Fprint(w, ns.Render(title))
}
