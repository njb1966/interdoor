// Package dominion implements "Empire Ascendant" against the engine.Game interface.
package dominion

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"interdoor.net/interdoor/internal/engine"
	"interdoor.net/interdoor/internal/engine/term"
)

const Version = "0.1.0"

// Dominion is the game module.
type Dominion struct{ nodeID string }

func New(nodeID string) *Dominion  { return &Dominion{nodeID: nodeID} }
func (g *Dominion) ID() string     { return "empire_ascendant" }
func (g *Dominion) Title() string  { return "Empire Ascendant" }

func (g *Dominion) Banner() string {
	y := term.Bright(term.Yellow)
	cy := term.FG(term.Cyan)
	rs := term.Reset()
	return y + bannerArt + rs + "\n\n" +
		cy + "    Command your empire across the stars." + rs + "\n" +
		cy + "    Expand. Research. Conquer. Survive." + rs
}

// ---- schema ----

func (g *Dominion) Migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS empires (
    global_id    TEXT PRIMARY KEY,
    world_name   TEXT NOT NULL DEFAULT '',
    empire_name  TEXT NOT NULL DEFAULT '',
    money        INTEGER NOT NULL DEFAULT 1000,
    money_bank   INTEGER NOT NULL DEFAULT 0,
    population   INTEGER NOT NULL DEFAULT 10000,
    food_storage INTEGER NOT NULL DEFAULT 500,
    energy       INTEGER NOT NULL DEFAULT 100,
    research_pts INTEGER NOT NULL DEFAULT 0,
    building_pts INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_empires_world
    ON empires(world_name) WHERE world_name != '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_empires_empire
    ON empires(empire_name) WHERE empire_name != '';

CREATE TABLE IF NOT EXISTS empire_regions (
    empire_id     TEXT NOT NULL,
    region_type   TEXT NOT NULL,
    quantity      INTEGER NOT NULL DEFAULT 0,
    activated     INTEGER NOT NULL DEFAULT 0,
    activate_cost INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (empire_id, region_type)
);

CREATE TABLE IF NOT EXISTS empire_buildings (
    empire_id            TEXT PRIMARY KEY,
    miners_guild         INTEGER NOT NULL DEFAULT 0,
    fishing_guild        INTEGER NOT NULL DEFAULT 0,
    fishers_assigned     INTEGER NOT NULL DEFAULT 0,
    construction_factory INTEGER NOT NULL DEFAULT 0,
    research_lab         INTEGER NOT NULL DEFAULT 0,
    intel_building       INTEGER NOT NULL DEFAULT 0,
    lottery              INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS empire_mines (
    empire_id       TEXT NOT NULL,
    mine_type       TEXT NOT NULL,
    num_mines       INTEGER NOT NULL DEFAULT 0,
    miners_assigned INTEGER NOT NULL DEFAULT 0,
    mineral_left    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (empire_id, mine_type)
);

CREATE TABLE IF NOT EXISTS empire_tech (
    empire_id  TEXT NOT NULL,
    tech_id    TEXT NOT NULL,
    researched INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (empire_id, tech_id)
);

CREATE TABLE IF NOT EXISTS mineral_store (
    empire_id    TEXT NOT NULL,
    mineral_type TEXT NOT NULL,
    quantity     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (empire_id, mineral_type)
);

CREATE TABLE IF NOT EXISTS empire_military (
    empire_id           TEXT PRIMARY KEY,
    soldiers_normal     INTEGER NOT NULL DEFAULT 0,
    soldiers_super      INTEGER NOT NULL DEFAULT 0,
    soldiers_mega       INTEGER NOT NULL DEFAULT 0,
    tanks               INTEGER NOT NULL DEFAULT 0,
    hovercrafts         INTEGER NOT NULL DEFAULT 0,
    missiles_nuclear    INTEGER NOT NULL DEFAULT 0,
    missiles_antimatter INTEGER NOT NULL DEFAULT 0,
    recon_drones        INTEGER NOT NULL DEFAULT 0,
    spies               INTEGER NOT NULL DEFAULT 0,
    turrets             INTEGER NOT NULL DEFAULT 0,
    satellites          INTEGER NOT NULL DEFAULT 0,
    global_shield       INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pvp_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    empire_id  TEXT NOT NULL,
    event_type TEXT NOT NULL,
    outcome    TEXT NOT NULL,
    detail     TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

CREATE TABLE IF NOT EXISTS attack_limits (
    attacker_id   TEXT NOT NULL,
    target_id     TEXT NOT NULL,
    day_index     INTEGER NOT NULL,
    ground_count  INTEGER NOT NULL DEFAULT 0,
    missile_count INTEGER NOT NULL DEFAULT 0,
    spy_count     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (attacker_id, target_id, day_index)
);
`)
	return err
}

// ---- empire data ----

type empireState struct {
	GlobalID    string `json:"global_id"`
	WorldName   string `json:"world_name"`
	EmpireName  string `json:"empire_name"`
	Money       int    `json:"money"`
	MoneyBank   int    `json:"money_bank"`
	Population  int    `json:"population"`
	FoodStorage int    `json:"food_storage"`
	Energy      int    `json:"energy"`
	ResearchPts int    `json:"research_pts"`
	BuildingPts int    `json:"building_pts"`
}

func loadEmpire(db *sql.DB, globalID string) (*empireState, error) {
	e := &empireState{GlobalID: globalID}
	err := db.QueryRow(
		`SELECT world_name,empire_name,money,money_bank,population,food_storage,energy,research_pts,building_pts
		 FROM empires WHERE global_id=?`, globalID).
		Scan(&e.WorldName, &e.EmpireName, &e.Money, &e.MoneyBank,
			&e.Population, &e.FoodStorage, &e.Energy, &e.ResearchPts, &e.BuildingPts)
	return e, err
}

func saveNames(db *sql.DB, globalID, worldName, empireName string) error {
	_, err := db.Exec(
		`UPDATE empires SET world_name=?, empire_name=? WHERE global_id=?`,
		worldName, empireName, globalID)
	return err
}

// ---- engine.Game interface ----

func (g *Dominion) NewCharacter(db *sql.DB, p *engine.Player) error {
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO empires(global_id, money, building_pts, research_pts) VALUES(?, 2500, 200, 50)`,
		p.GlobalID); err != nil {
		return err
	}
	// Seed starting regions: 2 agricultural (breakeven food), 1 industrial (base energy).
	startingRegions := []struct {
		rtype    string
		qty, act int
	}{
		{RegionAgricultural, 2, 2},
		{RegionIndustrial, 1, 1},
	}
	for _, sr := range startingRegions {
		cost := nextRegionCost(sr.qty, sr.rtype)
		if _, err := db.Exec(`
INSERT OR IGNORE INTO empire_regions(empire_id, region_type, quantity, activated, activate_cost)
VALUES(?,?,?,?,?)`, p.GlobalID, sr.rtype, sr.qty, sr.act, cost); err != nil {
			return err
		}
	}
	// Seed empty buildings row.
	_, err := db.Exec(`INSERT OR IGNORE INTO empire_buildings(empire_id) VALUES(?)`, p.GlobalID)
	return err
}

func (g *Dominion) ExportState(db *sql.DB, globalID string) (json.RawMessage, error) {
	e, err := loadEmpire(db, globalID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(e)
}

func (g *Dominion) ImportState(db *sql.DB, p *engine.Player, state json.RawMessage) error {
	var e empireState
	if err := json.Unmarshal(state, &e); err != nil {
		return err
	}
	e.GlobalID = p.GlobalID
	_, err := db.Exec(`
INSERT INTO empires(global_id,world_name,empire_name,money,money_bank,population,food_storage,energy,research_pts,building_pts)
VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(global_id) DO UPDATE SET
    world_name=excluded.world_name, empire_name=excluded.empire_name,
    money=excluded.money, money_bank=excluded.money_bank,
    population=excluded.population, food_storage=excluded.food_storage,
    energy=excluded.energy, research_pts=excluded.research_pts,
    building_pts=excluded.building_pts`,
		e.GlobalID, e.WorldName, e.EmpireName, e.Money, e.MoneyBank,
		e.Population, e.FoodStorage, e.Energy, e.ResearchPts, e.BuildingPts)
	return err
}

func (g *Dominion) RegisterGameHandlers(store *engine.Store) {
	store.OnEvent("pvp.resolved", func(e engine.Event) error {
		var p pvpResolvedPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return nil
		}
		// Only handle events where the attacker is homed on this node.
		if !strings.HasPrefix(p.AttackerGlobalID, g.nodeID+":") {
			return nil
		}
		db := store.DB()
		if p.LootAmount > 0 {
			_, _ = db.Exec(`UPDATE empires SET money=money+? WHERE global_id=?`,
				p.LootAmount, p.AttackerGlobalID)
		}
		detail := "[Cross-node result] " + p.ResultText
		if p.VictimName != "" {
			detail = fmt.Sprintf("[Cross-node vs %s of %s] %s",
				p.VictimName, p.VictimWorld, p.ResultText)
		}
		_ = writePvPLog(db, p.AttackerGlobalID, pvpLogEntry{
			AttackerName:  p.VictimName,
			AttackerWorld: p.VictimWorld,
			EventType:     "pvp_result",
			Outcome:       p.Outcome,
			Detail:        detail,
		})
		return nil
	})
}

func (g *Dominion) Run(ctx *engine.Context) error {
	db := ctx.Store.DB()

	e, err := loadEmpire(db, ctx.Player.GlobalID)
	if err != nil {
		if err2 := g.NewCharacter(db, ctx.Player); err2 != nil {
			return err2
		}
		if e, err = loadEmpire(db, ctx.Player.GlobalID); err != nil {
			return err
		}
	}

	if e.WorldName == "" {
		if err := g.setupWizard(ctx, e); err != nil {
			return err
		}
	}

	if newDay, _ := ctx.Store.ResetIfNewDay(ctx.Player.GlobalID); newDay {
		_ = runProductionTick(ctx.Store.DB(), ctx.Player.GlobalID)
		_ = purgeInactiveEmpires(ctx.Store.DB(), ctx.Store.NodeID())
		if e, err = loadEmpire(ctx.Store.DB(), ctx.Player.GlobalID); err != nil {
			return err
		}
	}

	if n := countPvPLog(ctx.Store.DB(), ctx.Player.GlobalID); n > 0 {
		g.infoPage(ctx, "GALACTIC DISPATCHES", []string{
			"",
			fmt.Sprintf("  You have %d incoming battle report(s) waiting.", n),
			"",
			"  Open Galactic Dispatches from the Attack Menu to read them.",
			"",
		})
	}

	g.mainMenu(ctx, e)
	ctx.Term.Clear()
	ctx.Term.Write("\r\n  Your empire endures. Until next time, Commander.\r\n\r\n")
	return nil
}

// ---- setup wizard (first login only) ----

func (g *Dominion) setupWizard(ctx *engine.Context, e *empireState) error {
	db := ctx.Store.DB()
	t := ctx.Term
	yc := term.Bright(term.Yellow)
	rs := term.Reset()

	worldName := ""
	empireName := ""
	notice := ""

	// Collect world name.
	for {
		f := term.NewFrame()
		f.Line(titleLine("FOUNDING YOUR EMPIRE", "choose your world"))
		f.Line(hRule(yc, screenW))
		f.Line("")
		if notice != "" {
			f.Line(noticeBad(notice))
			f.Line("")
			notice = ""
		}
		f.Line("  Every empire begins with a world. Choose yours carefully —")
		f.Line("  it is the name the galaxy will know you by.")
		f.Line("")
		f.Line("  World name: 2-30 characters, letters, numbers, spaces, hyphens.")
		f.Line("")
		f.Status("  World name: ")
		f.Render(t)

		name, err := t.ReadLine(true)
		if err != nil {
			return err
		}
		name = strings.TrimSpace(name)
		if msg := validateName(name); msg != "" {
			notice = msg
			continue
		}
		var count int
		_ = db.QueryRow(`SELECT COUNT(*) FROM empires WHERE world_name=?`, name).Scan(&count)
		if count > 0 {
			notice = fmt.Sprintf("World %q is already claimed. Choose another.", name)
			continue
		}
		worldName = name
		break
	}

	// Collect empire name.
	for {
		f := term.NewFrame()
		f.Line(titleLine("FOUNDING YOUR EMPIRE", "name your empire"))
		f.Line(hRule(yc, screenW))
		f.Line("")
		if notice != "" {
			f.Line(noticeBad(notice))
			f.Line("")
			notice = ""
		}
		f.Line(fmt.Sprintf("  World: %s%s%s", term.Bright(term.Yellow), worldName, rs))
		f.Line("")
		f.Line("  Now name the empire that rules it.")
		f.Line("  2-30 characters, letters, numbers, spaces, hyphens.")
		f.Line("")
		f.Status("  Empire name: ")
		f.Render(t)

		name, err := t.ReadLine(true)
		if err != nil {
			return err
		}
		name = strings.TrimSpace(name)
		if msg := validateName(name); msg != "" {
			notice = msg
			continue
		}
		var count int
		_ = db.QueryRow(`SELECT COUNT(*) FROM empires WHERE empire_name=?`, name).Scan(&count)
		if count > 0 {
			notice = fmt.Sprintf("Empire %q already exists on this node. Choose another.", name)
			continue
		}
		empireName = name
		break
	}

	if err := saveNames(db, e.GlobalID, worldName, empireName); err != nil {
		return err
	}
	e.WorldName = worldName
	e.EmpireName = empireName

	// Confirmation screen.
	f := term.NewFrame()
	f.Line(titleLine("EMPIRE FOUNDED", "your claim is recorded"))
	f.Line(hRule(yc, screenW))
	f.Line("")
	f.Line(fmt.Sprintf("  World:   %s%s%s", term.Bright(term.Yellow), worldName, rs))
	f.Line(fmt.Sprintf("  Empire:  %s%s%s", term.Bright(term.Yellow), empireName, rs))
	f.Line("")
	f.Line("  Your empire begins small. Build, research, and expand.")
	f.Line("  The galaxy awaits, Commander.")
	pauseStatus(f)
	f.Render(t)
	_, _ = t.ReadKey()
	return nil
}

func validateName(s string) string {
	if len(s) < 2 {
		return "Name must be at least 2 characters."
	}
	if len(s) > 30 {
		return "Name must be 30 characters or fewer."
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' && r != '-' {
			return fmt.Sprintf("Invalid character %q. Use letters, numbers, spaces, hyphens.", r)
		}
	}
	return ""
}

