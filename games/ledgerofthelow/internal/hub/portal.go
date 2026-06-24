package hub

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Portal is the InterDOOR hub SSH portal: a public-access ANSI terminal
// showing the live network directory. No authentication required.
type Portal struct {
	Store Store
}

// ListenAndServe starts the SSH portal on addr, loading the host private key
// from hostkeyPath. Blocks until the listener fails.
func (p *Portal) ListenAndServe(addr, hostkeyPath string) error {
	key, err := os.ReadFile(hostkeyPath)
	if err != nil {
		return fmt.Errorf("read host key %s: %w", hostkeyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return fmt.Errorf("parse host key: %w", err)
	}
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	for {
		nConn, err := ln.Accept()
		if err != nil {
			return err
		}
		go p.handleConn(nConn, cfg)
	}
}

func (p *Portal) handleConn(nConn net.Conn, cfg *ssh.ServerConfig) {
	defer nConn.Close()
	conn, chans, reqs, err := ssh.NewServerConn(nConn, cfg)
	if err != nil {
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		ch, chReqs, err := newCh.Accept()
		if err != nil {
			continue
		}
		go p.serveSession(ch, chReqs)
	}
}

func (p *Portal) serveSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	ready := make(chan struct{}, 1)
	go func() {
		for req := range reqs {
			switch req.Type {
			case "pty-req", "window-change", "env":
				_ = req.Reply(true, nil)
			case "shell":
				_ = req.Reply(true, nil)
				select {
				case ready <- struct{}{}:
				default:
				}
			default:
				_ = req.Reply(false, nil)
			}
		}
	}()
	<-ready
	p.runPortal(ch)
}

const portalClear = "\x1b[2J\x1b[H"

func portalWrite(w io.Writer, s string) { _, _ = io.WriteString(w, s) }

func (p *Portal) runPortal(ch ssh.Channel) {
	buf := make([]byte, 8)
	for {
		nodes, _ := p.Store.GetDirectory()
		eventCount, _ := p.Store.EventCount()
		online := portalOnline(nodes)
		portalRenderMain(ch, online, eventCount)

		n, err := ch.Read(buf)
		if err != nil || n == 0 {
			return
		}
		switch buf[0] {
		case 'q', 'Q', 3, 4: // Q, Ctrl-C, Ctrl-D
			portalWrite(ch, portalClear+"\r\n  Goodbye.\r\n\r\n")
			return
		case 'r', 'R':
			// re-render on next loop
		default:
			if buf[0] >= '1' && buf[0] <= '9' {
				idx := int(buf[0] - '1')
				if idx < len(online) {
					portalShowDetail(ch, online[idx])
					ch.Read(buf) //nolint — wait for any key; error means disconnect
				}
			}
		}
	}
}

func portalOnline(nodes []Node) []Node {
	var out []Node
	for _, n := range nodes {
		if time.Since(n.LastHeartbeat) < 3*time.Minute {
			out = append(out, n)
		}
	}
	return out
}

