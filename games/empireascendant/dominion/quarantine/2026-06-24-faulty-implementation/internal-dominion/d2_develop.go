package dominion

import (
	"fmt"
	"strconv"
	"strings"

	"interdoor.net/interdoor/internal/engine"
	"interdoor.net/interdoor/internal/engine/term"
)

// ---- develop empire top menu ----

func (g *Dominion) developMenu(ctx *engine.Context, e *empireState) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)

	for {
		if fresh, err := loadEmpire(ctx.Store.DB(), e.GlobalID); err == nil {
			*e = *fresh
		}
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("DEVELOP EMPIRE", e.EmpireName))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, menuItem("R", "Develop Region", "activate land for production  [1 turn + money]"), screenW))
		f.Line(boxLine(gc, menuItem("B", "Build Structure", "construct buildings  [bldg pts + money]"), screenW))
		f.Line(boxLine(gc, menuItem("T", "Research Tech", "unlock technologies  [research pts]"), screenW))
		f.Line(boxLine(gc, menuItem("U", "Recruit Units", "hire soldiers, vehicles, missiles  [money]"), screenW))
		f.Line(boxLine(gc, menuItem("K", "Banking", "deposit or withdraw credits  [free]"), screenW))
		f.Line(boxLine(gc, menuItem("S", "Sell Minerals", "convert stored minerals to credits  [free]"), screenW))
		f.Line(boxLine(gc, menuItem("Q", "Return to HQ", ""), screenW))
		f.Line(boxBot(gc, screenW))
		f.Line("")
		f.Line(fmt.Sprintf("  Turns: %s%d / %d%s    Credits: %s%s%s    Research: %s%d pts%s    Building: %s%d pts%s",
			term.Bright(term.Yellow), turn.Actions, engine.MainActionsPerDay, term.Reset(),
			term.Bright(term.White), fmtCredits(e.Money), term.Reset(),
			term.FG(term.Cyan), e.ResearchPts, term.Reset(),
			term.FG(term.Cyan), e.BuildingPts, term.Reset()))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch key {
		case 'r', 'R':
			g.developRegionMenu(ctx, e)
		case 'b', 'B':
			g.buildStructureMenu(ctx, e)
		case 't', 'T':
			g.researchTechMenu(ctx, e)
		case 'u', 'U':
			g.recruitMenu(ctx, e)
		case 'k', 'K':
			g.bankMenu(ctx, e)
		case 's', 'S':
			g.sellMineralsMenu(ctx, e)
		case 'q', 'Q':
			return
		}
	}
}

// ---- develop region ----

func (g *Dominion) developRegionMenu(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		if fresh, err := loadEmpire(db, e.GlobalID); err == nil {
			*e = *fresh
		}
		regions, _ := loadRegions(db, e.GlobalID)
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("DEVELOP REGION", "activate land for production"))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			f.Line(noticeBad(notice))
			notice = ""
		}
		f.Line(fmt.Sprintf("  Turns remaining: %s%d / %d%s    Credits: %s%s%s",
			term.Bright(term.Yellow), turn.Actions, engine.MainActionsPerDay, term.Reset(),
			term.Bright(term.White), fmtCredits(e.Money), term.Reset()))
		f.Line("")
		f.Line(padRight(fmt.Sprintf("  %s%-14s %6s %9s %10s%s",
			term.FG(term.Cyan), "TYPE", "OWNED", "ACTIVE", "NEXT COST", term.Reset()), screenW))
		f.Line(hRule(term.FG(term.Blue), screenW))

		keys := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
		for i, rt := range regionTypes {
			r := regions[rt]
			qty, act := 0, 0
			if r != nil {
				qty = r.Quantity
				act = r.Activated
			}
			cost := nextRegionCost(qty, rt)
			f.Line(fmt.Sprintf("  %s[%s]%s %-14s %6d %9d %9d cr",
				term.Bright(term.Green), keys[i], term.Reset(),
				rt, qty, act, cost))
		}
		f.Line("")
		f.Line("  [Q] Return")
		f.Line("")
		f.Line("  Choose a region type to develop (costs 1 turn + credits):")
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		if key == 'q' || key == 'Q' {
			return
		}

		idx := -1
		for i, k := range keys {
			if rune(k[0]) == key {
				idx = i
				break
			}
		}
		if idx < 0 || idx >= len(regionTypes) {
			continue
		}

		rt := regionTypes[idx]
		r := regions[rt]
		qty := 0
		if r != nil {
			qty = r.Quantity
		}
		cost := nextRegionCost(qty, rt)

		if turn.Actions < 1 {
			notice = "No turns remaining today."
			continue
		}
		if e.Money < cost {
			notice = fmt.Sprintf("Need %d cr to develop a %s region (you have %d cr).", cost, rt, e.Money)
			continue
		}

		e.Money -= cost
		if err := updateEmpire(db, e); err != nil {
			notice = "Error saving region: " + err.Error()
			continue
		}
		if err := addRegion(db, e.GlobalID, rt, nextRegionCost(qty+1, rt)); err != nil {
			e.Money += cost // rollback in-memory
			notice = "Error adding region: " + err.Error()
			continue
		}
		if err := ctx.Store.SpendActions(ctx.Player.GlobalID, 1); err != nil {
			notice = "Error spending turn: " + err.Error()
			continue
		}
		notice = fmt.Sprintf("%s Developed a new %s region for %d cr.", noticeWin(""), rt, cost)
	}
}

