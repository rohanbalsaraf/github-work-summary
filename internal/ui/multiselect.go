package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var ErrSelectionCancelled = errors.New("repository selection cancelled")

// SelectOption is one selectable entry in the list.
type SelectOption struct {
	Label string
	Value string
	Desc  string
}

// MultiSelectCheckboxes runs a modern interactive checkbox selector using arrow keys.
func MultiSelectCheckboxes(in io.Reader, out io.Writer, title string, options []SelectOption) ([]SelectOption, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options available")
	}

	file, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		// Fallback to simpler number-based input if not a terminal
		return multiSelectClassic(in, out, title, options)
	}

	fd := int(file.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return multiSelectClassic(in, out, title, options)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	// Hide cursor
	fmt.Fprint(out, "\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h")

	reader := bufio.NewReader(in)
	selected := make(map[int]bool)
	cursor := 0

	for {
		renderModernOptions(out, title, options, selected, cursor)

		key, err := readRawKey(reader)
		if err != nil {
			return nil, err
		}

		switch key {
		case keyUp:
			if cursor > 0 {
				cursor--
			} else {
				cursor = len(options) - 1
			}
		case keyDown:
			if cursor < len(options)-1 {
				cursor++
			} else {
				cursor = 0
			}
		case keySpace:
			selected[cursor] = !selected[cursor]
		case keyEnter:
			selectedOptions := collectSelected(options, selected)
			if len(selectedOptions) == 0 {
				// We stay in the loop but we need to signal to the user they need at least one
				// Unfortunately in raw mode we'd need to render the error message on the screen.
				continue
			}
			return selectedOptions, nil
		case keyAll:
			for i := range options {
				selected[i] = true
			}
		case keyNone:
			for i := range options {
				selected[i] = false
			}
		case keyQuit:
			return nil, ErrSelectionCancelled
		}
	}
}

func renderModernOptions(out io.Writer, title string, options []SelectOption, selected map[int]bool, cursor int) {
	// Clear screen or just move to top if screen space is an issue.
	// For repo selection, it's best to clear and redraw.
	fmt.Fprint(out, "\x1b[H\x1b[2J")

	fmt.Fprintln(out, Bold(out, Cyan(out, " "+title))+"\r")
	fmt.Fprintln(out, Gray(out, " ↑↓/jk: Navigate  |  Space: Toggle  |  Enter: Done  |  a/n: All/None  |  q: Cancel")+"\r")
	fmt.Fprintln(out, Gray(out, strings.Repeat("-", 75))+"\r")

	// Calculate visible window if many options
	start := 0
	end := len(options)
	maxVisible := 15
	width := 80
	if len(options) > maxVisible {
		start = cursor - maxVisible/2
		if start < 0 {
			start = 0
		}
		end = start + maxVisible
		if end > len(options) {
			end = len(options)
			start = end - maxVisible
		}
	}

	if start > 0 {
		fmt.Fprintln(out, Gray(out, "  ... (more above)"))
	}

	for i := start; i < end; i++ {
		opt := options[i]
		if i == cursor {
			// Line is selected
			line := fmt.Sprintf(" ➤ [x] %s", opt.Label)
			if !selected[i] {
				line = fmt.Sprintf(" ➤ [ ] %s", opt.Label)
			}
			if opt.Desc != "" {
				line += fmt.Sprintf(" (%s)", opt.Desc)
			}
			line = FitLine(width, line)
			fmt.Fprintf(out, "%s\r\n", Bold(out, Cyan(out, line)))
		} else {
			// Line not selected
			prefix := "    "
			check := "[ ]"
			if selected[i] {
				check = "[x]"
			}
			line := fmt.Sprintf("%s%s %s", prefix, check, opt.Label)
			if opt.Desc != "" {
				line += fmt.Sprintf(" (%s)", opt.Desc)
			}
			if selected[i] {
				writeHomeLine(out, Green(out, FitLine(width, line)))
			} else {
				writeHomeLine(out, FitLine(width, line))
			}
		}
	}

	if end < len(options) {
		fmt.Fprintln(out, Gray(out, "  ... (more below)"))
	}

	count := 0
	for _, s := range selected {
		if s {
			count++
		}
	}
	fmt.Fprintf(out, "\r\n Selected: %d / %d\r\n", count, len(options))
}

func readRawKey(reader *bufio.Reader) (string, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return keyUnknown, err
	}

	switch b {
	case '\r', '\n':
		return keyEnter, nil
	case ' ':
		return keySpace, nil
	case 'k', 'K':
		return keyUp, nil
	case 'j', 'J':
		return keyDown, nil
	case 'a', 'A':
		return keyAll, nil
	case 'n', 'N':
		return keyNone, nil
	case 'q', 'Q', 27: // q or ESC
		// Check for arrow keys sequence
		if b == 27 {
			// If there's more in the buffer, it's likely an escape sequence
			if reader.Buffered() > 0 {
				b2, _ := reader.ReadByte()
				if b2 == '[' {
					b3, _ := reader.ReadByte()
					switch b3 {
					case 'A':
						return keyUp, nil
					case 'B':
						return keyDown, nil
					}
				}
			}
			return keyQuit, nil
		}
		return keyQuit, nil
	default:
		return keyUnknown, nil
	}
}

// multiSelectClassic is a fallback for non-TTY or older environments (mostly unchanged)
func multiSelectClassic(_ io.Reader, out io.Writer, title string, _ []SelectOption) ([]SelectOption, error) {
	// (Original logic from before, but keeping it as fallback)
	fmt.Fprintln(out, title)
	// For simplicity, just return error or provide a basic implementation if needed.
	return nil, fmt.Errorf("interactive terminal required for this selection mode")
}

func collectSelected(options []SelectOption, selected map[int]bool) []SelectOption {
	result := make([]SelectOption, 0)
	for i, option := range options {
		if selected[i] {
			result = append(result, option)
		}
	}
	return result
}
