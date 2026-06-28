package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type HomeAction string

const (
	HomeActionSummary       HomeAction = "summary"
	HomeActionRepos         HomeAction = "repos"
	HomeActionLogin         HomeAction = "login"
	HomeActionLogout        HomeAction = "logout"
	HomeActionHelp          HomeAction = "help"
	HomeActionPR            HomeAction = "pr"
	HomeActionSchedule      HomeAction = "schedule"
	HomeActionWebhook       HomeAction = "webhook"
	HomeActionTickets       HomeAction = "tickets"
	HomeActionAILogin       HomeAction = "ai-login"
	HomeActionShare         HomeAction = "share"
	HomeActionSwitchProfile HomeAction = "switch-profile"
	HomeActionVersion       HomeAction = "version"
	HomeActionQuit          HomeAction = "quit"
)

type HomeMenuOptions struct {
	RepositoryURL string
	Tagline       string
	Version       string
	ActiveProfile string
}

type homeItem struct {
	Number int
	Label  string
	Desc   string
	Action HomeAction
}

var mainHomeItems = []homeItem{
	{Number: 1, Label: "Work Summary", Desc: "Generate your daily/trend reports", Action: HomeActionSummary},
	{Number: 2, Label: "Create PR", Desc: "AI-powered PR from current branch", Action: HomeActionPR},
	{Number: 3, Label: "Schedule", Desc: "Configure automated daily reports", Action: HomeActionSchedule},
	{Number: 4, Label: "Tickets", Desc: "Jira or Linear integration", Action: HomeActionTickets},
}

var extraHomeItems = []homeItem{
	{Number: 5, Label: "Profiles", Desc: "Manage multiple configurations", Action: HomeActionSwitchProfile},
	{Number: 6, Label: "Share Setup", Desc: "Configure Notion/Teams/Email", Action: HomeActionShare},
	{Number: 7, Label: "Webhook", Desc: "Event-driven summary listener", Action: HomeActionWebhook},
	{Number: 8, Label: "Repos", Desc: "Choose repositories to track", Action: HomeActionRepos},
	{Number: 9, Label: "Auth/AI", Desc: "GitHub Login/Logout & AI keys", Action: HomeActionLogin},
	{Number: 0, Label: "Quit", Desc: "Exit menu", Action: HomeActionQuit},
}

