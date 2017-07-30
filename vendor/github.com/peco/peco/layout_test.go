package peco

import (
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

func TestLayoutType(t *testing.T) {
	layouts := []struct {
		value    LayoutType
		expectOK bool
	}{
		{LayoutTypeTopDown, true},
		{LayoutTypeBottomUp, true},
		{"foobar", false},
	}
	for _, l := range layouts {
		valid := IsValidLayoutType(l.value)
		if valid != l.expectOK {
			t.Errorf("LayoutType %s, expected IsValidLayoutType to return %t, but got %t",
				l.value,
				l.expectOK,
				valid,
			)
		}
	}
}

func TestPrintScreen(t *testing.T) {
	screen := NewDummyScreen()

	makeVerifier := func(initX, initY int, fill bool) func(string) {
		return func(msg string) {
			screen.interceptor.reset()
			t.Logf("Checking printScreen(%d, %d, %s, %t)", initX, initY, msg, fill)
			width := utf8.RuneCountInString(msg)
			screen.Print(PrintArgs{
				X:    initX,
				Y:    initY,
				Fg:   termbox.ColorDefault,
				Bg:   termbox.ColorDefault,
				Msg:  msg,
				Fill: fill,
			})
			events := screen.interceptor.events["SetCell"]
			if !fill {
				if len(events) != width {
					t.Errorf("Expected %d SetCell events, got %d",
						width,
						len(events),
					)
				}
				return
			}

			// fill == true
			w, _ := screen.Size()
			if rw := runewidth.StringWidth(msg); rw != width {
				w -= rw - width
			}
			if len(events) != w {
				t.Errorf("Expected %d SetCell events, got %d",
					w,
					len(events),
				)
				return
			}
		}
	}

	verify := makeVerifier(0, 0, false)
	verify("Hello, World!")
	verify("日本語")

	verify = makeVerifier(0, 0, true)
	verify("Hello, World!")
	verify("日本語")
}

func TestStatusBar(t *testing.T) {
	screen := NewDummyScreen()
	st := NewStatusBar(screen, AnchorBottom, 0, NewStyleSet())
	st.PrintStatus("Hello, World!", 0)

	events := screen.interceptor.events
	if l := len(events["Flush"]); l != 1 {
		t.Errorf("Expected 1 Flush event, got %d", l)
		return
	}
}

func TestMergeAttribute(t *testing.T) {
	colors := stringToFg

	// merge colors
	tests := [][]string{
		{"red", "green", "yellow"},
		{"red", "blue", "magenta"},
		{"green", "blue", "cyan"},
		{"yellow", "blue", "white"},
		{"magenta", "green", "white"},
		{"cyan", "red", "white"},
		{"yellow", "magenta", "white"},
		{"magenta", "cyan", "white"},
		{"cyan", "yellow", "white"},
	}

	for _, c := range tests {
		if m := mergeAttribute(colors[c[0]], colors[c[1]]); m != colors[c[2]] {
			t.Errorf("(%s + %s) expected %d(%s), got %d", c[0], c[1], colors[c[2]], c[2], m)
		}
	}

	// merge with white
	for _, c := range colors {
		if m := mergeAttribute(c, colors["white"]); m != colors["white"] {
			t.Errorf("expected white(%d), got %d", colors["white"], m)
		}
	}

	// merge attributes
	if m := mergeAttribute(termbox.AttrBold|colors["red"], termbox.AttrUnderline|colors["cyan"]); m != termbox.AttrBold|termbox.AttrUnderline|colors["white"] {
		t.Errorf("expected %d, got %d", termbox.AttrBold|termbox.AttrUnderline|colors["white"], m)
	}

}