// ---- build structure ----

type structureDef struct {
	key      string
	name     string
	bldgCost int
	monCost  int
	desc     string
}

var structures = []structureDef{
	{"1", "Research Lab", 80, 400, "+10 research pts/day"},
	{"2", "Construction Factory", 100, 500, "+5 building pts/day"},
	{"3", "Fishing Guild", 60, 300, "hire fishers (+50 food/fisher/day)"},
	{"4", "Miners Guild", 70, 350, "assign miners to mine types"},
	{"5", "Intelligence Building", 90, 450, "unlocks spy recruitment and deployment"},
	// D3 defense structures — only show when tech is researched (filtered at render time).
	{"6", "Ground Turret", 150, 750, "+20 defense power per turret  [req. Ground Turrets tech]"},
	{"7", "Orbital Satellite", 250, 1500, "+50 def, intercepts missiles  [req. Orbital Satellites tech]"},
}

func (g *Dominion) buildStructureMenu(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		if fresh, err := loadEmpire(db, e.GlobalID); err == nil {
			*e = *fresh
		}
		buildings, _ := loadBuildings(db, e.GlobalID)
		mil, _ := loadMilitary(db, e.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("BUILD STRUCTURE", "construction costs building pts + credits"))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			f.Line(notice)
			notice = ""
		}
		f.Line(fmt.Sprintf("  Building pts: %s%d%s    Credits: %s%s%s",
			term.Bright(term.Yellow), e.BuildingPts, term.Reset(),
			term.Bright(term.White), fmtCredits(e.Money), term.Reset()))
		f.Line("")
		f.Line(fmt.Sprintf("  %s%-24s %8s %8s  %s%s",
			term.FG(term.Cyan), "STRUCTURE", "BLDG PTS", "CREDITS", "OWNED", term.Reset()))
		f.Line(hRule(term.FG(term.Blue), screenW))

		for _, sd := range structures {
			// Defense structures hidden until tech is researched.
			if sd.name == "Ground Turret" && !hasTech(db, e.GlobalID, TechTurrets) {
				continue
			}
			if sd.name == "Orbital Satellite" && !hasTech(db, e.GlobalID, TechSatellites) {
				continue
			}
			owned := structureOwned(buildings, mil, sd.name)
			f.Line(fmt.Sprintf("  %s[%s]%s %-24s %8d %7d cr  %d",
				term.Bright(term.Green), sd.key, term.Reset(),
				sd.name, sd.bldgCost, sd.monCost, owned))
			f.Line(fmt.Sprintf("       %s%s%s", term.FG(term.Cyan), sd.desc, term.Reset()))
		}
		f.Line("")
		f.Line("  [Q] Return")
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		if key == 'q' || key == 'Q' {
			return
		}

		var chosen *structureDef
		for i := range structures {
			if rune(structures[i].key[0]) == key {
				chosen = &structures[i]
				break
			}
		}
		if chosen == nil {
			continue
		}
		// Enforce tech gate even if key is pressed directly.
		if chosen.name == "Ground Turret" && !hasTech(db, e.GlobalID, TechTurrets) {
			continue
		}
		if chosen.name == "Orbital Satellite" && !hasTech(db, e.GlobalID, TechSatellites) {
			continue
		}

		if e.BuildingPts < chosen.bldgCost {
			notice = noticeBad(fmt.Sprintf("Need %d building pts (you have %d).", chosen.bldgCost, e.BuildingPts))
			continue
		}
		if e.Money < chosen.monCost {
			notice = noticeBad(fmt.Sprintf("Need %d cr (you have %d cr).", chosen.monCost, e.Money))
			continue
		}

		e.BuildingPts -= chosen.bldgCost
		e.Money -= chosen.monCost

		var saveErr error
		switch chosen.name {
		case "Ground Turret":
			mil.Turrets++
			saveErr = saveMilitary(db, e.GlobalID, mil)
		case "Orbital Satellite":
			mil.Satellites++
			saveErr = saveMilitary(db, e.GlobalID, mil)
		default:
			applyStructure(buildings, chosen.name, 1)
			saveErr = saveBuildings(db, e.GlobalID, buildings)
		}
		if saveErr != nil {
			notice = noticeBad("Error building: " + saveErr.Error())
			continue
		}
		if err := updateEmpire(db, e); err != nil {
			notice = noticeBad("Error saving resources: " + err.Error())
			continue
		}
		notice = noticeWin(fmt.Sprintf("Built a %s.", chosen.name))
	}
}