// ---- main menu ----

func (g *Dominion) mainMenu(ctx *engine.Context, e *empireState) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)

	for {
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("EMPIRE ASCENDANT", e.EmpireName+" of "+e.WorldName))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, menuItem("E", "Enter Your Empire", "manage your realm"), screenW))
		f.Line(boxLine(gc, menuItem("R", "Rankings", "top empires across the galaxy"), screenW))
		f.Line(boxLine(gc, menuItem("N", "Galactic News", "dispatches from the network"), screenW))
		f.Line(boxLine(gc, menuItem("S", "Story / Instructions", "how this all works"), screenW))
		f.Line(boxLine(gc, menuItem("Q", "Disconnect", "leave for now"), screenW))
		f.Line(boxBot(gc, screenW))
		f.Line("")
		f.Line(fmt.Sprintf("  Turns remaining today: %s%d / %d%s",
			term.Bright(term.Yellow), turn.Actions, engine.MainActionsPerDay, term.Reset()))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch key {
		case 'e', 'E':
			g.empireHQ(ctx, e)
		case 'r', 'R':
			g.rankingsScreen(ctx)
		case 'n', 'N':
			g.stubScreen(ctx, "GALACTIC NEWS", "The network is quiet for now.")
		case 's', 'S':
			g.infoPage(ctx, "HOW EMPIRE ASCENDANT WORKS", instructionLines)
		case 'q', 'Q':
			return
		}
	}
}

