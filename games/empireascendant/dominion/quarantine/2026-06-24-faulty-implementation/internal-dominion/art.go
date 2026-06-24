package dominion

import (
	"fmt"
	"strings"

	"interdoor.net/interdoor/internal/engine/term"
)

const screenW = 80

const (
	boxH  = "─"
	boxV  = "│"
	boxTL = "┌"
	boxTR = "┐"
	boxBL = "└"
	boxBR = "┘"
)

func visLen(s string) int {
	out := 0
	i := 0
	rs := []rune(s)
	for i < len(rs) {
		if rs[i] == '\x1b' && i+1 < len(rs) && rs[i+1] == '[' {
			i += 2
			for i < len(rs) && !isFinalByte(rs[i]) {
				i++
			}
			if i < len(rs) {
				i++
			}
		} else {
			out++
			i++
		}
	}
	return out
}

func isFinalByte(r rune) bool { return r >= 0x40 && r <= 0x7e }

func padRight(s string, w int) string {
	v := visLen(s)
	if v >= w {
		return s
	}
	return s + strings.Repeat(" ", w-v)
}

func hRule(color string, w int) string {
	return color + strings.Repeat(boxH, w) + term.Reset()
}

func boxTop(color string, w int) string {
	return color + boxTL + strings.Repeat(boxH, w-2) + boxTR + term.Reset()
}

func boxBot(color string, w int) string {
	return color + boxBL + strings.Repeat(boxH, w-2) + boxBR + term.Reset()
}

func boxLine(borderColor, content string, w int) string {
	padded := padRight(content, w-2)
	return borderColor + boxV + term.Reset() + padded + borderColor + boxV + term.Reset()
}

func menuItem(key, name, desc string) string {
	k := term.Bright(term.Green) + "[" + key + "]" + term.Reset()
	d := term.FG(term.Cyan) + desc + term.Reset()
	return "   " + k + "  " + padRight(name, 22) + " - " + d
}

func promptFooter(f *term.Frame, name string) {
	rule := term.FG(term.Blue)
	text := term.Bright(term.White)
	rs := term.Reset()
	f.Pre(rule + strings.Repeat(boxH, screenW) + rs)
	f.Status(text + "Command, " + name + "?: " + rs)
}

func pauseStatus(f *term.Frame) {
	c, rs := term.FG(term.Blue), term.Reset()
	const msg = " press any key "
	left := (screenW - len(msg)) / 2
	right := screenW - len(msg) - left
	f.Status(c + strings.Repeat(boxH, left) + msg + strings.Repeat(boxH, right) + rs)
}

func titleLine(title, subtitle string) string {
	return term.Bright(term.Yellow) + title + term.Reset() +
		"    " + term.FG(term.Cyan) + subtitle + term.Reset()
}

var bannerArt = buildBanner()

func buildBanner() string {
	cy := term.FG(term.Cyan)
	yc := term.Bright(term.Yellow)
	gr := term.Bright(term.Green)
	ma := term.FG(term.Magenta)
	rs := term.Reset()

	const inner = 74
	top := cy + "  ╔" + strings.Repeat("═", inner) + "╗" + rs
	bot := cy + "  ╚" + strings.Repeat("═", inner) + "╝" + rs
	row := func(content string) string {
		return cy + "  ║" + rs + padRight(content, inner) + cy + "║" + rs
	}
	dot := ma + "✦" + rs

	return strings.Join([]string{
		top,
		row(""),
		row("    " + yc + "★  E M P I R E   A S C E N D A N T  ★" + rs),
		row("    " + cy + strings.Repeat("─", 41) + rs),
		row(""),
		row("    " + gr + "A space empire strategy door game" + rs),
		row("    for the " + cy + "InterDOOR BBS Network" + rs),
		row(""),
		row("    " + dot + " Build your world       " + dot + " Research technologies"),
		row("    " + dot + " Command your forces    " + dot + " Strike rival empires"),
		row("    " + dot + " Warp across the galaxy to reach distant nodes"),
		row(""),
		bot,
	}, "\n")
}

func noticeWin(s string) string { return term.Bright(term.Green) + ">> " + term.Reset() + s }
func noticeBad(s string) string { return term.Bright(term.Red) + "!! " + term.Reset() + s }

func fmtCredits(n int) string { return fmt.Sprintf("%d cr", n) }