func structureOwned(b *buildingsRow, m *militaryRow, name string) int {
	switch name {
	case "Research Lab":
		return b.ResearchLab
	case "Construction Factory":
		return b.ConstructionFactory
	case "Fishing Guild":
		return b.FishingGuild
	case "Miners Guild":
		return b.MinersGuild
	case "Intelligence Building":
		return b.IntelBuilding
	case "Ground Turret":
		return m.Turrets
	case "Orbital Satellite":
		return m.Satellites
	}
	return 0
}

func applyStructure(b *buildingsRow, name string, delta int) {
	switch name {
	case "Research Lab":
		b.ResearchLab += delta
	case "Construction Factory":
		b.ConstructionFactory += delta
	case "Fishing Guild":
		b.FishingGuild += delta
	case "Miners Guild":
		b.MinersGuild += delta
	case "Intelligence Building":
		b.IntelBuilding += delta
	}
}

// ---- research tech ----

type techDef struct {
	key      string
	id       string
	name     string
	cost     int
	requires string // empty = no prereq
	desc     string
}

var techTree = []techDef{
	// Energy
	{"1", TechFission, "Fission Energy", techCost[TechFission], "", "Industrial regions produce 500 energy/day"},
	{"2", TechFusion, "Fusion Energy", techCost[TechFusion], TechFission, "Industrial produce 2000/day (req. Fission)"},
	// Soldiers
	{"3", TechSuperhuman, "SuperHuman Soldiers", techCost[TechSuperhuman], "", "Unlocks SuperHuman recruitment (str 3)"},
	{"4", TechMegahuman, "MegaHuman Soldiers", techCost[TechMegahuman], TechSuperhuman, "Unlocks MegaHuman recruitment (str 8, req. Super)"},
	// Vehicles
	{"5", TechTank, "Tank Warfare", techCost[TechTank], "", "Unlocks tank recruitment (str 15)"},
	{"6", TechHovercraft, "Hovercraft Warfare", techCost[TechHovercraft], TechTank, "Unlocks hovercrafts (str 20, req. Tank)"},
	// Ballistic
	{"7", TechNuclear, "Nuclear Missiles", techCost[TechNuclear], "", "Unlocks nuclear missile recruitment"},
	{"8", TechAntimatter, "Antimatter Missiles", techCost[TechAntimatter], TechNuclear, "Unlocks antimatter missiles (req. Nuclear)"},
	// Defense
	{"9", TechTurrets, "Ground Turrets", techCost[TechTurrets], "", "Unlocks ground turret construction (+20 def each)"},
	{"A", TechSatellites, "Orbital Satellites", techCost[TechSatellites], TechTurrets, "Unlocks satellites (+50 def, intercepts missiles)"},
	{"B", TechGlobalShield, "Global Shield", techCost[TechGlobalShield], TechSatellites, "Unlocks Global Shield (blocks all missiles, 1000 energy/day)"},
}

