package dominion

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"interdoor.net/interdoor/internal/engine"
	"interdoor.net/interdoor/internal/engine/term"
)

// ---- attack menu (top level) ----

func (g *Dominion) attackMenu(ctx *engine.Context, e *empireState) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)

	for {
		if fresh, err := loadEmpire(ctx.Store.DB(), e.GlobalID); err == nil {
			*e = *fresh
		}
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
		mil, _ := loadMilitary(ctx.Store.DB(), e.GlobalID)
		dispatches := countPvPLog(ctx.Store.DB(), e.GlobalID)

		shieldLabel := "offline"
		if mil.GlobalShieldActive > 0 {
			shieldLabel = term.Bright(term.Green) + "ACTIVE" + term.Reset()
		}

		dispLabel := ""
		if dispatches > 0 {
			dispLabel = fmt.Sprintf(" [%s%d unread%s]",
				term.Bright(term.Red), dispatches, term.Reset())
		}

		f := term.NewFrame()
		f.Line(titleLine("ATTACK MENU", e.EmpireName))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, menuItem("S", "Select Target", "launch an attack on a rival empire"), screenW))
		f.Line(boxLine(gc, menuItem("G", "Global Shield", "toggle shield  [1000 energy/day]  — "+shieldLabel), screenW))
		f.Line(boxLine(gc, menuItem("D", "Galactic Dispatches", "incoming battle reports"+dispLabel), screenW))
		f.Line(boxLine(gc, menuItem("Q", "Return to HQ", ""), screenW))
		f.Line(boxBot(gc, screenW))
		f.Line("")
		f.Line(fmt.Sprintf("  Turns: %s%d / %d%s    Attacks: %s%d / %d%s    ATK power: %s%d%s",
			term.Bright(term.Yellow), turn.Actions, engine.MainActionsPerDay, term.Reset(),
			term.Bright(term.Red), turn.Attacks, engine.AttacksPerDay, term.Reset(),
			term.FG(term.Cyan), attackPower(mil), term.Reset()))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch key {
		case 's', 'S':
			g.selectAndAttack(ctx, e)
		case 'g', 'G':
			g.globalShieldToggle(ctx, e)
		case 'd', 'D':
			g.galacticDispatches(ctx)
		case 'q', 'Q':
			return
		}
	}
}

// ---- target selection + attack ----

func (g *Dominion) selectAndAttack(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)
	cy := term.FG(term.Cyan)
	notice := ""

	for {
		locals, _ := listTargets(db, e.GlobalID)
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

		var remotes []engine.Player
		if ctx.CrossNodeAttack != nil {
			all, _ := ctx.Store.Players(e.GlobalID)
			for _, p := range all {
				if p.HomeNode != ctx.NodeID {
					remotes = append(remotes, p)
					if len(remotes) >= 9 {
						break
					}
				}
			}
		}

		f := term.NewFrame()
		f.Line(titleLine("SELECT TARGET", "choose an empire to attack"))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			f.Line(noticeBad(notice))
			notice = ""
		}
		f.Line(fmt.Sprintf("  Attacks remaining: %s%d / %d%s    Turns: %s%d / %d%s",
			term.Bright(term.Red), turn.Attacks, engine.AttacksPerDay, term.Reset(),
			term.Bright(term.Yellow), turn.Actions, engine.MainActionsPerDay, term.Reset()))
		f.Line("")

		if len(locals) == 0 && len(remotes) == 0 {
			f.Line("  No rival empires found.")
			f.Line("")
			f.Line("  [Q] Return")
			pauseStatus(f)
			f.Render(ctx.Term)
			ctx.Term.ReadKey()
			return
		}

		if len(locals) > 0 {
			f.Line(fmt.Sprintf("  %sLOCAL%s", cy, term.Reset()))
			for i, t := range locals {
				if i >= 9 {
					break
				}
				tm, _ := loadMilitary(db, t.GlobalID)
				f.Line(fmt.Sprintf("  %s[%d]%s %-20s of %-20s  ATK:%-4d DEF:%-4d",
					gc, i+1, term.Reset(), t.EmpireName, t.WorldName,
					attackPower(tm), defensePower(tm)))
			}
		}
		if len(remotes) > 0 {
			if len(locals) > 0 {
				f.Line("")
			}
			f.Line(fmt.Sprintf("  %sCROSS-NODE%s", cy, term.Reset()))
			for i, p := range remotes {
				f.Line(fmt.Sprintf("  %s[%s]%s %-20s @ %s",
					gc, string(rune('A'+i)), term.Reset(), p.Name, p.HomeNode))
			}
		}

		f.Line("")
		f.Line("  [Q] Return")
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		needTurn := func() bool {
			if turn.Actions < 1 {
				notice = "No turns remaining today."
				return false
			}
			if turn.Attacks < 1 {
				notice = "No attacks remaining today."
				return false
			}
			return true
		}
		switch {
		case key == 'q' || key == 'Q':
			return
		case key >= '1' && key <= '9':
			idx := int(key - '1')
			if idx >= len(locals) || !needTurn() {
				continue
			}
			g.attackTarget(ctx, e, locals[idx])
		case (key >= 'A' && key <= 'I') || (key >= 'a' && key <= 'i'):
			var idx int
			if key >= 'A' {
				idx = int(key - 'A')
			} else {
				idx = int(key - 'a')
			}
			if idx >= len(remotes) || !needTurn() {
				continue
			}
			g.remoteAttackTarget(ctx, e, remotes[idx])
		}
	}
}

