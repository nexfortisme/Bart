package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/nexfortisme/bart/internal/bot"
	"golang.org/x/sys/unix"
)

type Menu struct {
	bot       *bot.Bot
	log       *LogManager
	out       *os.File // real terminal stdout, bypasses log capture
	interrupt chan os.Signal
}

func NewMenu(b *bot.Bot, lm *LogManager, interrupt chan os.Signal) *Menu {
	return &Menu{bot: b, log: lm, out: lm.RealOut, interrupt: interrupt}
}

// showStartup waits briefly for goroutines to finish starting, then prints
// whatever landed in the log buffer so the user sees "Bot started" etc.
// before the menu takes over the screen.
func (m *Menu) showStartup() {
	time.Sleep(250 * time.Millisecond)
	for _, line := range m.log.Lines() {
		fmt.Fprintln(m.out, line)
	}
	fmt.Fprint(m.out, "\nPress any key to open menu...")
	m.readMenuKey()
}

func (m *Menu) clear() {
	fmt.Fprint(m.out, "\033[2J\033[H") // erase screen, move cursor to top-left
}

func (m *Menu) Run() {
	m.showStartup()
	for {
		m.clear()
		m.printMenu()

		ch, ok := m.readMenuKey()
		if !ok {
			return
		}

		switch unicode.ToLower(ch) {
		case 'w':
			m.clear()
			m.watchLogs()
		case 'c':
			m.clear()
			m.chat()
		case 'x':
			m.interrupt <- os.Interrupt
			return
		default:
			// Unknown key — just redraw the menu
		}
	}
}

func (m *Menu) printMenu() {
	fmt.Fprintln(m.out, "--- BART ---")
	fmt.Fprintln(m.out, "(W) Watch logs")
	fmt.Fprintln(m.out, "(C) Chat")
	fmt.Fprintln(m.out, "(X) Exit")
	fmt.Fprint(m.out, "\n> ")
}

// readMenuKey reads a single keypress in raw terminal mode.
// Returns the rune and true, or (0, false) if the interrupt signal fires.
func (m *Menu) readMenuKey() (rune, bool) {
	fd := int(os.Stdin.Fd())

	oldState, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err == nil {
		newState := *oldState
		newState.Lflag &^= unix.ECHO | unix.ICANON
		newState.Cc[unix.VMIN] = 1
		newState.Cc[unix.VTIME] = 0
		_ = unix.IoctlSetTermios(fd, unix.TIOCSETA, &newState)
		defer unix.IoctlSetTermios(fd, unix.TIOCSETA, oldState)
	}

	keyCh := make(chan rune, 1)
	go func() {
		buf := make([]byte, 1)
		n, err := os.Stdin.Read(buf)
		if err == nil && n > 0 {
			keyCh <- rune(buf[0])
		} else {
			close(keyCh)
		}
	}()

	select {
	case ch, ok := <-keyCh:
		if !ok {
			return 0, false
		}
		return ch, true
	case <-m.interrupt:
		return 0, false
	}
}

func (m *Menu) watchLogs() {
	fmt.Fprintln(m.out, "--- Watching logs (press Enter to return to menu) ---")
	fmt.Fprintln(m.out)

	done := make(chan struct{})
	go m.log.Watch(done)

	bufio.NewReader(os.Stdin).ReadString('\n')
	close(done)
}

func (m *Menu) chat() {
	fmt.Fprintln(m.out, "--- Chat (type 'exit' to return to menu) ---")
	fmt.Fprintln(m.out)
	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Fprint(m.out, "You: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			break
		}
		if input == "" {
			continue
		}

		fmt.Fprintln(m.out, "Thinking...")
		response, err := m.bot.Chat(ctx, input)
		if err != nil {
			fmt.Fprintf(m.out, "Error: %v\n", err)
			continue
		}
		fmt.Fprintf(m.out, "\nBART: %s\n\n", response)
	}
}