func (g *Dominion) researchTechMenu(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		if fresh, err := loadEmpire(db, e.GlobalID); err == nil {
			*e = *fresh
		}

		f := term.NewFrame()
		f.Line(titleLine("RESEARCH TECHNOLOGY", "spend research points to unlock technologies"))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			if strings.HasPrefix(notice, ">>") {
				f.Line(notice)
			} else {
				f.Line(noticeBad(notice))
			}
			notice = ""
		}
		f.Line(fmt.Sprintf("  Research pts: %s%d%s", term.Bright(term.Yellow), e.ResearchPts, term.Reset()))
		f.Line("")

		for _, td := range techTree {
			done := hasTech(db, e.GlobalID, td.id)
			locked := td.requires != "" && !hasTech(db, e.GlobalID, td.requires)
			var status string
			if done {
				status = term.Bright(term.Green) + "RESEARCHED" + term.Reset()
			} else if locked {
				status = term.FG(term.Blue) + "LOCKED" + term.Reset()
			} else {
				status = fmt.Sprintf("%s%d pts%s", term.Bright(term.Yellow), td.cost, term.Reset())
			}
			f.Line(fmt.Sprintf("  %s[%s]%s %-20s %s",
				term.Bright(term.Green), td.key, term.Reset(), td.name, status))
			f.Line(fmt.Sprintf("       %s%s%s", term.FG(term.Cyan), td.desc, term.Reset()))
		}
		f.Line("")
		f.Line("  [Q] Return")
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		if key == 'q' || key == 'Q' {
			return
		}

		var chosen *techDef
		for i := range techTree {
			if rune(techTree[i].key[0]) == key {
				chosen = &techTree[i]
				break
			}
		}
		if chosen == nil {
			continue
		}
		if hasTech(db, e.GlobalID, chosen.id) {
			notice = fmt.Sprintf("%s already researched.", chosen.name)
			continue
		}
		if chosen.requires != "" && !hasTech(db, e.GlobalID, chosen.requires) {
			notice = "Prerequisite technology not yet researched."
			continue
		}
		if e.ResearchPts < chosen.cost {
			notice = fmt.Sprintf("Need %d research pts (you have %d).", chosen.cost, e.ResearchPts)
			continue
		}

		e.ResearchPts -= chosen.cost
		if err := updateEmpire(db, e); err != nil {
			notice = "Error: " + err.Error()
			continue
		}
		if err := setTech(db, e.GlobalID, chosen.id); err != nil {
			notice = "Error: " + err.Error()
			continue
		}
		notice = noticeWin(fmt.Sprintf("Researched %s!", chosen.name))
	}
}

// ---- bank ----

func (g *Dominion) bankMenu(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		if fresh, err := loadEmpire(db, e.GlobalID); err == nil {
			*e = *fresh
		}

		f := term.NewFrame()
		f.Line(titleLine("GALACTIC BANK", "safe storage for your credits"))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			if strings.HasPrefix(notice, ">>") {
				f.Line(notice)
			} else {
				f.Line(noticeBad(notice))
			}
			notice = ""
		}
		f.Line(fmt.Sprintf("  Credits on hand: %s%s%s", term.Bright(term.White), fmtCredits(e.Money), term.Reset()))
		f.Line(fmt.Sprintf("  Credits banked:  %s%s%s", term.Bright(term.White), fmtCredits(e.MoneyBank), term.Reset()))
		f.Line("")
		f.Line("  [D] Deposit    [W] Withdraw    [Q] Return")
		f.Line("")
		f.Line("  Amount (or Q to go back): ")
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		if key == 'q' || key == 'Q' {
			return
		}
		if key != 'd' && key != 'D' && key != 'w' && key != 'W' {
			continue
		}
		deposit := key == 'd' || key == 'D'

		// Read amount
		f2 := term.NewFrame()
		if deposit {
			f2.Line(titleLine("GALACTIC BANK", "deposit"))
		} else {
			f2.Line(titleLine("GALACTIC BANK", "withdraw"))
		}
		f2.Line(hRule(yc, screenW))
		if deposit {
			f2.Line(fmt.Sprintf("  On hand: %s    Banked: %s",
				fmtCredits(e.Money), fmtCredits(e.MoneyBank)))
			f2.Line("")
			f2.Line("  How many credits to deposit?")
		} else {
			f2.Line(fmt.Sprintf("  On hand: %s    Banked: %s",
				fmtCredits(e.Money), fmtCredits(e.MoneyBank)))
			f2.Line("")
			f2.Line("  How many credits to withdraw?")
		}
		f2.Status("  Amount: ")
		f2.Render(ctx.Term)

		line, err := ctx.Term.ReadLine(true)
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line == "" || line == "q" || line == "Q" {
			continue
		}
		amt, err2 := strconv.Atoi(line)
		if err2 != nil || amt <= 0 {
			notice = "Enter a positive number."
			continue
		}

		if deposit {
			if amt > e.Money {
				notice = fmt.Sprintf("You only have %s on hand.", fmtCredits(e.Money))
				continue
			}
			e.Money -= amt
			e.MoneyBank += amt
			notice = noticeWin(fmt.Sprintf("Deposited %s.", fmtCredits(amt)))
		} else {
			if amt > e.MoneyBank {
				notice = fmt.Sprintf("Only %s in the bank.", fmtCredits(e.MoneyBank))
				continue
			}
			e.MoneyBank -= amt
			e.Money += amt
			notice = noticeWin(fmt.Sprintf("Withdrew %s.", fmtCredits(amt)))
		}
		_ = updateEmpire(db, e)
	}
}