// ---- empire HQ ----

func (g *Dominion) empireHQ(ctx *engine.Context, e *empireState) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)

	for {
		// Reload from DB each visit to catch any state changes.
		fresh, err := loadEmpire(ctx.Store.DB(), e.GlobalID)
		if err == nil {
			*e = *fresh
		}
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("EMPIRE HQ", e.EmpireName+" — "+e.WorldName))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, menuItem("T", "Empire Report", "resources, population, status"), screenW))
		f.Line(boxLine(gc, menuItem("D", "Develop Empire", "build, research, recruit"), screenW))
		f.Line(boxLine(gc, menuItem("A", "Attack Menu", "assault, missiles, spies"), screenW))
		f.Line(boxLine(gc, menuItem("W", "Wanderers", "roster of known empires"), screenW))
		f.Line(boxLine(gc, menuItem("V", "Warp to Galaxy", "cross-node travel  [requires Hyperdrive]"), screenW))
		f.Line(boxLine(gc, menuItem("I", "Intel Report", "recon and spy results  [coming soon]"), screenW))
		f.Line(boxLine(gc, menuItem("M", "Messages", "player-to-player mail  [coming soon]"), screenW))
		f.Line(boxLine(gc, menuItem("Q", "Return to Main", ""), screenW))
		f.Line(boxBot(gc, screenW))
		f.Line("")
		f.Line(fmt.Sprintf("  Turns remaining: %s%d / %d%s",
			term.Bright(term.Yellow), turn.Actions, engine.MainActionsPerDay, term.Reset()))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch key {
		case 't', 'T':
			g.empireReport(ctx, e)
		case 'd', 'D':
			g.developMenu(ctx, e)
		case 'a', 'A':
			g.attackMenu(ctx, e)
		case 'w', 'W':
			g.wanderersScreen(ctx)
		case 'v', 'V':
			g.warpScreen(ctx, e)
		case 'i', 'I':
			g.stubScreen(ctx, "INTEL REPORT", "Intelligence systems arrive in phase D5.")
		case 'm', 'M':
			g.stubScreen(ctx, "MESSAGES", "Player messaging arrives in phase D5.")
		case 'q', 'Q':
			return
		}
	}
}

