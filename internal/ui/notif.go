package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

type notification struct {
	Text    string
	IsErr   bool
	Expires time.Time
}

type notifExpiredMsg struct{}

func newNotif(text string, isErr bool) notification {
	return notification{Text: text, IsErr: isErr, Expires: time.Now().Add(3 * time.Second)}
}

func (n notification) alive() bool {
	return !n.Expires.IsZero() && time.Now().Before(n.Expires)
}

func notifExpireCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg { return notifExpiredMsg{} })
}

func remainingNotifTTL(n notification) time.Duration {
	if n.Expires.IsZero() {
		return 200 * time.Millisecond
	}
	d := time.Until(n.Expires)
	if d < 50*time.Millisecond {
		return 200 * time.Millisecond
	}
	return d
}

func (m *Model) notifTickOnce() tea.Cmd {
	return notifExpireCmd(remainingNotifTTL(m.Notif))
}

func (m *Model) setNotif(text string, isErr bool) tea.Cmd {
	m.Notif = newNotif(text, isErr)
	m.ViewDirty = true
	return notifExpireCmd(remainingNotifTTL(m.Notif))
}

func (m *Model) setErrNotif(text string) tea.Cmd {
	m.Notif = newNotif(text, true)
	m.ViewDirty = true
	return notifExpireCmd(remainingNotifTTL(m.Notif))
}
