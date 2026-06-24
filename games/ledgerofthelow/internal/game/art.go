package game

import (
	"fmt"
	"strings"

	"interdoor.net/interdoor/internal/engine/term"
)

// Unicode single-line box-drawing characters (1-cell wide on all standard terminals).
const (
	boxH  = "─" // ─
	boxV  = "│" // │
	boxTL = "┌" // ┌
	boxTR = "┐" // ┐
	boxBL = "└" // └
	boxBR = "┘" // ┘
)

// Status bar icons (single-width Unicode, broadly supported).
const (
	iconHP    = "♥" // ♥
	iconExp   = "◉" // ◉
	iconAtk   = "✕" // ✕
	iconDebt  = "◊" // ◊
	iconDepth = "≡" // ≡
)

// visLen returns the visible column count of s, ignoring ANSI CSI escape sequences.
// Only handles ESC [ ... <final-byte> form, which is all we generate.
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
				i++ // skip the final byte
			}
		} else {
			out++
			i++
		}
	}
	return out
}

// isFinalByte reports whether r is a CSI final byte (0x40–0x7E).
func isFinalByte(r rune) bool { return r >= 0x40 && r <= 0x7e }

// padRight pads s to visual width w with trailing spaces.
func padRight(s string, w int) string {
	v := visLen(s)
	if v >= w {
		return s
	}
	return s + strings.Repeat(" ", w-v)
}

// hRule returns a full-width colored horizontal rule of Unicode box-H characters.
func hRule(color string, w int) string {
	return color + strings.Repeat(boxH, w) + term.Reset()
}

// boxTop returns a colored top border, w visual chars wide.
func boxTop(color string, w int) string {
	return color + boxTL + strings.Repeat(boxH, w-2) + boxTR + term.Reset()
}

// boxBot returns a colored bottom border, w visual chars wide.
func boxBot(color string, w int) string {
	return color + boxBL + strings.Repeat(boxH, w-2) + boxBR + term.Reset()
}

// boxLine returns a content row: left-border, content padded to inner width, right-border.
func boxLine(borderColor, content string, w int) string {
	padded := padRight(content, w-2)
	return borderColor + boxV + term.Reset() + padded + borderColor + boxV + term.Reset()
}

// menuItem formats one menu entry for inside a box:
//
//	[key]  name                   - description
func menuItem(key, name, desc string) string {
	k := term.Bright(term.Green) + "[" + key + "]" + term.Reset()
	d := term.FG(term.Cyan) + desc + term.Reset()
	return "   " + k + "  " + padRight(name, 22) + " - " + d
}

// --- title art (used at login) ---

var titleArt = buildTitle()

func buildTitle() string {
	const w = 60
	bar := "  +" + strings.Repeat("=", w) + "+"
	blank := "  |" + strings.Repeat(" ", w) + "|"
	center := func(s string) string {
		if len(s) >= w {
			return "  |" + s[:w] + "|"
		}
		left := (w - len(s)) / 2
		return "  |" + strings.Repeat(" ", left) + s + strings.Repeat(" ", w-len(s)-left) + "|"
	}
	return strings.Join([]string{
		bar, blank,
		center("L E D G E R   O F   T H E   L O W"),
		blank, center(". . .   the Old Bargain   . . ."),
		blank, bar,
	}, "\n")
}

// Banner is the welcome screen rendered before login.
func (g *Ledger) Banner() string {
	y, dim, r := term.Bright(term.Yellow), term.FG(term.Cyan), term.Reset()
	return y + titleArt + r + "\n\n" +
		dim + "    You don't remember arriving. No one ever does." + r + "\n" +
		dim + "    Beneath the city of Dornhaven, the Low keeps its accounts -" + r + "\n" +
		dim + "    and now, it seems, it keeps you." + r
}

// --- combat HP bar ---

// hpBar renders a width-char colored ASCII bar: [####------]
func hpBar(hp, max, width int) string {
	if max <= 0 {
		max = 1
	}
	pct := hp * 100 / max
	filled := hp * width / max
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	var color string
	switch {
	case pct > 50:
		color = term.Bright(term.Green)
	case pct > 25:
		color = term.Bright(term.Yellow)
	default:
		color = term.Bright(term.Red)
	}
	return color + "[" + bar + "]" + term.Reset()
}

// fmtHP formats "  Label:          [bar] hp/max" for the combat screen.
func fmtHP(label string, hp, max int) string {
	return fmt.Sprintf("  %-16s %s  %d/%d", label+":", hpBar(hp, max, 14), hp, max)
}

// --- status bar column content ---