func (g *Dominion) attackTarget(ctx *engine.Context, attacker *empireState, target empireStub) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		if fresh, err := loadEmpire(db, attacker.GlobalID); err == nil {
			*attacker = *fresh
		}
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
		atm, _ := loadMilitary(db, attacker.GlobalID)
		defender, _ := loadEmpire(db, target.GlobalID)
		dfm, _ := loadMilitary(db, target.GlobalID)

		ground, missile, spy := attackCounts(db, attacker.GlobalID, target.GlobalID, turn.DayIndex)
		shieldStr := "offline"
		if dfm.GlobalShieldActive > 0 {
			shieldStr = term.Bright(term.Red) + "ACTIVE" + term.Reset()
		}

		f := term.NewFrame()
		f.Line(titleLine("ATTACK", target.EmpireName+" of "+target.WorldName))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			f.Line(notice)
			notice = ""
		}
		f.Line(fmt.Sprintf("  Your ATK: %s%d%s    Target DEF: %s%d%s    Shield: %s",
			term.Bright(term.Green), attackPower(atm), term.Reset(),
			term.Bright(term.Red), defensePower(dfm), term.Reset(),
			shieldStr))
		f.Line(fmt.Sprintf("  Turns: %s%d%s    Attacks left today: %s%d%s",
			term.Bright(term.Yellow), turn.Actions, term.Reset(),
			term.Bright(term.Red), turn.Attacks, term.Reset()))
		f.Line("")
		f.Line(fmt.Sprintf("  %s[G]%s Ground Assault   [%d/3 today vs this target]  — 1 turn + 1 attack",
			term.Bright(term.Green), term.Reset(), ground))
		f.Line(fmt.Sprintf("  %s[M]%s Missile Strike   [%d/2 today vs this target]  — 1 turn + 1 attack + 1 missile",
			term.Bright(term.Green), term.Reset(), missile))
		f.Line(fmt.Sprintf("  %s[Y]%s Deploy Spy       [%d/1 today vs this target]  — 1 turn + 1 attack",
			term.Bright(term.Green), term.Reset(), spy))
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
			notice = g.doGroundAssault(ctx, attacker, atm, defender, dfm, turn.DayIndex)
		case 'm', 'M':
			notice = g.doMissileStrike(ctx, attacker, atm, defender, dfm, turn.DayIndex)
		case 'y', 'Y':
			notice = g.doSpyMission(ctx, attacker, atm, defender, dfm, turn.DayIndex)
		case 'q', 'Q':
			return
		}
	}
}

