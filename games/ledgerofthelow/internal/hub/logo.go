package hub

// ANSI color constants shared across the portal.
const (
	ansiReset = "\x1b[0m"
	ansiBY    = "\x1b[1;33m" // bright yellow  — logo body
	ansiDY    = "\x1b[0;33m" // dim yellow     — logo underline row
	ansiBW    = "\x1b[1;37m" // bright white   — subtitle
	ansiDC    = "\x1b[0;36m" // dim cyan       — hub address / node address
	ansiBL    = "\x1b[0;34m" // dim blue       — rules / separators
	ansiBG    = "\x1b[1;32m" // bright green   — online count
	ansiDG    = "\x1b[0;37m" // dim grey       — secondary labels / nav hint
)

// logoLines is the 9-line ANSI logo for the InterDOOR hub portal.
// Every line fits within 80 visual columns (block chars are single-width).
var logoLines = []string{
	ansiBL + "────────────────────────────────────────────────────────────────────────────────" + ansiReset,
	"",
	ansiDY + `              ─▄─  ▄──▄  ─▄─  ▄──▄  ▄──▄  ▄──▄   ▄──▄  ▄──▄  ▄──▄` + ansiReset,
	ansiBG + `               █   █  █   █   █─    █─▄▀  █  ▐▌  █  █  █  █  █─▄▀` + ansiReset,
	ansiBG + `              ─▀─  ▀  ▀   ▀   ▀──▀  ▀  ▀  ▀──▀   ▀──▀  ▀──▀  ▀  ▀` + ansiReset,
	"",
	"",
	ansiBW + "                   Federated Terminal Game Network" + ansiDC + "   hub.interdoor.net" + ansiReset,
	ansiBL + "────────────────────────────────────────────────────────────────────────────────" + ansiReset,
}