// ---- empire report ----

func (g *Dominion) empireReport(ctx *engine.Context, e *empireState) {
	db := ctx.Store.DB()
	if fresh, err := loadEmpire(db, e.GlobalID); err == nil {
		*e = *fresh
	}
	turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
	mil, _ := loadMilitary(db, e.GlobalID)

	yc := term.Bright(term.Yellow)
	cy := term.FG(term.Cyan)
	rs := term.Reset()

	section := func(label string) string { return cy + "  " + label + rs }
	row := func(label, value string) string {
		return fmt.Sprintf("    %-28s%s%s%s", label, term.Bright(term.White), value, rs)
	}

	shieldStr := "offline"
	if mil.GlobalShieldActive > 0 {
		shieldStr = term.Bright(term.Green) + "ACTIVE" + rs
	}

	f := term.NewFrame()
	f.Line(titleLine("EMPIRE REPORT", e.EmpireName))
	f.Line(hRule(yc, screenW))
	f.Line(boxTop(yc, screenW))
	f.Line(boxLine(yc, fmt.Sprintf("  Empire: %s%-28s%s  World: %s%s%s",
		term.Bright(term.Yellow), e.EmpireName, rs,
		term.Bright(term.Yellow), e.WorldName, rs), screenW))
	f.Line(boxLine(yc, fmt.Sprintf("  Commander: %-18s  Turns today: %s%d / %d%s",
		ctx.Player.Name, term.Bright(term.Yellow), turn.Actions, engine.MainActionsPerDay, rs), screenW))
	f.Line(boxBot(yc, screenW))
	f.Line("")
	f.Line(section("TREASURY"))
	f.Line(row("Credits (on hand):", fmtCredits(e.Money)))
	f.Line(row("Credits (banked):", fmtCredits(e.MoneyBank)))
	f.Line("")
	f.Line(section("POPULATION & RESOURCES"))
	f.Line(row("Population:", fmt.Sprintf("%d", e.Population)))
	f.Line(row("Food storage:", fmt.Sprintf("%d", e.FoodStorage)))
	f.Line(row("Energy:", fmt.Sprintf("%d", e.Energy)))
	f.Line("")
	f.Line(section("DEVELOPMENT"))
	f.Line(row("Research points:", fmt.Sprintf("%d", e.ResearchPts)))
	f.Line(row("Building points:", fmt.Sprintf("%d", e.BuildingPts)))
	f.Line("")
	f.Line(section("MILITARY"))
	f.Line(row("Soldiers:", fmt.Sprintf("%d normal / %d super / %d mega",
		mil.SoldiersNormal, mil.SoldiersSuper, mil.SoldiersMega)))
	f.Line(row("Vehicles:", fmt.Sprintf("%d tanks / %d hovercrafts", mil.Tanks, mil.Hovercrafts)))
	f.Line(row("Missiles:", fmt.Sprintf("%d nuclear / %d antimatter", mil.MissilesNuclear, mil.MissilesAntimatter)))
	f.Line(row("Support:", fmt.Sprintf("%d recon drones / %d spies", mil.ReconDrones, mil.Spies)))
	f.Line(row("Defenses:", fmt.Sprintf("%d turrets / %d satellites  Shield: %s",
		mil.Turrets, mil.Satellites, shieldStr)))
	f.Line(row("Attack power:", fmt.Sprintf("%d", attackPower(mil))))
	f.Line(row("Defense power:", fmt.Sprintf("%d", defensePower(mil))))
	pauseStatus(f)
	f.Render(ctx.Term)
	_, _ = ctx.Term.ReadKey()
}