func (g *Dominion) doGroundAssault(ctx *engine.Context,
	attacker *empireState, atm *militaryRow,
	defender *empireState, dfm *militaryRow,
	dayIndex int) string {

	db := ctx.Store.DB()
	turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

	if attackPower(atm) == 0 {
		return noticeBad("You have no combat units to attack with.")
	}
	if turn.Actions < 1 {
		return noticeBad("No turns remaining today.")
	}
	if turn.Attacks < 1 {
		return noticeBad("No attacks remaining today.")
	}
	if !attackLimitOK(db, attacker.GlobalID, defender.GlobalID, dayIndex, "ground_count", 3) {
		return noticeBad("You have already attacked this empire 3 times today (ground limit).")
	}

	result := groundAssault(attacker, defender, atm, dfm)

	_ = ctx.Store.SpendActions(ctx.Player.GlobalID, 1)
	_ = ctx.Store.SpendAttack(ctx.Player.GlobalID)
	_ = incAttackLimit(db, attacker.GlobalID, defender.GlobalID, dayIndex, "ground_count")
	_ = updateEmpire(db, attacker)
	_ = updateEmpire(db, defender)
	_ = saveMilitary(db, attacker.GlobalID, atm)
	_ = saveMilitary(db, defender.GlobalID, dfm)

	// Outcome text for the defender's inbox.
	defOutcome := "defeat"
	if strings.Contains(result, "VICTORY") {
		defOutcome = "victory"
	}
	_ = writePvPLog(db, defender.GlobalID, pvpLogEntry{
		AttackerName:  attacker.EmpireName,
		AttackerWorld: attacker.WorldName,
		EventType:     "ground",
		Outcome:       defOutcome,
		Detail:        fmt.Sprintf("%s assaulted your forces.", attacker.EmpireName),
	})
	return result
}

func (g *Dominion) doMissileStrike(ctx *engine.Context,
	attacker *empireState, atm *militaryRow,
	defender *empireState, dfm *militaryRow,
	dayIndex int) string {

	db := ctx.Store.DB()
	turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

	if turn.Actions < 1 {
		return noticeBad("No turns remaining today.")
	}
	if turn.Attacks < 1 {
		return noticeBad("No attacks remaining today.")
	}
	if !attackLimitOK(db, attacker.GlobalID, defender.GlobalID, dayIndex, "missile_count", 2) {
		return noticeBad("You have already fired 2 missiles at this empire today.")
	}

	// Choose missile type by what the attacker has.
	missileType := ""
	if atm.MissilesNuclear > 0 {
		missileType = "nuclear"
	} else if atm.MissilesAntimatter > 0 {
		missileType = "antimatter"
	}
	if missileType == "" {
		return noticeBad("You have no missiles. Recruit some first.")
	}

	// Show missile selection if they have both.
	if atm.MissilesNuclear > 0 && atm.MissilesAntimatter > 0 {
		missileType = g.chooseMissileType(ctx, atm)
		if missileType == "" {
			return ""
		}
	}

	result, _ := ballisticStrike(db, defender, dfm, missileType)

	// Consume one missile of the chosen type.
	switch missileType {
	case "nuclear":
		atm.MissilesNuclear--
	case "antimatter":
		atm.MissilesAntimatter--
	}

	_ = ctx.Store.SpendActions(ctx.Player.GlobalID, 1)
	_ = ctx.Store.SpendAttack(ctx.Player.GlobalID)
	_ = incAttackLimit(db, attacker.GlobalID, defender.GlobalID, dayIndex, "missile_count")
	_ = updateEmpire(db, defender)
	_ = saveMilitary(db, attacker.GlobalID, atm)
	_ = saveMilitary(db, defender.GlobalID, dfm)

	_ = writePvPLog(db, defender.GlobalID, pvpLogEntry{
		AttackerName:  attacker.EmpireName,
		AttackerWorld: attacker.WorldName,
		EventType:     "missile",
		Outcome:       "strike",
		Detail:        fmt.Sprintf("%s launched a %s missile at your world.", attacker.EmpireName, missileType),
	})
	return result
}