// portalRenderMain draws the full 24-row hub directory screen.
//
// Layout (1-indexed rows, 80-col terminal):
//
//	Rows  1-9:  logo
//	Row  10:    blank
//	Row  11:    network status
//	Row  12:    blank
//	Row  13:    column headers
//	Row  14:    separator
//	Rows 15+:   numbered node entries (max 9; shown ≥ 1)
//	…blank fill to row 22…
//	Row  23:    nav hint
//	Row  24:    "  Select: " prompt (no trailing newline)
func portalRenderMain(w io.Writer, online []Node, eventCount int64) {
	var sb strings.Builder
	sb.WriteString(portalClear)

	// Logo — 9 lines, rows 1-9
	for _, line := range logoLines {
		sb.WriteString(line + "\r\n")
	}

	// Row 10: blank
	sb.WriteString("\r\n")

	// Row 11: network status
	nodeWord := "nodes"
	if len(online) == 1 {
		nodeWord = "node"
	}
	sb.WriteString(fmt.Sprintf("  %s%d %s online%s  %s·  %d events indexed%s\r\n",
		ansiBG, len(online), nodeWord, ansiReset,
		ansiDG, eventCount, ansiReset,
	))

	// Row 12: blank
	sb.WriteString("\r\n")

	// Row 13: column headers
	sb.WriteString(fmt.Sprintf(ansiDG+"  #    %-16s  %-21s  %-23s  PLAYERS"+ansiReset+"\r\n", "NODE", "GAME", "ADDRESS"))

	// Row 14: separator
	sb.WriteString(fmt.Sprintf(ansiBL+"  ───  %-16s  %-21s  %-23s  ───────"+ansiReset+"\r\n",
		"────────────────", "─────────────────────", "───────────────────────"))

	// Rows 15+: node entries (cursor is now at row 15)
	shown := 0
	for i, n := range online {
		if i >= 9 {
			break
		}
		sb.WriteString(fmt.Sprintf("  %s[%d]%s  %-16s  %-21s  %-23s  %d\r\n",
			ansiBW, i+1, ansiReset,
			portalTrunc(n.NodeID, 16),
			portalTrunc(n.DisplayGameTitle(), 21),
			portalTrunc(portalAddr(n.AdvertiseAddr), 23),
			n.PlayerCount,
		))
		shown++
	}
	if shown == 0 {
		sb.WriteString(ansiDG + "  No nodes are currently online." + ansiReset + "\r\n")
		shown = 1
	}

	// Fill blank rows so nav hint always lands on row 23.
	// After logo(9) + blank(1) + status(1) + blank(1) + header(1) + sep(1) + shown
	// the cursor is at row 14+shown+1 = 15+shown (1-indexed).
	for i := 15 + shown; i < 23; i++ {
		sb.WriteString("\r\n")
	}

	// Row 23: nav hint
	sb.WriteString(ansiDG + "  [1-9] Connect  ·  [R] Refresh  ·  [Q] Quit" + ansiReset + "\r\n")

	// Row 24: prompt — no trailing newline; cursor waits here
	sb.WriteString("  Select: ")

	portalWrite(w, sb.String())
}

func portalShowDetail(w io.Writer, n Node) {
	var sb strings.Builder
	sb.WriteString(portalClear)

	for _, line := range logoLines {
		sb.WriteString(line + "\r\n")
	}
	sb.WriteString("\r\n")

	sb.WriteString(fmt.Sprintf("  %sNode:%s     %s%s%s\r\n", ansiDG, ansiReset, ansiBW, n.NodeID, ansiReset))
	sb.WriteString(fmt.Sprintf("  %sGame:%s     %s%s%s\r\n", ansiDG, ansiReset, ansiDY, n.DisplayGameTitle(), ansiReset))
	sb.WriteString(fmt.Sprintf("  %sVersion:%s  %s%s%s\r\n", ansiDG, ansiReset, ansiDG, n.GameVersion, ansiReset))
	sb.WriteString(fmt.Sprintf("  %sAddress:%s  %s%s%s\r\n", ansiDG, ansiReset, ansiDC, portalAddr(n.AdvertiseAddr), ansiReset))
	sb.WriteString(fmt.Sprintf("  %sPlayers:%s  %d online\r\n", ansiDG, ansiReset, n.PlayerCount))
	sb.WriteString("\r\n")
	sb.WriteString("  This hub lists nodes; it does not launch game sessions.\r\n")
	sb.WriteString("  Open a new terminal and run:\r\n")
	sb.WriteString(fmt.Sprintf("  %s%s%s\r\n", ansiBY, portalSSHCmd(n.AdvertiseAddr), ansiReset))
	sb.WriteString("\r\n")
	sb.WriteString(ansiDG + "  Press any key to return to the directory..." + ansiReset + "\r\n")

	portalWrite(w, sb.String())
}

// portalAddr strips the optional "ssh://" scheme nodes may register with.
func portalAddr(addr string) string {
	return strings.TrimPrefix(addr, "ssh://")
}

func portalSSHCmd(addr string) string {
	plain := portalAddr(addr)
	host, port, err := net.SplitHostPort(plain)
	if err != nil {
		return "ssh " + plain
	}
	if port == "22" {
		return "ssh " + host
	}
	return fmt.Sprintf("ssh -p %s %s", port, host)
}

func portalTrunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