// IsInteractiveTerminal reports whether input can run interactive raw-key UI.
func IsInteractiveTerminal(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// RunHomeMenu displays an interactive startup dashboard and returns the selected action.
func RunHomeMenu(in *os.File, out io.Writer, opts HomeMenuOptions) (HomeAction, error) {
	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		return HomeActionHelp, nil
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return HomeActionHelp, err
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	fmt.Fprint(out, "\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h")

	reader := bufio.NewReader(in)
	selected := 0
	showMore := false

	for {
		items := visibleHomeItems(showMore)
		if selected >= len(items) {
			selected = len(items) - 1
		}
		width := terminalWidth(fd)
		renderHomeMenu(out, items, selected, showMore, opts, width)

		key, raw, err := readHomeKey(reader)
		if err != nil {
			return HomeActionQuit, err
		}

		switch key {
		case keyUp:
			if selected > 0 {
				selected--
			}
		case keyDown:
			if selected < len(items)-1 {
				selected++
			}
		case keyEnter:
			return items[selected].Action, nil
		case keyMore:
			showMore = !showMore
			if !showMore && selected >= len(mainHomeItems) {
				selected = len(mainHomeItems) - 1
			}
		case keyQuit:
			return HomeActionQuit, nil
		case keyUnknown:
			if action, ok := actionFromDigit(raw, items); ok {
				return action, nil
			}
		}
	}
}

func visibleHomeItems(showMore bool) []homeItem {
	if !showMore {
		return mainHomeItems
	}
	items := make([]homeItem, 0, len(mainHomeItems)+len(extraHomeItems))
	items = append(items, mainHomeItems...)
	items = append(items, extraHomeItems...)
	return items
}

func renderHomeMenu(out io.Writer, items []homeItem, selected int, showMore bool, opts HomeMenuOptions, width int) {
	repoURL := strings.TrimSpace(opts.RepositoryURL)
	if repoURL == "" {
		repoURL = "https://github.com/RDX463/github-work-summary"
	}
	tagline := strings.TrimSpace(opts.Tagline)
	if tagline == "" {
		tagline = "Summarize your GitHub work from terminal."
	}

	fmt.Fprint(out, "\x1b[H\x1b[2J")
	writeHomeLine(out, Bold(out, Cyan(out, fitLine(width, "  ____ _ _   _   _       __        __         _"))))
	writeHomeLine(out, Bold(out, Cyan(out, fitLine(width, " / ___(_) |_| | | | ___  \\ \\      / /__  _ __| | __"))))
	writeHomeLine(out, Bold(out, Cyan(out, fitLine(width, "| |  _| | __| |_| |/ _ \\  \\ \\ /\\ / / _ \\| '__| |/ /"))))
	writeHomeLine(out, Bold(out, Cyan(out, fitLine(width, "| |_| | | |_|  _  | (_) |  \\ V  V / (_) | |  |   <"))))
	writeHomeLine(out, Bold(out, Cyan(out, fitLine(width, " \\____|_|\\__|_| |_|\\___/    \\_/\\_/ \\___/|_|  |_|\\_\\"))))
	writeHomeLine(out, Gray(out, fitLine(width, " "+repoURL)))
	writeHomeLine(out, Gray(out, fitLine(width, " "+tagline)))
	if opts.Version != "" {
		writeHomeLine(out, Gray(out, fitLine(width, " Version: "+opts.Version)))
	}
	if opts.ActiveProfile != "" {
		writeHomeLine(out, Bold(out, Cyan(out, fitLine(width, " Profile: "+opts.ActiveProfile))))
	}
	writeHomeLine(out, "")

	for i, item := range items {
		prefix := "  "
		if i == selected {
			prefix = "➤ "
		}
		line := fmt.Sprintf("%s%d. %-10s %s", prefix, item.Number, item.Label, item.Desc)
		line = fitLine(width, line)
		if i == selected {
			line = Bold(out, Green(out, line))
		}
		writeHomeLine(out, line)
	}
	writeHomeLine(out, "")

	moreLabel := "M More"
	if showMore {
		moreLabel = "M Less"
	}
	footer := fmt.Sprintf("↑↓  |  Enter  |  %s  |  1-9 Jump  |  Q Quit", moreLabel)
	writeHomeLine(out, Gray(out, fitLine(width, footer)))
}

func readHomeKey(reader *bufio.Reader) (string, string, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return keyUnknown, "", err
	}

	switch b {
	case '\r', '\n':
		return keyEnter, "", nil
	case 'k', 'K':
		return keyUp, "", nil
	case 'j', 'J':
		return keyDown, "", nil
	case 'm', 'M':
		return keyMore, "", nil
	case 'q', 'Q', 3: // q/Q/Ctrl-C
		return keyQuit, "", nil
	case 27: // Escape / arrow keys
		b2, err := reader.ReadByte()
		if err != nil {
			return keyQuit, "", nil
		}
		if b2 != '[' {
			return keyUnknown, "", nil
		}
		b3, err := reader.ReadByte()
		if err != nil {
			return keyUnknown, "", nil
		}
		switch b3 {
		case 'A':
			return keyUp, "", nil
		case 'B':
			return keyDown, "", nil
		default:
			return keyUnknown, "", nil
		}
	default:
		if b >= '1' && b <= '9' {
			return keyUnknown, string(b), nil
		}
		return keyUnknown, "", nil
	}
}

func actionFromDigit(digit string, items []homeItem) (HomeAction, bool) {
	if digit == "" {
		return "", false
	}
	n, err := strconv.Atoi(digit)
	if err != nil {
		return "", false
	}
	for _, item := range items {
		if item.Number == n {
			return item.Action, true
		}
	}
	return "", false
}

func terminalWidth(fd int) int {
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 100
	}
	return width
}

func fitLine(width int, text string) string {
	if width <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

// writeHomeLine writes CRLF so line starts stay correct while terminal is in raw mode.
func writeHomeLine(out io.Writer, text string) {
	fmt.Fprintf(out, "%s\r\n", text)
}
