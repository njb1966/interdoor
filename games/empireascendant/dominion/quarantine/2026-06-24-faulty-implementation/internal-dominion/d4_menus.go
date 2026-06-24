package dominion

import (
	"fmt"
	"strings"
	"time"

	"interdoor.net/interdoor/internal/engine"
	"interdoor.net/interdoor/internal/engine/term"
)

// ---- wanderers screen ----

func (g *Dominion) wanderersScreen(ctx *engine.Context) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)
	cy := term.FG(term.Cyan)
	rs := term.Reset()

	players, _ := ctx.Store.Players(ctx.Player.GlobalID)
	db := ctx.Store.DB()

	f := term.NewFrame()
	f.Line(titleLine("WANDERERS", "empires known to this node"))
	f.Line(hRule(yc, screenW))
	f.Line(boxTop(gc, screenW))

	if len(players) == 0 {
		f.Line(boxLine(gc, "  No other empires are known to this node.", screenW))
	} else {
		f.Line(boxLine(gc, fmt.Sprintf("  %s%-20s %-20s %-14s %s%s",
			cy, "PLAYER", "EMPIRE", "NODE", "LAST SEEN", rs), screenW))
		f.Line(boxLine(gc, hRule(cy, 74), screenW))
		for _, p := range players {
			var empName string
			_ = db.QueryRow(`SELECT empire_name FROM empires WHERE global_id=?`, p.GlobalID).Scan(&empName)
			if empName == "" {
				empName = "(unknown)"
			}
			nodeLabel := p.HomeNode
			if p.HomeNode == ctx.NodeID {
				nodeLabel = term.Bright(term.Green) + "local" + rs
			}
			f.Line(boxLine(gc, fmt.Sprintf("  %-20s %-20s %-14s %s",
				truncate(p.Name, 20), truncate(empName, 20), nodeLabel, formatAge(p.LastSeen)), screenW))
		}
	}
	f.Line(boxBot(gc, screenW))
	pauseStatus(f)
	f.Render(ctx.Term)
	_, _ = ctx.Term.ReadKey()
}

// ---- warp screen ----

func (g *Dominion) warpScreen(ctx *engine.Context, e *empireState) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)

	if ctx.Travel == nil {
		g.infoPage(ctx, "WARP TO GALAXY", []string{
			"", "  This node is not connected to the InterDOOR network.", "",
		})
		return
	}
	if !hasTech(ctx.Store.DB(), e.GlobalID, TechHyperdrive) {
		g.infoPage(ctx, "WARP TO GALAXY", []string{
			"", "  Hyperdrive technology required. Research it in the Tech Tree.", "",
		})
		return
	}
	// Visiting player: offer return home.
	if ctx.Player.HomeNode != ctx.NodeID {
		g.departScreen(ctx)
		return
	}

	others, _ := ctx.Store.Players(ctx.Player.GlobalID)
	seen := map[string]bool{}
	var nodes []string
	for _, p := range others {
		if p.HomeNode != ctx.NodeID && !seen[p.HomeNode] {
			seen[p.HomeNode] = true
			nodes = append(nodes, p.HomeNode)
		}
	}
	if len(nodes) == 0 {
		g.infoPage(ctx, "WARP TO GALAXY", []string{
			"", "  No other nodes known yet. Wanderers from distant nodes must appear first.", "",
		})
		return
	}

	notice := "Hyperdrive online. Select destination. Costs 2 turns."
	for {
		limit := len(nodes)
		if limit > 9 {
			limit = 9
		}
		f := term.NewFrame()
		f.Line(titleLine("WARP TO GALAXY", "cross-node travel"))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, "  "+notice, screenW))
		f.Line(boxBot(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, "  Known nodes:", screenW))
		f.Line(boxLine(gc, "", screenW))
		for i := 0; i < limit; i++ {
			f.Line(boxLine(gc, fmt.Sprintf("    [%d]  %s", i+1, nodes[i]), screenW))
		}
		f.Line(boxLine(gc, "", screenW))
		f.Line(boxLine(gc, "    [1-9] Warp     [Q] Stay", screenW))
		f.Line(boxBot(gc, screenW))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch {
		case key >= '1' && key <= '9':
			idx := int(key - '1')
			if idx >= limit {
				continue
			}
			turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
			if turn.Actions < 2 {
				notice = noticeBad("Warp costs 2 turns. Not enough turns remaining.")
				continue
			}
			if err := ctx.Travel(ctx.Player.GlobalID, nodes[idx]); err != nil {
				notice = noticeBad("Warp failed: " + err.Error())
			} else {
				_ = ctx.Store.SpendActions(ctx.Player.GlobalID, 2)
				g.showDeparting(ctx, nodes[idx])
				return
			}
		case key == 'q' || key == 'Q':
			return
		}
	}
}

