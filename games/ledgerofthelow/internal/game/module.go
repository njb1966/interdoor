// Package game implements "Ledger of the Low: The Old Bargain", the InterDOOR
// reference game, against the engine.Game interface.
package game

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"interdoor.net/interdoor/internal/engine"
	"interdoor.net/interdoor/internal/engine/term"
)

const Version = "1.0.0"

// screenW is the fixed terminal width (NETWORK_REQUIREMENTS Req 8: 80x24).
const screenW = 80

const remoteRosterStaleAfter = 15 * time.Minute

type Ledger struct{ nodeID string }

func New(nodeID string) *Ledger { return &Ledger{nodeID: nodeID} }
func (g *Ledger) ID() string    { return "ledger_of_the_low" }
func (g *Ledger) Title() string { return "Ledger of the Low" }

func (g *Ledger) Migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS char_state (
    global_id    TEXT PRIMARY KEY,
    hp           INTEGER NOT NULL,
    max_hp       INTEGER NOT NULL,
    strength     INTEGER NOT NULL,
    defense      INTEGER NOT NULL,
    luck         INTEGER NOT NULL,
    weapon        TEXT NOT NULL,
    armor         TEXT NOT NULL,
    depth_record  INTEGER NOT NULL DEFAULT 0,
    blooded       INTEGER NOT NULL DEFAULT 0,
    tutorial_done INTEGER NOT NULL DEFAULT 0,
    rests_today   INTEGER NOT NULL DEFAULT 0,
    rest_day      INTEGER NOT NULL DEFAULT -1,
    dead_day      INTEGER NOT NULL DEFAULT -1
);
CREATE TABLE IF NOT EXISTS inventory (
    global_id TEXT NOT NULL,
    item_id   TEXT NOT NULL,
    qty       INTEGER NOT NULL,
    PRIMARY KEY (global_id, item_id)
);
CREATE TABLE IF NOT EXISTS pvp_log (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    attacker_id   TEXT NOT NULL,
    attacker_name TEXT NOT NULL,
    victim_id     TEXT NOT NULL,
    day_index     INTEGER NOT NULL,
    attacker_won  INTEGER NOT NULL,
    loot_desc     TEXT NOT NULL,
    seen          INTEGER NOT NULL DEFAULT 0,
    created_at    INTEGER NOT NULL
);`)
	return err
}

type charState struct {
	GlobalID     string
	HP, MaxHP    int
	Strength     int
	Defense      int
	Luck         int
	Weapon       string
	Armor        string
	DepthRecord  int
	Blooded      bool
	TutorialDone bool
	RestsToday   int
	RestDay      int
	DeadDay      int
}

func isDeadToday(c *charState, turn engine.Turn) bool { return c.DeadDay == turn.DayIndex }

const restsPerDay = 3

func (g *Ledger) NewCharacter(db *sql.DB, p *engine.Player) error {
	_, err := db.Exec(
		`INSERT INTO char_state(global_id,hp,max_hp,strength,defense,luck,weapon,armor,depth_record,blooded,tutorial_done,rests_today,rest_day,dead_day)
		 VALUES(?,?,?,?,?,?,?,?,?,0,0,0,-1,-1)`,
		p.GlobalID, 20, 20, 10, 4, 5, "wpn_rusted_shiv", "arm_scavengers_wrap", 0)
	return err
}

func loadChar(db *sql.DB, id string) (*charState, error) {
	c := &charState{GlobalID: id}
	var blooded, tutorial int
	err := db.QueryRow(
		`SELECT hp,max_hp,strength,defense,luck,weapon,armor,depth_record,blooded,tutorial_done,rests_today,rest_day,dead_day
		 FROM char_state WHERE global_id=?`, id).
		Scan(&c.HP, &c.MaxHP, &c.Strength, &c.Defense, &c.Luck, &c.Weapon, &c.Armor,
			&c.DepthRecord, &blooded, &tutorial, &c.RestsToday, &c.RestDay, &c.DeadDay)
	c.Blooded = blooded != 0
	c.TutorialDone = tutorial != 0
	return c, err
}

func saveChar(db *sql.DB, c *charState) error {
	_, err := db.Exec(
		`UPDATE char_state SET hp=?,depth_record=?,blooded=?,tutorial_done=?,rests_today=?,rest_day=?,dead_day=? WHERE global_id=?`,
		c.HP, c.DepthRecord, b2i(c.Blooded), b2i(c.TutorialDone), c.RestsToday, c.RestDay, c.DeadDay, c.GlobalID)
	return err
}

type stateBlob struct {
	Char  charState `json:"char"`
	Goods []invLine `json:"goods"`
}

func (g *Ledger) ExportState(db *sql.DB, globalID string) (json.RawMessage, error) {
	c, err := loadChar(db, globalID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(stateBlob{Char: *c, Goods: listGoods(db, globalID)})
}

func (g *Ledger) ImportState(db *sql.DB, p *engine.Player, state json.RawMessage) error {
	var blob stateBlob
	if err := json.Unmarshal(state, &blob); err != nil {
		return err
	}
	c := blob.Char
	c.GlobalID = p.GlobalID
	if _, err := db.Exec(
		`INSERT INTO char_state(global_id,hp,max_hp,strength,defense,luck,weapon,armor,depth_record,blooded,tutorial_done,rests_today,rest_day,dead_day)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(global_id) DO UPDATE SET
		   hp=excluded.hp, max_hp=excluded.max_hp, strength=excluded.strength,
		   defense=excluded.defense, luck=excluded.luck, weapon=excluded.weapon,
		   armor=excluded.armor, depth_record=excluded.depth_record, blooded=excluded.blooded,
		   tutorial_done=excluded.tutorial_done, rests_today=excluded.rests_today,
		   rest_day=excluded.rest_day, dead_day=excluded.dead_day`,
		c.GlobalID, c.HP, c.MaxHP, c.Strength, c.Defense, c.Luck, c.Weapon, c.Armor,
		c.DepthRecord, b2i(c.Blooded), b2i(c.TutorialDone), c.RestsToday, c.RestDay, c.DeadDay,
	); err != nil {
		return err
	}
	for _, gd := range blob.Goods {
		if err := addGoods(db, p.GlobalID, gd.ItemID, gd.Qty); err != nil {
			return err
		}
	}
	return nil
}

func (g *Ledger) Run(ctx *engine.Context) error {
	db := ctx.Store.DB()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	c, err := loadChar(db, ctx.Player.GlobalID)
	if err != nil {
		if err := g.NewCharacter(db, ctx.Player); err != nil {
			return err
		}
		if c, err = loadChar(db, ctx.Player.GlobalID); err != nil {
			return err
		}
	}
	if reset, _ := ctx.Store.ResetIfNewDay(ctx.Player.GlobalID); reset {
		c.HP = c.MaxHP
		_ = saveChar(db, c)
	}
	if !c.TutorialDone {
		g.showArrival(ctx)
		c.TutorialDone = true
		_ = saveChar(db, c)
	}
	g.showPvpInbox(ctx, rng)

	for g.threshold(ctx, c, rng) != thresholdQuit {
	}
	ctx.Term.Clear()
	ctx.Term.Write("\r\n  You climb back toward the noise of the world. The Low keeps your place.\r\n\r\n")
	return nil
}

type thresholdResult int

const thresholdQuit thresholdResult = iota

// ---- notice helpers ----

func noticeWin(s string) string { return term.Bright(term.Green) + ">> " + term.Reset() + s }
func noticeBad(s string) string { return term.Bright(term.Red) + "!! " + term.Reset() + s }

// ---- status bar ----

// renderStatus calls statusCols with current game state values.
func renderStatus(p *engine.Player, c *charState, turn engine.Turn, debt int) (lbl, val string) {
	return statusCols(
		c.HP, c.MaxHP,
		turn.Actions, engine.MainActionsPerDay,
		turn.Attacks, engine.AttacksPerDay,
		debt, p.Level, c.DepthRecord,
		screenW,
	)
}

// ---- standard screen chrome helpers ----

// titleLine returns the colored title + subtitle without a box.
func titleLine(title, subtitle string) string {
	return term.Bright(term.Yellow) + title + term.Reset() +
		"    " + term.FG(term.Cyan) + subtitle + term.Reset()
}

// promptFooter sets row 23 to a blue rule and row 24 to a LoRD-style prompt.
// Body is capped at 22 rows so the rule always lands at exactly row 23.
func promptFooter(f *term.Frame, name string) {
	rule := term.FG(term.Blue)
	text := term.Bright(term.White)
	rs := term.Reset()
	f.Pre(rule + strings.Repeat(boxH, screenW) + rs)
	f.Status(text + "What now, " + name + "?: " + rs)
}

// pauseStatus sets a centered "press any key" rule as the status line (row 24).
func pauseStatus(f *term.Frame) {
	c, rs := term.FG(term.Blue), term.Reset()
	const msg = " press any key "
	left := (screenW - len(msg)) / 2
	right := screenW - len(msg) - left
	f.Status(c + strings.Repeat(boxH, left) + msg + strings.Repeat(boxH, right) + rs)
}

// ---- Threshold (home-base menu) ----

func (g *Ledger) threshold(ctx *engine.Context, c *charState, rng *rand.Rand) thresholdResult {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)
	mc := term.FG(term.Magenta)

	for {
		debt, _ := ctx.Store.DebtLoad(ctx.Player.GlobalID)
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
		lbl, val := renderStatus(ctx.Player, c, turn, debt)

		desc := thresholdDescs[rng.Intn(len(thresholdDescs))]
		f := term.NewFrame()
		f.Line(titleLine("THE THRESHOLD", "your bearings, such as they are"))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, desc[0], screenW))
		f.Line(boxLine(yc, desc[1], screenW))
		f.Line(boxBot(yc, screenW))
		f.Line("") // breathing room between desc and menu
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, menuItem("E", "Enter the Low", "trade, rest, and walk the Warrens"), screenW))
		f.Line(boxLine(gc, menuItem("C", "Your character", "wounds, gear, what you've become"), screenW))
		f.Line(boxLine(gc, menuItem("N", "News", "what stirred while you slept"), screenW))
		f.Line(boxLine(gc, menuItem("W", "Other wanderers", "who else is down here"), screenW))
		f.Line(boxLine(gc, menuItem("I", "How the Low works", "if you're still guessing"), screenW))
		f.Line(boxLine(gc, menuItem("V", "Travel", "move to another node"), screenW))
		f.Line(boxLine(gc, menuItem("Q", "Back to the surface", "leave for now"), screenW))
		f.Line(boxBot(gc, screenW))
		f.Line(boxTop(mc, screenW))
		f.Line(boxLine(mc, lbl, screenW))
		f.Line(boxLine(mc, val, screenW))
		f.Line(boxBot(mc, screenW))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return thresholdQuit
		}
		switch key {
		case 'e', 'E':
			g.theLow(ctx, c, rng)
		case 'c', 'C':
			g.characterScreen(ctx, c)
		case 'n', 'N':
			g.newsScreen(ctx, c)
		case 'w', 'W':
			g.wanderersScreen(ctx)
		case 'i', 'I':
			g.infoPage(ctx, "HOW THE LOW WORKS", instructionLines)
		case 'v', 'V':
			g.travelScreen(ctx)
		case 'q', 'Q':
			return thresholdQuit
		}
	}
}

// ---- The Lanternmarket (gameplay hub) ----

func (g *Ledger) theLow(ctx *engine.Context, c *charState, rng *rand.Rand) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)
	mc := term.FG(term.Magenta)
	notice := term.FG(term.Cyan) + pickRandom(rng, theLowAmbient) + term.Reset()

	for {
		debt, _ := ctx.Store.DebtLoad(ctx.Player.GlobalID)
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
		lbl, val := renderStatus(ctx.Player, c, turn, debt)

		f := term.NewFrame()
		f.Line(titleLine("THE LANTERNMARKET", "where the living gets done"))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, notice, screenW))
		f.Line(boxBot(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, "", screenW)) // breathing room
		f.Line(boxLine(gc, menuItem("E", "Explore the Warrens (shallow)", "-1 expedition"), screenW))
		f.Line(boxLine(gc, menuItem("D", "Explore the Warrens (deep)", "-1 expedition, worse & richer"), screenW))
		f.Line(boxLine(gc, menuItem("T", "Trade with Maren", "free"), screenW))
		f.Line(boxLine(gc, menuItem("R", "Rest", "free, mends a little"), screenW))
		f.Line(boxLine(gc, menuItem("B", "See the Bonesetter", "full mend, goods or tab"), screenW))
		f.Line(boxLine(gc, menuItem("A", "Attack a sleeping rival", "uses 1 attack"), screenW))
		f.Line(boxLine(gc, menuItem("L", "Back to the Threshold", ""), screenW))
		f.Line(boxLine(gc, "", screenW)) // breathing room
		f.Line(boxBot(gc, screenW))
		f.Line(boxTop(mc, screenW))
		f.Line(boxLine(mc, lbl, screenW))
		f.Line(boxLine(mc, val, screenW))
		f.Line(boxBot(mc, screenW))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch key {
		case 'e', 'E':
			t2, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
			if isDeadToday(c, t2) {
				notice = noticeBad(" The Warrens took you today. Rest, trade, or come back tomorrow.")
			} else {
				notice = " " + g.explore(ctx, c, false, rng)
			}
		case 'd', 'D':
			t2, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
			if isDeadToday(c, t2) {
				notice = noticeBad(" The Warrens took you today. Rest, trade, or come back tomorrow.")
			} else {
				notice = " " + g.explore(ctx, c, true, rng)
			}
		case 't', 'T':
			notice = " " + g.trade(ctx, c, rng)
		case 'r', 'R':
			notice = " " + g.rest(ctx, c)
		case 'b', 'B':
			notice = " " + g.bonesetter(ctx, c, rng)
		case 'a', 'A':
			t2, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
			if isDeadToday(c, t2) {
				notice = noticeBad(" The Warrens took you today. You're in no state for that.")
			} else {
				notice = " " + g.attackMenu(ctx, c, rng)
			}
		case 'l', 'L':
			return
		}
	}
}

// ---- info screens ----

// infoPage renders a title + lines inside a yellow box, then waits for a key.
func (g *Ledger) infoPage(ctx *engine.Context, title string, lines []string) {
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

// pageScreen is a backward-compatible wrapper around infoPage.
func (g *Ledger) pageScreen(ctx *engine.Context, lines []string) {
	title := ""
	if len(lines) > 0 {
		title = lines[0]
		lines = lines[1:]
	}
	g.infoPage(ctx, title, lines)
}

func (g *Ledger) showArrival(ctx *engine.Context) {
	g.infoPage(ctx, "YOU WAKE IN THE DARK", arrivalLines)
}

func (g *Ledger) characterScreen(ctx *engine.Context, c *charState) {
	debt, _ := ctx.Store.DebtLoad(ctx.Player.GlobalID)
	worth := goodsValue(ctx.Store.DB(), ctx.Player.GlobalID)
	standing := "untested by the dark"
	if c.Blooded {
		standing = "blooded -- the Warrens know you now"
	}
	g.infoPage(ctx, strings.ToUpper(ctx.Player.Name), []string{
		"",
		fmt.Sprintf("Health      %d / %d", c.HP, c.MaxHP),
		fmt.Sprintf("Strength    %d", c.Strength),
		fmt.Sprintf("Defense     %d", c.Defense),
		fmt.Sprintf("Luck        %d", c.Luck),
		fmt.Sprintf("Level       %d", ctx.Player.Level),
		"",
		fmt.Sprintf("Weapon      %s", itemName(c.Weapon)),
		fmt.Sprintf("Armor       %s", itemName(c.Armor)),
		"",
		fmt.Sprintf("Deepest reached    band %d", c.DepthRecord),
		fmt.Sprintf("Scavenge worth     %d in barter", worth),
		fmt.Sprintf("Owed to the Low    %d", debt),
		fmt.Sprintf("Standing           %s", standing),
	})
}

func (g *Ledger) newsScreen(ctx *engine.Context, c *charState) {
	db := ctx.Store.DB()
	rows, _ := db.Query(
		`SELECT attacker_name,attacker_won,loot_desc FROM pvp_log WHERE victim_id=? ORDER BY created_at DESC LIMIT 5`,
		ctx.Player.GlobalID)
	var lines []string
	had := false
	if rows != nil {
		for rows.Next() {
			var name, loot string
			var won int
			if rows.Scan(&name, &won, &loot) == nil {
				had = true
				if won != 0 {
					lines = append(lines, term.Bright(term.Red)+name+" caught you sleeping and took "+loot+"."+term.Reset())
				} else {
					lines = append(lines, term.Bright(term.Green)+name+" came for you and left empty-handed."+term.Reset())
				}
			}
		}
		rows.Close()
	}
	if !had {
		lines = append(lines, "No one has troubled your sleep. Yet.")
	}
	lines = append(lines, "")
	if debt, _ := ctx.Store.DebtLoad(ctx.Player.GlobalID); debt > 0 {
		lines = append(lines, fmt.Sprintf(
			term.FG(term.Yellow)+"The Ledger notes you owe %d. It is patient. It is always patient."+term.Reset(), debt))
	} else {
		lines = append(lines, "You owe nothing today. A rare and weightless feeling.")
	}
	g.infoPage(ctx, "THE NEWS, SUCH AS IT TRAVELS", lines)
}

func (g *Ledger) wanderersScreen(ctx *engine.Context) {
	others, _ := ctx.Store.Players(ctx.Player.GlobalID)
	var lines []string
	if len(others) == 0 {
		lines = append(lines, "No one else walks the Low tonight. Or none you can sense.")
	}
	for i, o := range others {
		if i >= 16 {
			lines = append(lines, fmt.Sprintf("...and %d more.", len(others)-16))
			break
		}
		lines = append(lines, fmt.Sprintf("%-18s Lv %d   %s%s",
			o.Name, o.Level, o.Status, remoteRosterNote(ctx.NodeID, o.HomeNode, o.LastSeen, time.Now())))
	}
	g.infoPage(ctx, "OTHER WANDERERS", lines)
}

func remoteRosterNote(nodeID, homeNode string, lastSeen, now time.Time) string {
	if homeNode == "" || homeNode == nodeID {
		return ""
	}
	details := []string{"from " + homeNode}
	if !lastSeen.IsZero() {
		age := now.Sub(lastSeen)
		if age < 0 {
			age = 0
		}
		if age > remoteRosterStaleAfter {
			details = append(details, "stale")
		}
		details = append(details, "seen "+formatRosterAge(age)+" ago")
	}
	return "   " + term.FG(term.Cyan) + "(" + strings.Join(details, ", ") + ")" + term.Reset()
}

func formatRosterAge(age time.Duration) string {
	age = age.Round(time.Minute)
	if age < time.Minute {
		return "just now"
	}
	if age < time.Hour {
		return fmt.Sprintf("%dm", int(age/time.Minute))
	}
	return fmt.Sprintf("%dh", int(age/time.Hour))
}

// ---- mending ----

func (g *Ledger) rest(ctx *engine.Context, c *charState) string {
	turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
	if c.RestDay != turn.DayIndex {
		c.RestDay = turn.DayIndex
		c.RestsToday = 0
	}
	if c.HP >= c.MaxHP {
		return "You're as whole as the Low will let you be."
	}
	if c.RestsToday >= restsPerDay {
		return noticeBad("You've rested all you can today. The Bonesetter waits, for a price.")
	}
	c.RestsToday++
	c.HP += c.MaxHP * 30 / 100
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	_ = saveChar(ctx.Store.DB(), c)
	return noticeWin("You sit in the lantern-light. The ache settles to something you can carry.")
}

func (g *Ledger) bonesetter(ctx *engine.Context, c *charState, rng *rand.Rand) string {
	if c.HP >= c.MaxHP {
		return pickRandom(rng, bonesetterHealthy)
	}
	db := ctx.Store.DB()
	gid := ctx.Player.GlobalID
	const fee = 15
	owed := 0
	if val := goodsValue(db, gid); val >= fee {
		consumeForCost(db, gid, fee)
	} else {
		consumeForCost(db, gid, val)
		owed = fee - val
		_, _ = ctx.Store.CreateObligation("npc:npc_bonesetter", gid, "debt",
			"a mending from the Bonesetter", owed)
	}
	c.HP = c.MaxHP
	_ = saveChar(db, c)
	if owed > 0 {
		return noticeWin(fmt.Sprintf(
			"The Bonesetter mends you whole and writes %d to your tab.", owed))
	}
	return noticeWin("The Bonesetter mends you whole, paid in scavenge.")
}

// ---- exploration and combat ----

func (g *Ledger) explore(ctx *engine.Context, c *charState, deep bool, rng *rand.Rand) string {
	if err := ctx.Store.SpendActions(ctx.Player.GlobalID, 1); err != nil {
		return noticeBad("No expeditions left in you today. Rest, trade, or come back tomorrow.")
	}
	band := 1
	if deep {
		band = c.DepthRecord + 1
		if band < 2 {
			band = 2
		}
	}
	creature := creatureForBand(band, rng.Intn)
	outcome := g.fight(ctx, c, creature, band, rng)
	c.Blooded = true

	switch outcome {
	case fightWon:
		good := lootDrops[rng.Intn(len(lootDrops))]
		_ = addGood(ctx.Store.DB(), ctx.Player.GlobalID, good)
		deeper := false
		if deep && band > c.DepthRecord {
			c.DepthRecord = band
			deeper = true
		}
		_ = saveChar(ctx.Store.DB(), c)
		g.showFightWin(ctx, creature, good, deeper, band, rng)
		return noticeWin("Back among the lanterns, you take stock.")
	case fightFled:
		_ = saveChar(ctx.Store.DB(), c)
		g.showFightFled(ctx, creature, rng)
		return "You catch your breath among the lanterns."
	default:
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
		c.DeadDay = turn.DayIndex
		_ = saveChar(ctx.Store.DB(), c)
		_ = ctx.Store.SpendActions(ctx.Player.GlobalID, turn.Actions)
		g.showFightDeath(ctx, creature, rng)
		return noticeBad("The Low has claimed your day. Rest, trade, or come back tomorrow.")
	}
}

// resultPage renders a full-screen result before returning to the menu.
func (g *Ledger) resultPage(ctx *engine.Context, headerColor, headerText string, lines []string) {
	f := term.NewFrame()
	f.Line(boxTop(headerColor, screenW))
	f.Line(boxLine(headerColor, " "+headerText, screenW))
	f.Line(boxBot(headerColor, screenW))
	f.Line("")
	for _, ln := range lines {
		f.Line("  " + ln)
	}
	pauseStatus(f)
	f.Render(ctx.Term)
	_, _ = ctx.Term.ReadKey()
}

func (g *Ledger) showFightWin(ctx *engine.Context, cr Creature, good string, deeper bool, band int, rng *rand.Rand) {
	killLine := term.Bright(term.Green) + fmt.Sprintf(pickRandom(rng, fightWinLines), cr.Name) + term.Reset()
	flavor := term.FG(term.Cyan) + pickRandom(rng, fightWinFlavors) + term.Reset()
	lines := []string{
		killLine,
		"",
		"You search what's left and pocket:",
		"    " + term.Bright(term.Yellow) + "1x " + goodName(good) + term.Reset(),
		"",
		flavor,
	}
	if deeper {
		lines = append(lines, "", term.Bright(term.Cyan)+
			fmt.Sprintf("Deeper than you have been.  Band %d.  The Ledger notes it.", band)+term.Reset())
	}
	g.resultPage(ctx, term.Bright(term.Red), "THE WARRENS", lines)
}

func (g *Ledger) showFightFled(ctx *engine.Context, cr Creature, rng *rand.Rand) {
	g.resultPage(ctx, term.Bright(term.Red), "THE WARRENS", []string{
		fmt.Sprintf(pickRandom(rng, fleeSuccessLines), cr.Name),
	})
}

func (g *Ledger) showFightDeath(ctx *engine.Context, cr Creature, rng *rand.Rand) {
	killLine := term.Bright(term.Red) + fmt.Sprintf(pickRandom(rng, fightDeathLines), cr.Name) + term.Reset()
	wakeLine := term.FG(term.Yellow) + pickRandom(rng, wakeAtThresholdLines) + term.Reset()
	flavorLine := pickRandom(rng, thresholdWakeFlavors)
	g.resultPage(ctx, term.Bright(term.Red), "THE WARRENS", []string{
		killLine,
		"",
		wakeLine,
		flavorLine,
	})
}

type fightResult int

const (
	fightWon fightResult = iota
	fightFled
	fightDied
)

func (g *Ledger) fight(ctx *engine.Context, c *charState, cr Creature, band int, rng *rand.Rand) fightResult {
	t := ctx.Term
	enemyHP := cr.HP
	atkPow := c.Strength + itemBonus(c.Weapon)
	defPow := c.Defense + itemBonus(c.Armor)
	rc := term.Bright(term.Red)
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)

	combatLog := []string{term.FG(term.Cyan) + cr.Intro + term.Reset()}

	for {
		f := term.NewFrame()
		// Header box (red)
		f.Line(boxTop(rc, screenW))
		f.Line(boxLine(rc, fmt.Sprintf("  THE WARRENS, BAND %d", band), screenW))
		f.Line(boxBot(rc, screenW))
		f.Line("")
		// HP box (yellow)
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, fmtHP("You", c.HP, c.MaxHP), screenW))
		f.Line(boxLine(yc, fmtHP(cr.Name, enemyHP, cr.HP), screenW))
		f.Line(boxBot(yc, screenW))
		f.Line("")
		// Combat log -- always 4 lines for stable layout
		for i := 0; i < 4; i++ {
			if i < len(combatLog) {
				f.Line("  " + combatLog[i])
			} else {
				f.Line("")
			}
		}
		f.Line("")
		// Action box (green)
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, "    [A] Strike    [F] Flee", screenW))
		f.Line(boxBot(gc, screenW))
		f.Status(fmt.Sprintf("  HP %d/%d   Depth band %d", c.HP, c.MaxHP, band))
		f.Render(t)

		key, err := t.ReadKey()
		if err != nil {
			return fightFled
		}
		switch key {
		case 'a', 'A':
			dmg, crit := Damage(atkPow, cr.Defense, c.Luck, rng)
			enemyHP -= dmg
			var entry string
			if crit {
				entry = term.Bright(term.Yellow) + "CRITICAL!  " + pickRandom(rng, attackCrits) +
					fmt.Sprintf("  [%d dmg]", dmg) + term.Reset()
			} else {
				entry = term.FG(term.Green) + pickRandom(rng, attackHits) +
					fmt.Sprintf("  [%d dmg]", dmg) + term.Reset()
			}
			combatLog = appendCombatLog(combatLog, entry, 4)
			if enemyHP <= 0 {
				return fightWon
			}
			cdmg, _ := Damage(cr.Strength, defPow, 0, rng)
			c.HP -= cdmg
			combatLog = appendCombatLog(combatLog,
				term.FG(term.Red)+pickRandom(rng, creatureHits)+fmt.Sprintf("  [%d dmg]", cdmg)+term.Reset(), 4)
			if c.HP <= 0 {
				c.HP = 1
				g.die(ctx, cr, band)
				return fightDied
			}
		case 'f', 'F':
			if rng.Intn(100) < fleeChance(c.Luck) {
				return fightFled
			}
			cdmg, _ := Damage(cr.Strength, defPow, 0, rng)
			c.HP -= cdmg
			combatLog = appendCombatLog(combatLog,
				term.FG(term.Red)+pickRandom(rng, fleeHits)+fmt.Sprintf("  [%d dmg]", cdmg)+term.Reset(), 4)
			if c.HP <= 0 {
				c.HP = 1
				g.die(ctx, cr, band)
				return fightDied
			}
		}
	}
}

func (g *Ledger) die(ctx *engine.Context, cr Creature, band int) {
	_ = ctx.Store.Emit("player.died", map[string]any{
		"global_id": ctx.Player.GlobalID,
		"cause": map[string]any{
			"type": "creature", "source_ref": cr.ID,
			"district": "dst_warrens", "depth": band,
		},
		"timestamp": time.Now().Unix(),
	})
}

// ---- Maren's stall ----

func (g *Ledger) trade(ctx *engine.Context, c *charState, rng *rand.Rand) string {
	db := ctx.Store.DB()
	gid := ctx.Player.GlobalID
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)
	notice := pickRandom(rng, marenGreetings)

	for {
		debt, _ := ctx.Store.DebtLoad(gid)
		val := goodsValue(db, gid)

		f := term.NewFrame()
		f.Line(titleLine("MAREN'S STALL", "trade and the weight of what you carry"))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, " "+notice, screenW))
		f.Line(boxBot(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, "  For sale (barter units):", screenW))
		f.Line(boxLine(gc, "", screenW))
		for i, w := range marenWares {
			f.Line(boxLine(gc, fmt.Sprintf("    [%d]  %-24s  %d", i+1, itemName(w.ItemID), w.Price), screenW))
		}
		f.Line(boxLine(gc, "", screenW))
		f.Line(boxLine(gc, fmt.Sprintf("  Your scavenge is worth %d in barter.", val), screenW))
		if debt > 0 {
			f.Line(boxLine(gc,
				term.FG(term.Yellow)+fmt.Sprintf("  You owe Maren %d.  [P] pay it down with goods.", debt)+term.Reset(),
				screenW))
		}
		f.Line(boxLine(gc, "", screenW))
		f.Line(boxLine(gc, "    [1-9] Buy     [P] Pay debt     [B] Back to the market", screenW))
		f.Line(boxBot(gc, screenW))
		f.Line(boxTop(term.FG(term.Magenta), screenW))
		f.Line(boxLine(term.FG(term.Magenta), fmt.Sprintf("  Barter %d   Debt %d", val, debt), screenW))
		f.Line(boxBot(term.FG(term.Magenta), screenW))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return "You step away from the stall."
		}
		switch {
		case key >= '1' && key <= '9':
			if idx := int(key - '1'); idx < len(marenWares) {
				notice = g.buy(ctx, c, marenWares[idx])
			}
		case key == 'p' || key == 'P':
			notice = g.payDebt(ctx)
		case key == 'b' || key == 'B':
			return "You leave Maren to her counting."
		}
	}
}

func (g *Ledger) buy(ctx *engine.Context, c *charState, w Ware) string {
	db := ctx.Store.DB()
	gid := ctx.Player.GlobalID
	consumed := consumeForCost(db, gid, w.Price)
	owed := 0
	if consumed < w.Price {
		owed = w.Price - consumed
		_, _ = ctx.Store.CreateObligation("npc:npc_maren", gid, "debt",
			fmt.Sprintf("%s advanced by Maren", itemName(w.ItemID)), owed)
	}
	switch w.Slot {
	case "weapon":
		c.Weapon = w.ItemID
	case "armor":
		c.Armor = w.ItemID
	}
	_, _ = db.Exec(`UPDATE char_state SET weapon=?, armor=? WHERE global_id=?`, c.Weapon, c.Armor, gid)
	if owed > 0 {
		return fmt.Sprintf("You take the %s. You owe Maren %d more. She writes it down.", itemName(w.ItemID), owed)
	}
	return noticeWin(fmt.Sprintf("You take the %s, paid in scavenge.", itemName(w.ItemID)))
}

func (g *Ledger) payDebt(ctx *engine.Context) string {
	db := ctx.Store.DB()
	gid := ctx.Player.GlobalID
	debt, _ := ctx.Store.DebtLoad(gid)
	if debt == 0 {
		return "You owe Maren nothing. She seems almost disappointed."
	}
	val := goodsValue(db, gid)
	if val == 0 {
		return noticeBad("You've nothing to pay with. She's heard that before.")
	}
	target := debt
	if val < target {
		target = val
	}
	consumed := consumeForCost(db, gid, target)
	applied, _ := ctx.Store.PayDebt(gid, consumed)
	return noticeWin(fmt.Sprintf("You hand over goods. %d of what you owed is settled.", applied))
}

// ---- inventory ----

type invLine struct {
	ItemID string
	Qty    int
}

type pvpLootItem struct {
	ItemID string `json:"item_id"`
	Qty    int    `json:"qty"`
}

func addGood(db *sql.DB, gid, itemID string) error { return addGoods(db, gid, itemID, 1) }

func addGoods(db *sql.DB, gid, itemID string, n int) error {
	if n <= 0 {
		return nil
	}
	_, err := db.Exec(
		`INSERT INTO inventory(global_id,item_id,qty) VALUES(?,?,?)
		 ON CONFLICT(global_id,item_id) DO UPDATE SET qty=qty+?`, gid, itemID, n, n)
	return err
}

func listGoods(db *sql.DB, gid string) []invLine {
	rows, err := db.Query(`SELECT item_id,qty FROM inventory WHERE global_id=? AND qty>0`, gid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []invLine
	for rows.Next() {
		var l invLine
		if err := rows.Scan(&l.ItemID, &l.Qty); err == nil {
			out = append(out, l)
		}
	}
	return out
}

func goodsValue(db *sql.DB, gid string) int {
	total := 0
	for _, l := range listGoods(db, gid) {
		total += l.Qty * goodWeight(l.ItemID)
	}
	return total
}

func consumeForCost(db *sql.DB, gid string, cost int) int {
	gs := listGoods(db, gid)
	sort.Slice(gs, func(i, j int) bool { return goodWeight(gs[i].ItemID) < goodWeight(gs[j].ItemID) })
	consumed := 0
	for _, l := range gs {
		w := goodWeight(l.ItemID)
		for n := 0; n < l.Qty && consumed < cost; n++ {
			_, _ = db.Exec(`UPDATE inventory SET qty=qty-1 WHERE global_id=? AND item_id=?`, gid, l.ItemID)
			consumed += w
		}
	}
	_, _ = db.Exec(`DELETE FROM inventory WHERE global_id=? AND qty<=0`, gid)
	return consumed
}

// ---- PvP ----

func serverDay() int { return int(time.Now().UTC().Unix() / 86400) }

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (g *Ledger) attackMenu(ctx *engine.Context, c *charState, rng *rand.Rand) string {
	others, _ := ctx.Store.Players(ctx.Player.GlobalID)
	if len(others) == 0 {
		return "No one else walks the Low tonight. You keep your edge sheathed."
	}
	rc := term.Bright(term.Red)
	gc := term.Bright(term.Green)
	mc := term.FG(term.Magenta)
	notice := `"Everyone sleeps eventually," Maren said once. "That's the whole trouble."`

	for {
		limit := len(others)
		if limit > 9 {
			limit = 9
		}
		turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)

		f := term.NewFrame()
		f.Line(titleLine("WHO SLEEPS", "the weight of a sleeping rival"))
		f.Line(hRule(rc, screenW))
		f.Line(boxTop(rc, screenW))
		f.Line(boxLine(rc, " "+notice, screenW))
		f.Line(boxBot(rc, screenW))
		f.Line(boxTop(gc, screenW))
		for i := 0; i < limit; i++ {
			o := others[i]
			tag := term.Bright(term.Green) + "vulnerable" + term.Reset()
			if ok, reason := g.canAttackVictim(ctx, o); !ok {
				tag = term.FG(term.Cyan) + reason + term.Reset()
			}
			remote := ""
			if o.HomeNode != ctx.NodeID {
				remote = " (remote)"
			}
			f.Line(boxLine(gc, fmt.Sprintf("    [%d] %-16s Lv %d   %s%s", i+1, o.Name, o.Level, tag, remote), screenW))
		}
		f.Line(boxLine(gc, "", screenW))
		f.Line(boxLine(gc, "    [1-9] Attack     [B] Back to the market", screenW))
		f.Line(boxBot(gc, screenW))
		f.Line(boxTop(mc, screenW))
		f.Line(boxLine(mc, fmt.Sprintf("  Attacks left: %d/%d", turn.Attacks, engine.AttacksPerDay), screenW))
		f.Line(boxBot(mc, screenW))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return "You think better of it."
		}
		switch {
		case key >= '1' && key <= '9':
			if idx := int(key - '1'); idx < limit {
				if ok, reason := g.canAttackVictim(ctx, others[idx]); !ok {
					notice = noticeBad("You can't: " + reason + ".")
					continue
				}
				if others[idx].HomeNode != ctx.NodeID {
					return g.resolvePvPRemote(ctx, c, others[idx])
				}
				return g.resolvePvP(ctx, c, others[idx], rng)
			}
		case key == 'b' || key == 'B':
			return "You let them sleep. For now."
		}
	}
}

func (g *Ledger) canAttackVictim(ctx *engine.Context, victim engine.Player) (bool, string) {
	db := ctx.Store.DB()
	remote := victim.HomeNode != ctx.NodeID
	if !remote {
		vc, err := loadChar(db, victim.GlobalID)
		if err != nil {
			return false, "no quarry"
		}
		if !vc.Blooded {
			return false, "untested"
		}
	}
	if victim.Level < ctx.Player.Level-3 {
		return false, "beneath you"
	}
	today := serverDay()
	var last int
	_ = db.QueryRow(`SELECT COALESCE(MAX(day_index),-999) FROM pvp_log WHERE attacker_id=? AND victim_id=?`,
		ctx.Player.GlobalID, victim.GlobalID).Scan(&last)
	if last >= today-1 {
		return false, "too soon"
	}
	if !remote {
		var hits int
		_ = db.QueryRow(`SELECT COUNT(*) FROM pvp_log WHERE victim_id=? AND day_index=?`,
			victim.GlobalID, today).Scan(&hits)
		if hits >= 3 {
			return false, "already hunted"
		}
	}
	return true, ""
}

func (g *Ledger) resolvePvP(ctx *engine.Context, c *charState, victim engine.Player, rng *rand.Rand) string {
	if err := ctx.Store.SpendAttack(ctx.Player.GlobalID); err != nil {
		return noticeBad("You're out of attacks for today.")
	}
	db := ctx.Store.DB()
	vc, err := loadChar(db, victim.GlobalID)
	if err != nil {
		return "Your quarry has slipped the Low entirely."
	}
	aHP, vHP := c.HP, vc.HP
	aAtk, aDef := c.Strength+itemBonus(c.Weapon), c.Defense+itemBonus(c.Armor)
	vAtk, vDef := vc.Strength+itemBonus(vc.Weapon), vc.Defense+itemBonus(vc.Armor)
	won := false
	for round := 0; round < 30; round++ {
		d, _ := Damage(aAtk, vDef, c.Luck, rng)
		if vHP -= d; vHP <= 0 {
			won = true
			break
		}
		d2, _ := Damage(vAtk, aDef, vc.Luck, rng)
		if aHP -= d2; aHP <= 0 {
			won = false
			break
		}
	}
	if aHP > 0 && vHP > 0 {
		won = float64(aHP)/float64(c.MaxHP) > float64(vHP)/float64(vc.MaxHP)
	}
	if aHP < 1 {
		aHP = 1
	}
	if vHP < 1 {
		vHP = 1
	}
	c.HP, vc.HP = aHP, vHP
	_ = saveChar(db, c)
	_ = saveChar(db, vc)
	lootDesc := "nothing"
	var lootItems []pvpLootItem
	if won {
		lootDesc, lootItems = transferLoot(db, victim.GlobalID, ctx.Player.GlobalID, rng)
	}
	if lootItems == nil {
		lootItems = []pvpLootItem{}
	}
	_, _ = db.Exec(
		`INSERT INTO pvp_log(attacker_id,attacker_name,victim_id,day_index,attacker_won,loot_desc,seen,created_at)
		 VALUES(?,?,?,?,?,?,0,?)`,
		ctx.Player.GlobalID, ctx.Player.Name, victim.GlobalID, serverDay(), b2i(won), lootDesc, time.Now().Unix())
	outcome := "loss"
	if won {
		outcome = "win"
	}
	winner := victim.GlobalID
	if won {
		winner = ctx.Player.GlobalID
	}
	resolvedAt := time.Now().Unix()
	_ = ctx.Store.Emit("pvp.resolved", map[string]any{
		"request_id":         fmt.Sprintf("local:%s:%s:%d", ctx.Player.GlobalID, victim.GlobalID, resolvedAt),
		"attacker_global_id": ctx.Player.GlobalID,
		"victim_global_id":   victim.GlobalID,
		"winner_global_id":   winner,
		"result_text":        outcome,
		"resolved_at":        resolvedAt,
		"outcome":            outcome,
		"loot":               lootItems,
		"victim_died":        false,
		"day_index":          serverDay(),
	})
	g.showPvpResult(ctx, victim.Name, won, lootDesc)
	if won {
		return noticeWin("You melt back into the Lanternmarket crowd.")
	}
	return noticeBad("You nurse your pride among the lanterns.")
}

func (g *Ledger) showPvpResult(ctx *engine.Context, name string, won bool, loot string) {
	var lines []string
	if won {
		lines = []string{
			term.Bright(term.Green) + "You catch " + name + " sleeping and leave them the worse for it." + term.Reset(),
			"",
			"You take:",
			"    " + term.Bright(term.Yellow) + loot + term.Reset(),
		}
	} else {
		lines = []string{
			term.Bright(term.Red) + name + " wakes and gives better than they get." + term.Reset(),
			"",
			"You slink off with nothing.",
		}
	}
	g.resultPage(ctx, term.Bright(term.Red), "IN THE DARK", lines)
}

func stealLoot(db *sql.DB, victimID string, rng *rand.Rand) (string, []pvpLootItem) {
	frac := 0.25 + rng.Float64()*0.25
	goods := listGoods(db, victimID)
	sort.Slice(goods, func(i, j int) bool { return goodWeight(goods[i].ItemID) > goodWeight(goods[j].ItemID) })
	var parts []string
	var items []pvpLootItem
	take := func(itemID string, n int) {
		_, _ = db.Exec(`UPDATE inventory SET qty=qty-? WHERE global_id=? AND item_id=?`, n, victimID, itemID)
		parts = append(parts, fmt.Sprintf("%dx %s", n, goodName(itemID)))
		items = append(items, pvpLootItem{ItemID: itemID, Qty: n})
	}
	took := 0
	for _, l := range goods {
		if n := int(math.Round(float64(l.Qty) * frac)); n > 0 {
			take(l.ItemID, n)
			took += n
		}
	}
	if took == 0 && len(goods) > 0 {
		take(goods[0].ItemID, 1)
	}
	_, _ = db.Exec(`DELETE FROM inventory WHERE global_id=? AND qty<=0`, victimID)
	if len(parts) == 0 {
		return "nothing worth carrying", nil
	}
	return strings.Join(parts, ", "), items
}

func transferLoot(db *sql.DB, victimID, attackerID string, rng *rand.Rand) (string, []pvpLootItem) {
	desc, items := stealLoot(db, victimID, rng)
	for _, item := range items {
		_ = addGoods(db, attackerID, item.ItemID, item.Qty)
	}
	return desc, items
}

// ---- travel ----

func (g *Ledger) travelScreen(ctx *engine.Context) {
	yc := term.Bright(term.Yellow)
	if ctx.Travel == nil {
		g.infoPage(ctx, "TRAVEL", []string{
			"This node is not connected to the InterDOOR network.",
			"Travel between nodes is not available here.",
		})
		return
	}
	if ctx.Player.HomeNode != ctx.NodeID {
		g.departScreen(ctx)
		return
	}
	others, _ := ctx.Store.Players(ctx.Player.GlobalID)
	seen := map[string]bool{}
	var nodes []string
	for _, o := range others {
		if o.HomeNode != ctx.NodeID && !seen[o.HomeNode] {
			seen[o.HomeNode] = true
			nodes = append(nodes, o.HomeNode)
		}
	}
	if len(nodes) == 0 {
		g.infoPage(ctx, "TRAVEL", []string{
			"No other nodes are known to this one yet.",
			"When wanderers from other nodes appear here, their home becomes reachable.",
		})
		return
	}
	gc := term.Bright(term.Green)
	notice := "The road between nodes is the longest road there is."
	for {
		limit := len(nodes)
		if limit > 9 {
			limit = 9
		}
		f := term.NewFrame()
		f.Line(titleLine("TRAVEL", "the road between nodes"))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, " "+notice, screenW))
		f.Line(boxBot(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, "  Known nodes:", screenW))
		f.Line(boxLine(gc, "", screenW))
		for i := 0; i < limit; i++ {
			f.Line(boxLine(gc, fmt.Sprintf("    [%d]  %s", i+1, nodes[i]), screenW))
		}
		f.Line(boxLine(gc, "", screenW))
		f.Line(boxLine(gc, "    [1-9] Depart     [B] Stay", screenW))
		f.Line(boxBot(gc, screenW))
		promptFooter(f, ctx.Player.Name)
		f.Render(ctx.Term)

		key, err := ctx.Term.ReadKey()
		if err != nil {
			return
		}
		switch {
		case key >= '1' && key <= '9':
			if idx := int(key - '1'); idx < limit {
				if err := ctx.Travel(ctx.Player.GlobalID, nodes[idx]); err != nil {
					notice = noticeBad("The road is closed: " + err.Error())
				} else {
					g.showTraveling(ctx, nodes[idx])
					return
				}
			}
		case key == 'b' || key == 'B':
			return
		}
	}
}

func (g *Ledger) departScreen(ctx *engine.Context) {
	yc := term.Bright(term.Yellow)
	gc := term.Bright(term.Green)
	notice := "You are visiting " + ctx.NodeID + ". Your home is " + ctx.Player.HomeNode + "."
	for {
		f := term.NewFrame()
		f.Line(titleLine("VISITING "+strings.ToUpper(ctx.NodeID), "a node not your own"))
		f.Line(hRule(yc, screenW))
		f.Line(boxTop(yc, screenW))
		f.Line(boxLine(yc, " "+notice, screenW))
		f.Line(boxBot(yc, screenW))
		f.Line(boxTop(gc, screenW))
		f.Line(boxLine(gc, menuItem("R", "Return to "+ctx.Player.HomeNode, ""), screenW))
		f.Line(boxLine(gc, menuItem("B", "Stay here for now", ""), screenW))
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
				notice = noticeBad("The road home is closed: " + err.Error())
			} else {
				g.showTraveling(ctx, ctx.Player.HomeNode)
				return
			}
		case 'b', 'B':
			return
		}
	}
}

func (g *Ledger) showTraveling(ctx *engine.Context, destNode string) {
	g.resultPage(ctx, term.FG(term.Cyan), "DEPARTING", []string{
		"You gather what you carry and set out for " + destNode + ".",
		"",
		"Your place in the Low will hold. The hub carries your record ahead of you.",
		"Log in at " + destNode + " when you are ready.",
	})
}

// ---- cross-node PvP ----

type attackerPayload struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	Strength int    `json:"strength"`
	Defense  int    `json:"defense"`
	Luck     int    `json:"luck"`
	Weapon   string `json:"weapon"`
	Armor    string `json:"armor"`
}

func (g *Ledger) resolvePvPRemote(ctx *engine.Context, c *charState, victim engine.Player) string {
	if ctx.CrossNodeAttack == nil {
		return noticeBad("The hub is unreachable. Your attack can't be routed.")
	}
	if err := ctx.Store.SpendAttack(ctx.Player.GlobalID); err != nil {
		return noticeBad("You're out of attacks for today.")
	}
	payload, _ := json.Marshal(attackerPayload{
		Name: ctx.Player.Name, Level: ctx.Player.Level,
		HP: c.HP, MaxHP: c.MaxHP, Strength: c.Strength, Defense: c.Defense,
		Luck: c.Luck, Weapon: c.Weapon, Armor: c.Armor,
	})
	_, err := ctx.CrossNodeAttack(ctx.Player.GlobalID, victim.GlobalID, payload)
	if err != nil {
		log.Printf("cross-node pvp %s->%s: %v", ctx.Player.GlobalID, victim.GlobalID, err)
		return "Your attack was dispatched but the network stumbled. The Low will sort it out."
	}
	db := ctx.Store.DB()
	_, _ = db.Exec(
		`INSERT INTO pvp_log(attacker_id,attacker_name,victim_id,day_index,attacker_won,loot_desc,seen,created_at)
		 VALUES(?,?,?,?,0,'pending',0,?)`,
		ctx.Player.GlobalID, ctx.Player.Name, victim.GlobalID, serverDay(), time.Now().Unix())
	g.resultPage(ctx, term.FG(term.Cyan), "IN THE DARK", []string{
		"Your move against " + victim.Name + " has been relayed across the Low.",
		"",
		term.FG(term.Cyan) + "The result will find you in the news when it settles." + term.Reset(),
	})
	return "The Low carries your intent. The answer comes when it comes."
}

func (g *Ledger) showPvpInbox(ctx *engine.Context, rng *rand.Rand) {
	db := ctx.Store.DB()
	rows, err := db.Query(
		`SELECT attacker_name,attacker_won,loot_desc FROM pvp_log WHERE victim_id=? AND seen=0 ORDER BY created_at`,
		ctx.Player.GlobalID)
	if err != nil {
		return
	}
	type hit struct {
		name string
		won  int
		loot string
	}
	var hits []hit
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.name, &h.won, &h.loot); err == nil {
			hits = append(hits, h)
		}
	}
	rows.Close()
	if len(hits) == 0 {
		return
	}
	var lines []string
	for _, h := range hits {
		if h.won != 0 {
			lines = append(lines, term.Bright(term.Red)+
				fmt.Sprintf("%s found you sleeping and took %s.", h.name, h.loot)+term.Reset())
		} else {
			lines = append(lines, term.Bright(term.Green)+
				fmt.Sprintf("%s came for you in the dark, and you saw them off.", h.name)+term.Reset())
		}
	}
	lines = append(lines, "", "The Low keeps its accounts.")
	g.resultPage(ctx, term.Bright(term.Red), "WHILE YOU SLEPT", lines)
	_, _ = db.Exec(`UPDATE pvp_log SET seen=1 WHERE victim_id=? AND seen=0`, ctx.Player.GlobalID)
}

func (g *Ledger) IncomingPvP(store *engine.Store, reqID, attackerID, victimID string, payload json.RawMessage) error {
	var atk attackerPayload
	if err := json.Unmarshal(payload, &atk); err != nil {
		return fmt.Errorf("IncomingPvP: unmarshal: %w", err)
	}
	db := store.DB()
	vc, err := loadChar(db, victimID)
	if err != nil {
		resolvedAt := time.Now().Unix()
		return store.Emit("pvp.resolved", map[string]any{
			"request_id":         reqID,
			"attacker_global_id": attackerID,
			"victim_global_id":   victimID,
			"winner_global_id":   victimID,
			"result_text":        "loss",
			"resolved_at":        resolvedAt,
			"outcome":            "loss",
			"loot":               []pvpLootItem{},
			"victim_died":        false,
			"day_index":          serverDay(),
		})
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	aAtk := atk.Strength + itemBonus(atk.Weapon)
	aDef := atk.Defense + itemBonus(atk.Armor)
	vAtk := vc.Strength + itemBonus(vc.Weapon)
	vDef := vc.Defense + itemBonus(vc.Armor)
	aHP, vHP := atk.HP, vc.HP
	won := false
	for round := 0; round < 30; round++ {
		d, _ := Damage(aAtk, vDef, atk.Luck, rng)
		if vHP -= d; vHP <= 0 {
			won = true
			break
		}
		d2, _ := Damage(vAtk, aDef, vc.Luck, rng)
		if aHP -= d2; aHP <= 0 {
			break
		}
	}
	if aHP > 0 && vHP > 0 {
		won = float64(aHP)/float64(atk.MaxHP) > float64(vHP)/float64(vc.MaxHP)
	}
	if vHP < 1 {
		vHP = 1
	}
	vc.HP = vHP
	_ = saveChar(db, vc)
	lootDesc := "nothing"
	var lootItems []pvpLootItem
	if won {
		lootDesc, lootItems = stealLoot(db, victimID, rng)
	}
	if lootItems == nil {
		lootItems = []pvpLootItem{}
	}
	outcome := "loss"
	if won {
		outcome = "win"
	}
	winner := victimID
	if won {
		winner = attackerID
	}
	resolvedAt := time.Now().Unix()
	_, _ = db.Exec(
		`INSERT INTO pvp_log(attacker_id,attacker_name,victim_id,day_index,attacker_won,loot_desc,seen,created_at)
		 VALUES(?,?,?,?,?,?,0,?)`,
		attackerID, atk.Name, victimID, serverDay(), b2i(won), lootDesc, time.Now().Unix())
	return store.Emit("pvp.resolved", map[string]any{
		"request_id":         reqID,
		"attacker_global_id": attackerID,
		"victim_global_id":   victimID,
		"winner_global_id":   winner,
		"result_text":        outcome,
		"resolved_at":        resolvedAt,
		"outcome":            outcome,
		"loot":               lootItems,
		"victim_died":        false,
		"day_index":          serverDay(),
	})
}

func (g *Ledger) RegisterGameHandlers(store *engine.Store) {
	store.OnEvent("pvp.resolved", func(e engine.Event) error {
		var p struct {
			RequestID        string        `json:"request_id"`
			AttackerGlobalID string        `json:"attacker_global_id"`
			VictimGlobalID   string        `json:"victim_global_id"`
			Outcome          string        `json:"outcome"`
			Loot             []pvpLootItem `json:"loot"`
		}
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return nil
		}
		if p.RequestID == "" || strings.HasPrefix(p.RequestID, "local:") || !strings.HasPrefix(p.AttackerGlobalID, g.nodeID+":") {
			return nil
		}
		db := store.DB()
		won := p.Outcome == "win"
		for _, item := range p.Loot {
			if err := addGoods(db, p.AttackerGlobalID, item.ItemID, item.Qty); err != nil {
				log.Printf("pvp.resolved: addGoods %s: %v", p.AttackerGlobalID, err)
			}
		}
		lootDesc := "nothing"
		if len(p.Loot) > 0 {
			parts := make([]string, len(p.Loot))
			for i, item := range p.Loot {
				parts[i] = fmt.Sprintf("%dx %s", item.Qty, goodName(item.ItemID))
			}
			lootDesc = strings.Join(parts, ", ")
		}
		_, _ = db.Exec(
			`UPDATE pvp_log SET attacker_won=?, loot_desc=? WHERE attacker_id=? AND victim_id=? AND loot_desc='pending'`,
			b2i(won), lootDesc, p.AttackerGlobalID, p.VictimGlobalID)
		return nil
	})
}
