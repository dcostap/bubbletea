package tea

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	windowsPasteStartWindow    = 5 * time.Millisecond
	windowsPasteContinueWindow = 60 * time.Millisecond
	windowsPasteBurstRunes     = 2
)

type windowsPasteBurst struct {
	active     bool
	burst      bool
	hasNewline bool
	runes      int
	deadline   time.Time
	buffered   []Msg
	text       strings.Builder
}

func (b *windowsPasteBurst) Push(msg Msg, now time.Time) []Msg {
	if !b.active {
		text, ok := pasteBurstStartText(msg)
		if !ok {
			return []Msg{msg}
		}
		b.start(msg, text, now)
		return nil
	}

	if text, ok := pasteBurstContinueText(msg); ok {
		b.buffered = append(b.buffered, msg)
		b.appendText(text)
		b.extend(now)
		return nil
	}

	if isPasteBurstAuxiliaryMsg(msg) {
		b.buffered = append(b.buffered, msg)
		return nil
	}

	out := b.Flush()
	return append(out, msg)
}

func (b *windowsPasteBurst) Flush() []Msg {
	if !b.active {
		return nil
	}

	out := b.flushLocked()
	return out
}

func (b *windowsPasteBurst) Due(now time.Time) bool {
	return b.active && !now.Before(b.deadline)
}

func (b *windowsPasteBurst) NextDelay(now time.Time) (time.Duration, bool) {
	if !b.active {
		return 0, false
	}

	if !now.Before(b.deadline) {
		return 0, true
	}
	return time.Until(b.deadline), true
}

func (b *windowsPasteBurst) start(msg Msg, text string, now time.Time) {
	b.active = true
	b.buffered = append(b.buffered[:0], msg)
	b.text.Reset()
	b.text.WriteString(text)
	b.runes = utf8.RuneCountInString(text)
	b.hasNewline = strings.Contains(text, "\n")
	b.burst = b.runes >= windowsPasteBurstRunes || b.hasNewline
	b.deadline = now.Add(windowsPasteStartWindow)
	if b.burst {
		b.deadline = now.Add(windowsPasteContinueWindow)
	}
}

func (b *windowsPasteBurst) appendText(text string) {
	b.text.WriteString(text)
	b.runes += utf8.RuneCountInString(text)
	b.hasNewline = b.hasNewline || strings.Contains(text, "\n")
	if b.runes >= windowsPasteBurstRunes || b.hasNewline {
		b.burst = true
	}
}

func (b *windowsPasteBurst) extend(now time.Time) {
	if b.burst {
		b.deadline = now.Add(windowsPasteContinueWindow)
		return
	}
	b.deadline = now.Add(windowsPasteStartWindow)
}

func (b *windowsPasteBurst) flushLocked() []Msg {
	var out []Msg
	if b.burst {
		out = append(out, PasteMsg{Content: b.text.String()})
	} else {
		out = append(out, b.buffered...)
	}

	b.active = false
	b.burst = false
	b.hasNewline = false
	b.runes = 0
	b.buffered = b.buffered[:0]
	b.text.Reset()
	b.deadline = time.Time{}
	return out
}

func pasteBurstStartText(msg Msg) (string, bool) {
	key, ok := pasteBurstKeyPress(msg)
	if !ok || key.Text == "" {
		return "", false
	}
	return key.Text, true
}

func pasteBurstContinueText(msg Msg) (string, bool) {
	key, ok := pasteBurstKeyPress(msg)
	if !ok {
		return "", false
	}
	if key.Text != "" {
		return key.Text, true
	}
	if key.Mod != 0 {
		return "", false
	}
	switch key.Code {
	case KeyEnter, KeyKpEnter:
		return "\n", true
	default:
		return "", false
	}
}

func pasteBurstKeyPress(msg Msg) (KeyPressMsg, bool) {
	key, ok := msg.(KeyPressMsg)
	if !ok {
		return KeyPressMsg{}, false
	}
	return key, true
}

func isPasteBurstAuxiliaryMsg(msg Msg) bool {
	switch msg := msg.(type) {
	case KeyReleaseMsg:
		return true
	case KeyPressMsg:
		return isModifierOnlyKeyPress(msg)
	default:
		return false
	}
}

func isModifierOnlyKeyPress(msg KeyPressMsg) bool {
	if msg.Text != "" {
		return false
	}

	switch msg.Code {
	case KeyLeftShift, KeyRightShift,
		KeyLeftCtrl, KeyRightCtrl,
		KeyLeftAlt, KeyRightAlt,
		KeyLeftSuper, KeyRightSuper,
		KeyCapsLock, KeyScrollLock,
		KeyNumLock:
		return true
	default:
		return false
	}
}