func (g *Dominion) chooseMissileType(ctx *engine.Context, atm *militaryRow) string {
	f := term.NewFrame()
	f.Line(titleLine("MISSILE STRIKE", "select warhead"))
	f.Line(hRule(term.Bright(term.Yellow), screenW))
	f.Line("")
	f.Line(fmt.Sprintf("  %s[N]%s Nuclear     (%d available) — kills population, damages regions",
		term.Bright(term.Green), term.Reset(), atm.MissilesNuclear))
	f.Line(fmt.Sprintf("  %s[A]%s Antimatter  (%d available) — destroys enemy military",
		term.Bright(term.Green), term.Reset(), atm.MissilesAntimatter))
	f.Line("")
	f.Line("  [Q] Cancel")
	pauseStatus(f)
	f.Render(ctx.Term)
	key, _ := ctx.Term.ReadKey()
	switch key {
	case 'n', 'N':
		return "nuclear"
	case 'a', 'A':
		return "antimatter"
	}
	return ""
}

func (g *Dominion) doSpyMission(ctx *engine.Context,
	attacker *empireState, atm *militaryRow,
	defender *empireState, dfm *militaryRow,
	dayIndex int) string {

	db := ctx.Store.DB()
	turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

	if turn.Actions < 1 {
		return noticeBad("No turns remaining today.")
	}
	if turn.Attacks < 1 {
		return noticeBad("No attacks remaining today.")
	}
	if atm.Spies < 1 {
		return noticeBad("You have no spies. Recruit some first.")
	}
	buildings, _ := loadBuildings(db, attacker.GlobalID)
	if buildings.IntelBuilding < 1 {
		return noticeBad("You need an Intelligence Building to deploy spies.")
	}
	if !attackLimitOK(db, attacker.GlobalID, defender.GlobalID, dayIndex, "spy_count", 1) {
		return noticeBad("You have already deployed a spy against this empire today.")
	}

	// Choose mission type.
	mission := g.chooseSpyMission(ctx)
	if mission == "" {
		return ""
	}

	_ = ctx.Store.SpendActions(ctx.Player.GlobalID, 1)
	_ = ctx.Store.SpendAttack(ctx.Player.GlobalID)
	_ = incAttackLimit(db, attacker.GlobalID, defender.GlobalID, dayIndex, "spy_count")

	var result string
	switch mission {
	case "scout":
		result = spyScout(defender, dfm)
		g.showMultilineResult(ctx, "SCOUT REPORT", result)
		return ""
	case "sabotage":
		msg, consumed := spySabotage(db, defender, dfm)
		if consumed {
			atm.Spies--
		}
		_ = saveMilitary(db, attacker.GlobalID, atm)
		_ = saveMilitary(db, defender.GlobalID, dfm)
		_ = writePvPLog(db, defender.GlobalID, pvpLogEntry{
			AttackerName:  attacker.EmpireName,
			AttackerWorld: attacker.WorldName,
			EventType:     "spy",
			Outcome:       "sabotage",
			Detail:        fmt.Sprintf("A spy from %s was detected near your facilities.", attacker.EmpireName),
		})
		result = msg
	}
	return result
}

func (g *Dominion) chooseSpyMission(ctx *engine.Context) string {
	f := term.NewFrame()
	f.Line(titleLine("SPY MISSION", "choose operation type"))
	f.Line(hRule(term.Bright(term.Yellow), screenW))
	f.Line("")
	f.Line(fmt.Sprintf("  %s[S]%s Scout      — reveal army, money, defenses (spy returns)",
		term.Bright(term.Green), term.Reset()))
	f.Line(fmt.Sprintf("  %s[B]%s Sabotage   — destroy a target; spy consumed on failure (50%% success)",
		term.Bright(term.Green), term.Reset()))
	f.Line("")
	f.Line("  [Q] Cancel")
	pauseStatus(f)
	f.Render(ctx.Term)
	key, _ := ctx.Term.ReadKey()
	switch key {
	case 's', 'S':
		return "scout"
	case 'b', 'B':
		return "sabotage"
	}
	return ""
}