// statusCols returns the two inner content strings (labelRow, valueRow) for the
// magenta status box at the given total box width w.
// Each returned string is exactly w-2 visual chars wide (padded with spaces).
func statusCols(
	hp, maxHP int,
	actions, maxActions int,
	attacks, maxAttacks int,
	debt int,
	level int,
	depthRecord int,
	w int,
) (labelRow, valueRow string) {
	r := term.Bright(term.Red)
	y := term.Bright(term.Yellow)
	mg := term.FG(term.Magenta)
	bl := term.FG(term.Blue)
	cy := term.FG(term.Cyan)
	rs := term.Reset()
	sep := mg + boxV + rs

	// 6 columns + 5 internal │ separators must fill inner width (w-2).
	// Column widths (visual, each includes 1 leading space): 12+15+12+11+8+rem.
	inner := w - 2 // e.g. 78 for w=80
	c1, c2, c3, c4, c5 := 12, 15, 12, 11, 8
	c6 := inner - c1 - c2 - c3 - c4 - c5 - 5 // minus 5 internal separators

	l1 := padRight(" "+r+iconHP+" HP"+rs, c1)
	l2 := padRight(" "+y+iconExp+" Expeditions"+rs, c2)
	l3 := padRight(" "+r+iconAtk+" Attacks"+rs, c3)
	l4 := padRight(" "+mg+iconDebt+" Debt"+rs, c4)
	l5 := padRight(" "+bl+"Lv"+rs, c5)
	l6 := padRight(" "+cy+iconDepth+" Depth"+rs, c6)

	v1 := padRight(" "+r+fmt.Sprintf("%d/%d", hp, maxHP)+rs, c1)
	v2 := padRight(" "+y+fmt.Sprintf("%d/%d", actions, maxActions)+rs, c2)
	v3 := padRight(" "+r+fmt.Sprintf("%d/%d", attacks, maxAttacks)+rs, c3)
	v4 := padRight(" "+mg+fmt.Sprintf("%d", debt)+rs, c4)
	v5 := padRight(" "+bl+fmt.Sprintf("%d", level)+rs, c5)
	v6 := padRight(" "+cy+fmt.Sprintf("%d", depthRecord)+rs, c6)

	labelRow = strings.Join([]string{l1, l2, l3, l4, l5, l6}, sep)
	valueRow = strings.Join([]string{v1, v2, v3, v4, v5, v6}, sep)
	return
}

// --- text content ---

var arrivalLines = []string{
	"A man with a lantern is crouched nearby, watching you wake. He does not",
	"seem surprised to see you. Few things surprise a lamplighter.",
	"",
	`"Welcome down," he says. "You're in the Low now -- the old dark under`,
	`Dornhaven, where the city keeps what it would rather forget. You weren't`,
	`banished. You were balanced out. The arithmetic needed you gone, so gone`,
	`you went. It happens."`,
	"",
	`"Here's the shape of it. The LANTERNMARKET is where the living's done --`,
	`trade, rest, and the way down into the WARRENS, where you'll do your`,
	`fighting and your finding. Everything runs on goods, favors, and debts.`,
	`There's no coin down here, and there's no forgetting. The Ledger sees to`,
	`both."`,
	"",
	`"Mind the line along the bottom of your sight: your wounds, your`,
	`expeditions left for the day, and what you owe. Spend the day. Come back`,
	`tomorrow. The Low will still be here. The Low is always still here."`,
}

var instructionLines = []string{
	"HOW THE LOW WORKS",
	"",
	"  Each day you may make a number of EXPEDITIONS into the Warrens -- that's",
	"  your fighting and scavenging. Town doings -- trading, resting, reading the",
	"  news, taking stock -- cost you nothing but time, of which the dead have",
	"  plenty and you have less.",
	"",
	"  THE WARRENS.  Go shallow for a gentler time, deep for worse things and",
	"  better salvage. Win and you'll pocket goods. Lose badly and you'll wake",
	"  in the Threshold, lighter than you were and owing the same.",
	"",
	"  GOODS, FAVORS & DEBTS.  Scavenge sells. When your scavenge won't cover a",
	"  purchase, the difference goes on your tab -- a debt. The Ledger remembers",
	"  every one. Owe too much and folk grow less friendly.",
	"",
	"  MENDING.  REST patches you up a little, and costs nothing. The BONESETTER",
	"  will mend you fully -- for goods, or on the tab. You wake whole each day.",
	"",
	"  OTHER WANDERERS.  You are not the only one down here. You may catch one",
	"  sleeping. One may catch you. Banked goods are safe; what you carry is not.",
}
