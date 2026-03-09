package tea

import (
	"reflect"
	"testing"
	"time"
)

func TestWindowsPasteBurstCoalescesRapidText(t *testing.T) {
	var burst windowsPasteBurst
	now := time.Unix(0, 0)

	for _, msg := range []Msg{
		KeyPressMsg{Code: 'a', Text: "a"},
		KeyReleaseMsg{Code: 'a'},
		KeyPressMsg{Code: 'b', Text: "b"},
		KeyReleaseMsg{Code: 'b'},
		KeyPressMsg{Code: KeyEnter},
		KeyReleaseMsg{Code: KeyEnter},
		KeyPressMsg{Code: 'c', Text: "c"},
	} {
		_ = burst.Push(msg, now)
		now = now.Add(time.Millisecond)
	}

	got := burst.Flush()
	want := []Msg{
		PasteMsg{Content: "ab\nc"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected burst flush:\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestWindowsPasteBurstPreservesSingleKey(t *testing.T) {
	var burst windowsPasteBurst
	now := time.Unix(0, 0)

	if got := burst.Push(KeyPressMsg{Code: 'a', Text: "a"}, now); got != nil {
		t.Fatalf("expected initial key to stay buffered, got %#v", got)
	}

	got := burst.Flush()
	want := []Msg{
		KeyPressMsg{Code: 'a', Text: "a"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected single-key flush:\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestWindowsPasteBurstPreservesSingleEnter(t *testing.T) {
	var burst windowsPasteBurst
	now := time.Unix(0, 0)

	got := burst.Push(KeyPressMsg{Code: KeyEnter}, now)
	want := []Msg{
		KeyPressMsg{Code: KeyEnter},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected single-enter push:\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestWindowsPasteBurstFlushesBeforeUnrelatedMsg(t *testing.T) {
	var burst windowsPasteBurst
	now := time.Unix(0, 0)

	_ = burst.Push(KeyPressMsg{Code: 'a', Text: "a"}, now)
	_ = burst.Push(KeyPressMsg{Code: 'b', Text: "b"}, now.Add(time.Millisecond))

	got := burst.Push(WindowSizeMsg{Width: 80, Height: 24}, now.Add(2*time.Millisecond))
	want := []Msg{
		PasteMsg{Content: "ab"},
		WindowSizeMsg{Width: 80, Height: 24},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected flush-before-msg:\nwant: %#v\ngot:  %#v", want, got)
	}
}