func (g *Dominion) showMultilineResult(ctx *engine.Context, title, text string) {
	yc := term.Bright(term.Yellow)
	lines := strings.Split(text, "\n")
	f := term.NewFrame()
	f.Line(titleLine(title, ""))
	f.Line(hRule(yc, screenW))
	for _, ln := range lines {
		f.Line("  " + ln)
	}
	pauseStatus(f)
	f.Render(ctx.Term)
	ctx.Term.ReadKey()
}

// ---- global shield toggle ----

func (g *Dominion) globalShieldToggle(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	if !hasTech(db, e.GlobalID, TechGlobalShield) {
		g.infoPage(ctx, "GLOBAL SHIELD", []string{
			"", "Global Shield requires the Global Shield technology.", "",
			"Research it via Develop Empire → Research Tech.", "",
		})
		return
	}
	mil, _ := loadMilitary(db, e.GlobalID)
	if mil.GlobalShieldActive > 0 {
		mil.GlobalShieldActive = 0
		_ = saveMilitary(db, e.GlobalID, mil)
		g.infoPage(ctx, "GLOBAL SHIELD", []string{"", noticeWin("Shield deactivated."), ""})
	} else {
		if e.Energy < 1000 {
			g.infoPage(ctx, "GLOBAL SHIELD", []string{
				"", noticeBad(fmt.Sprintf("Insufficient energy. Need 1000, have %d.", e.Energy)), "",
			})
			return
		}
		mil.GlobalShieldActive = 1
		_ = saveMilitary(db, e.GlobalID, mil)
		g.infoPage(ctx, "GLOBAL SHIELD", []string{
			"", noticeWin("Shield activated. Costs 1000 energy per day."),
			"", "The shield blocks all incoming ballistic strikes.",
			"If energy drops to 0, the shield will go offline automatically.", "",
		})
	}
}

// ---- galactic dispatches ----

func (g *Dominion) galacticDispatches(ctx *engine.Context) {
	db := ctx.Store.DB()
	entries, _ := loadPvPLog(db, ctx.Player.GlobalID)
	_ = clearPvPLog(db, ctx.Player.GlobalID)

	yc := term.Bright(term.Yellow)
	f := term.NewFrame()
	f.Line(titleLine("GALACTIC DISPATCHES", "incoming battle reports"))
	f.Line(hRule(yc, screenW))
	f.Line("")
	if len(entries) == 0 {
		f.Line("  No new dispatches.")
	} else {
		for i, e := range entries {
			if i >= 15 {
				f.Line(fmt.Sprintf("  ... and %d more.", len(entries)-15))
				break
			}
			outcomeColor := term.Bright(term.Green)
			if e.Outcome == "defeat" || e.Outcome == "strike" {
				outcomeColor = term.Bright(term.Red)
			}
			f.Line(fmt.Sprintf("  %s[%s]%s %s — %s — %s",
				outcomeColor, strings.ToUpper(e.EventType), term.Reset(),
				e.AttackerName+" of "+e.AttackerWorld,
				e.Outcome,
				e.Detail))
		}
	}
	pauseStatus(f)
	f.Render(ctx.Term)
	ctx.Term.ReadKey()
}

// ---- rankings ----

func (g *Dominion) rankingsScreen(ctx *engine.Context) {
	db := ctx.Store.DB()
	ranks, err := localRankings(db)
	yc := term.Bright(term.Yellow)

	f := term.NewFrame()
	f.Line(titleLine("EMPIRE RANKINGS", "by galactic score"))
	f.Line(hRule(yc, screenW))
	f.Line("")

	if err != nil || len(ranks) == 0 {
		f.Line("  No rankings data available.")
	} else {
		f.Line(fmt.Sprintf("  %s%-4s %-22s %-22s %8s%s",
			term.FG(term.Cyan), "RANK", "EMPIRE", "WORLD", "SCORE", term.Reset()))
		f.Line(hRule(term.FG(term.Blue), screenW))
		for i, r := range ranks {
			if i >= 16 {
				break
			}
			marker := "  "
			if i == 0 {
				marker = term.Bright(term.Yellow) + " *" + term.Reset()
			}
			f.Line(fmt.Sprintf("  %s%-4d %-22s %-22s %8d",
				marker, i+1, r.EmpireName, r.WorldName, r.Score))
		}
	}
	pauseStatus(f)
	f.Render(ctx.Term)
	ctx.Term.ReadKey()
}

