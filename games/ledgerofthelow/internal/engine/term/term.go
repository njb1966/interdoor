// Package term provides the InterDOOR terminal layer: raw 80x24 ANSI rendering
// and keyboard input over an SSH channel. It is engine-generic — it knows nothing
// about any particular game.
package term

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"
)

// The fixed playfield. NETWORK_REQUIREMENTS Req 8: 80x24, base-16 ANSI.
const (
	Width  = 80
	Height = 24
)

const (
	esc       = "\x1b["
	reset     = esc + "0m"
	clearHome = esc + "2J" + esc + "H"
)

// Base-16 ANSI foreground colors.
const (
	Black = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// FG returns the escape for a normal foreground color.
func FG(c int) string { return fmt.Sprintf("%s3%dm", esc, c) }

// Bright returns the escape for a bright foreground color.
func Bright(c int) string { return fmt.Sprintf("%s9%dm", esc, c) }

// Reset returns the SGR reset escape.
func Reset() string { return reset }

// Terminal wraps a client connection (typically an SSH channel). The client is
// expected to be in raw mode (no local echo); we echo deliberately where needed.
type Terminal struct {
	r       *bufio.Reader
	w       io.Writer
	lastAct atomic.Int64 // unix-nano of last input byte, for idle detection
}

// New builds a Terminal over the given reader/writer.
func New(r io.Reader, w io.Writer) *Terminal {
	t := &Terminal{r: bufio.NewReader(r), w: w}
	t.touch()
	return t
}

func (t *Terminal) touch() { t.lastAct.Store(time.Now().UnixNano()) }

// LastActivity reports when the client last sent input — used for idle timeouts.
func (t *Terminal) LastActivity() time.Time { return time.Unix(0, t.lastAct.Load()) }

// Write sends a raw string to the client.
func (t *Terminal) Write(s string) { _, _ = io.WriteString(t.w, s) }

// Clear clears the screen and homes the cursor.
func (t *Terminal) Clear() { t.Write(clearHome) }

// ReadKey returns a single keypress. CR and LF both normalize to '\n'.
func (t *Terminal) ReadKey() (rune, error) {
	b, err := t.r.ReadByte()
	if err != nil {
		return 0, err
	}
	t.touch()
	if b == '\r' || b == '\n' {
		return '\n', nil
	}
	return rune(b), nil
}

// ReadLine reads a line of printable input, handling backspace. If echo is false
// (passwords), input is masked with '*'. The client is assumed to be in raw mode,
// so the server is responsible for all echo.
func (t *Terminal) ReadLine(echo bool) (string, error) {
	var sb strings.Builder
	for {
		b, err := t.r.ReadByte()
		if err != nil {
			return "", err
		}
		t.touch()
		switch {
		case b == '\r' || b == '\n':
			t.Write("\r\n")
			return sb.String(), nil
		case b == 0x7f || b == 0x08: // DEL / BS
			if sb.Len() > 0 {
				s := sb.String()[:sb.Len()-1]
				sb.Reset()
				sb.WriteString(s)
				t.Write("\b \b")
			}
		case b >= 0x20 && b < 0x7f:
			sb.WriteByte(b)
			if echo {
				t.Write(string(rune(b)))
			} else {
				t.Write("*")
			}
		}
	}
}

// Frame accumulates body lines, an optional pre-status line, and a status line,
// then renders one 80x24 screen.
type Frame struct {
	lines  []string
	pre    string // row 23 when set; body is capped at row 22
	status string // row 24
}

// NewFrame starts an empty frame.
func NewFrame() *Frame { return &Frame{} }

// Line appends a body line, word-wrapping plain text that exceeds the playfield
// width. Continuation lines keep the original line's leading indent.
func (f *Frame) Line(s string) *Frame {
	f.lines = append(f.lines, wrapLine(s, Width)...)
	return f
}

// Blank appends an empty line.
func (f *Frame) Blank() *Frame { return f.Line("") }

// Pre sets the row-23 line (just above the status). When set, body is capped at
// 22 rows so the pre line always renders at exactly row 23.
func (f *Frame) Pre(s string) *Frame { f.pre = s; return f }

// Status sets the row-24 status line.
func (f *Frame) Status(s string) *Frame { f.status = s; return f }

// Render draws the frame: clear, body padded to row 22 or 23, optional pre on
// row 23, status on row 24.
func (f *Frame) Render(t *Terminal) {
	var sb strings.Builder
	sb.WriteString(clearHome)
	maxBody := Height - 1
	if f.pre != "" {
		maxBody = Height - 2
	}
	n := 0
	for _, ln := range f.lines {
		if n >= maxBody {
			break
		}
		sb.WriteString(clip(ln))
		sb.WriteString("\r\n")
		n++
	}
	for n < maxBody {
		sb.WriteString("\r\n")
		n++
	}
	if f.pre != "" {
		sb.WriteString(clip(f.pre))
		sb.WriteString("\r\n")
	}
	sb.WriteString(clip(f.status))
	t.Write(sb.String())
}

// wrapLine word-wraps a plain line to width, preserving the leading indent on
// continuation lines. Lines with escape sequences or already within width are
// returned unchanged.
func wrapLine(s string, width int) []string {
	if strings.ContainsRune(s, '\x1b') || len([]rune(s)) <= width {
		return []string{s}
	}
	indent := s[:len(s)-len(strings.TrimLeft(s, " "))]
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}
	var lines []string
	cur := indent + words[0]
	for _, w := range words[1:] {
		if len([]rune(cur))+1+len([]rune(w)) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = indent + w
		}
	}
	return append(lines, cur)
}

// clip truncates a plain line to the playfield width. Lines containing escape
// sequences are passed through untouched (the caller manages their width).
func clip(s string) string {
	if strings.ContainsRune(s, '\x1b') {
		return s
	}
	if r := []rune(s); len(r) > Width {
		return string(r[:Width])
	}
	return s
}