// ---- shared helpers ----

func (g *Dominion) infoPage(ctx *engine.Context, title string, lines []string) {
	yc := term.Bright(term.Yellow)
	f := term.NewFrame()
	f.Line(titleLine(title, ""))
	f.Line(hRule(yc, screenW))
	f.Line(boxTop(yc, screenW))
	for _, ln := range lines {
		f.Line(boxLine(yc, " "+ln, screenW))
	}
	f.Line(boxBot(yc, screenW))
	pauseStatus(f)
	f.Render(ctx.Term)
	_, _ = ctx.Term.ReadKey()
}

func (g *Dominion) stubScreen(ctx *engine.Context, title, msg string) {
	g.infoPage(ctx, title, []string{"", msg, ""})
}

// ---- static text ----

var instructionLines = []string{
	"",
	"  Empire Ascendant is a space empire strategy game for the InterDOOR",
	"  network. Each day you receive 15 turns to spend building your empire.",
	"",
	"  ── ECONOMY ─────────────────────────────────────────────────────────",
	"",
	"  Develop regions to grow food, generate energy, and accumulate wealth.",
	"  Build a Construction Factory to earn building points, then construct",
	"  Research Labs, Miners Guilds, and Fishing Guilds. Mine gold, silver,",
	"  iron, and other minerals — then sell them for credits.",
	"",
	"  ── RESEARCH ────────────────────────────────────────────────────────",
	"",
	"  Research Labs generate research points each day. Spend them in the",
	"  Tech Tree to unlock advanced energy, superior soldiers, vehicles,",
	"  ballistic weapons, orbital defenses, and Hyperdrive travel.",
	"",
	"  ── MILITARY ────────────────────────────────────────────────────────",
	"",
	"  Recruit soldiers, tanks, and hovercrafts. Build turrets and orbital",
	"  satellites. Launch ground assaults or ballistic strikes against rival",
	"  empires. Win to loot credits from their treasury.",
	"",
	"  ── NETWORK ─────────────────────────────────────────────────────────",
	"",
	"  Research Hyperdrive to warp to other nodes in the InterDOOR network.",
	"  Attack empires on distant nodes. Your score is broadcast galaxy-wide.",
	"  Inactive empires (14+ days) are hidden from attack. Abandoned empires",
	"  (30+ days) are purged entirely.",
	"",
	"  ── TURNS ───────────────────────────────────────────────────────────",
	"",
	"  15 turns per day. Warp costs 2. Reports, rankings, and banking are",
	"  always free. You also get 3 attack actions per day.",
	"",
}

// Confirm Dominion satisfies the engine.Game interface at compile time.
var _ engine.Game = (*Dominion)(nil)

// b2i converts bool to int for SQLite storage.
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Suppress unused warning for b2i until D3 uses it.
var _ = b2i