// ---- recruit units ----

func (g *Dominion) recruitMenu(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		if fresh, err := loadEmpire(db, e.GlobalID); err == nil {
			*e = *fresh
		}
		mil, _ := loadMilitary(db, e.GlobalID)
		buildings, _ := loadBuildings(db, e.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("RECRUIT UNITS", "spend credits to build your forces"))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			f.Line(notice)
			notice = ""
		}
		f.Line(fmt.Sprintf("  Credits: %s%s%s",
			term.Bright(term.White), fmtCredits(e.Money), term.Reset()))
		f.Line("")
		f.Line(fmt.Sprintf("  %s%-3s %-22s %7s %8s  %s%s",
			term.FG(term.Cyan), "#", "UNIT", "COST", "OWNED", "STATUS", term.Reset()))
		f.Line(hRule(term.FG(term.Blue), screenW))

		for _, ud := range unitDefs {
			available, reason := unitAvailable(db, e.GlobalID, buildings, ud)
			owned := unitCount(mil, ud.field)
			statusStr := term.Bright(term.Green) + "available" + term.Reset()
			if !available {
				statusStr = term.FG(term.Blue) + reason + term.Reset()
			}
			f.Line(fmt.Sprintf("  [%s] %-22s %6d cr %7d  %s",
				ud.key, ud.name, ud.costCr, owned, statusStr))
		}
		f.Line("")
		f.Line("  [Q] Return")
		f.Line("")
		f.Line("  Choose a unit type, then enter quantity:")
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		if key == 'q' || key == 'Q' {
			return
		}

		var chosen *unitDef
		for i := range unitDefs {
			if rune(unitDefs[i].key[0]) == key {
				chosen = &unitDefs[i]
				break
			}
		}
		if chosen == nil {
			continue
		}

		if avail, reason := unitAvailable(db, e.GlobalID, buildings, *chosen); !avail {
			notice = noticeBad("Cannot recruit " + chosen.name + ": " + reason)
			continue
		}

		// Ask how many.
		f2 := term.NewFrame()
		f2.Line(titleLine("RECRUIT", chosen.name))
		f2.Line(hRule(yc, screenW))
		f2.Line(fmt.Sprintf("  Cost: %d cr each    You have: %s",
			chosen.costCr, fmtCredits(e.Money)))
		maxAffordable := e.Money / chosen.costCr
		if maxAffordable > 0 {
			f2.Line(fmt.Sprintf("  You can afford up to %d.", maxAffordable))
		}
		f2.Status("  Quantity: ")
		f2.Render(ctx.Term)

		line, err := ctx.Term.ReadLine(true)
		if err != nil {
			return
		}
		qty, err2 := strconv.Atoi(strings.TrimSpace(line))
		if err2 != nil || qty <= 0 {
			notice = noticeBad("Enter a positive number.")
			continue
		}
		total := qty * chosen.costCr
		if total > e.Money {
			notice = noticeBad(fmt.Sprintf("Need %s, have %s.", fmtCredits(total), fmtCredits(e.Money)))
			continue
		}
		e.Money -= total
		addUnits(mil, chosen.field, qty)
		_ = updateEmpire(db, e)
		_ = saveMilitary(db, e.GlobalID, mil)
		notice = noticeWin(fmt.Sprintf("Recruited %d %s for %s.", qty, chosen.name, fmtCredits(total)))
	}
}

func unitAvailable(db *sql.DB, globalID string, b *buildingsRow, ud unitDef) (bool, string) {
	if ud.reqBldg == "intel" && b.IntelBuilding < 1 {
		return false, "requires Intelligence Building"
	}
	if ud.requires != "" && !hasTech(db, globalID, ud.requires) {
		return false, "requires " + ud.requires
	}
	return true, ""
}