// ---- sell minerals ----

func (g *Dominion) sellMineralsMenu(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	yc := term.Bright(term.Yellow)
	notice := ""

	for {
		if fresh, err := loadEmpire(db, e.GlobalID); err == nil {
			*e = *fresh
		}
		store, _ := loadMineralStore(db, e.GlobalID)

		hasAny := false
		for _, q := range store {
			if q > 0 {
				hasAny = true
				break
			}
		}

		f := term.NewFrame()
		f.Line(titleLine("MINERAL MARKET", "sell stored minerals for credits"))
		f.Line(hRule(yc, screenW))
		if notice != "" {
			if strings.HasPrefix(notice, ">>") {
				f.Line(notice)
			} else {
				f.Line(noticeBad(notice))
			}
			notice = ""
		}
		f.Line(fmt.Sprintf("  Credits on hand: %s%s%s", term.Bright(term.White), fmtCredits(e.Money), term.Reset()))
		f.Line("")
		f.Line(fmt.Sprintf("  %s%-12s %8s %10s %12s%s",
			term.FG(term.Cyan), "MINERAL", "STORED", "PRICE", "VALUE", term.Reset()))
		f.Line(hRule(term.FG(term.Blue), screenW))

		for i, mt := range mineTypes {
			qty := store[mt]
			price := mineralPrice[mt]
			value := qty * price
			f.Line(fmt.Sprintf("  %s[%d]%s %-12s %8d %9d cr %11d cr",
				term.Bright(term.Green), i+1, term.Reset(), mt, qty, price, value))
		}
		f.Line("")
		if hasAny {
			f.Line("  [A] Sell All    [1-5] Sell one type    [Q] Return")
		} else {
			f.Line("  No minerals in storage.")
			f.Line("  [Q] Return")
		}
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		if key == 'q' || key == 'Q' {
			return
		}

		if !hasAny {
			continue
		}

		if key == 'a' || key == 'A' {
			total := 0
			for _, mt := range mineTypes {
				qty := store[mt]
				if qty <= 0 {
					continue
				}
				total += qty * mineralPrice[mt]
				_ = clearMineralStore(db, e.GlobalID, mt)
			}
			e.Money += total
			_ = updateEmpire(db, e)
			notice = noticeWin(fmt.Sprintf("Sold all minerals for %s.", fmtCredits(total)))
			continue
		}

		idx := int(key - '1')
		if idx < 0 || idx >= len(mineTypes) {
			continue
		}
		mt := mineTypes[idx]
		qty := store[mt]
		if qty <= 0 {
			notice = fmt.Sprintf("No %s in storage.", mt)
			continue
		}
		earned := qty * mineralPrice[mt]
		e.Money += earned
		_ = clearMineralStore(db, e.GlobalID, mt)
		_ = updateEmpire(db, e)
		notice = noticeWin(fmt.Sprintf("Sold %d units of %s for %s.", qty, mt, fmtCredits(earned)))
	}
}