func (g *Dominion) departScreen(ctx *engine.Context) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)
	notice := fmt.Sprintf("Visiting %s. Your empire is homed at %s.", ctx.NodeID, ctx.Player.HomeNode)
	for {
		f := term.NewFrame()
		f.Line(titleLine("VISITING "+strings.ToUpper(ctx.NodeID), "far from home"))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, "  "+notice, screenW))
		f.Line(boxBot(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, menuItem("R", "Return to "+ctx.Player.HomeNode, ""), screenW))
		f.Line(boxLine(gc, menuItem("Q", "Stay here for now", ""), screenW))
		f.Line(boxBot(gc, screenW))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch key {
		case 'r', 'R':
			if err := ctx.Travel(ctx.Player.GlobalID, ctx.Player.HomeNode); err != nil {
				notice = noticeBad("Return failed: " + err.Error())
			} else {
				g.showDeparting(ctx, ctx.Player.HomeNode)
				return
			}
		case 'q', 'Q':
			return
		}
	}
}

func (g *Dominion) showDeparting(ctx *engine.Context, dest string) {
	g.infoPage(ctx, "WARPING", []string{
		"", "  Your empire's beacon transmits your coordinates across the network.", "",
		"  Destination: " + dest, "",
		"  Log in at the destination node when you are ready.", "",
	})
}

// ---- cross-node attack target menu ----

func (g *Dominion) remoteAttackTarget(ctx *engine.Context, attacker *empireState, victim engine.Player) {
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		atm, _ := loadMilitary(ctx.Store.DB(), ctx.Player.GlobalID)
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("CROSS-NODE ATTACK", victim.Name+" @ "+victim.HomeNode))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			f.Line(notice)
			notice = ""
		}
		f.Line(fmt.Sprintf("  Your ATK: %s%d%s    Target DEF: %sunknown%s",
			term.Bright(term.Green), attackPower(atm), term.Reset(),
			term.Bright(term.Red), term.Reset()))
		f.Line(fmt.Sprintf("  Turns: %s%d%s    Attacks left: %s%d%s",
			term.Bright(term.Yellow), turn.Actions, term.Reset(),
			term.Bright(term.Red), turn.Attacks, term.Reset()))
		f.Line("")
		f.Line("  Cross-node attacks are relayed via the hub. Results arrive")
		f.Line("  in your Galactic Dispatches after the defender's node resolves them.")
		f.Line("")
		f.Line(fmt.Sprintf("  %s[G]%s Ground Assault   — 1 turn + 1 attack",
			term.Bright(term.Green), term.Reset()))
		missiles := atm.MissilesNuclear + atm.MissilesAntimatter
		f.Line(fmt.Sprintf("  %s[M]%s Missile Strike   — 1 turn + 1 attack + 1 missile  (have: %d)",
			term.Bright(term.Green), term.Reset(), missiles))
		f.Line("")
		f.Line("  [Q] Return")
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch key {
		case 'g', 'G':
			result := doRemoteGroundAssault(ctx, attacker, atm, victim)
			if strings.Contains(result, "dispatched") {
				g.infoPage(ctx, "ATTACK DISPATCHED", []string{"", "  " + result, ""})
				return
			}
			notice = result
		case 'm', 'M':
			if missiles == 0 {
				notice = noticeBad("No missiles in arsenal.")
				continue
			}
			missileType := g.chooseMissileType(ctx, atm)
			if missileType == "" {
				continue
			}
			result := doRemoteMissileStrike(ctx, attacker, atm, victim, missileType)
			if strings.Contains(result, "launched") {
				g.infoPage(ctx, "MISSILE LAUNCHED", []string{"", "  " + result, ""})
				return
			}
			notice = result
		case 'q', 'Q':
			return
		}
	}
}

// ---- helpers ----

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < 24*time.Hour {
		return "today"
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-2] + ".."
}
