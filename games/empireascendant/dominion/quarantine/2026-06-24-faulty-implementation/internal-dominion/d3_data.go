package dominion

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ---- military ----

type militaryRow struct {
	SoldiersNormal     int
	SoldiersSuper      int
	SoldiersMega       int
	Tanks              int
	Hovercrafts        int
	MissilesNuclear    int
	MissilesAntimatter int
	ReconDrones        int
	Spies              int
	Turrets            int
	Satellites         int
	GlobalShieldActive int // 1 = on
}

func loadMilitary(db *sql.DB, globalID string) (*militaryRow, error) {
	m := &militaryRow{}
	err := db.QueryRow(`
SELECT soldiers_normal, soldiers_super, soldiers_mega, tanks, hovercrafts,
       missiles_nuclear, missiles_antimatter, recon_drones, spies,
       turrets, satellites, global_shield
FROM empire_military WHERE empire_id=?`, globalID).
		Scan(&m.SoldiersNormal, &m.SoldiersSuper, &m.SoldiersMega,
			&m.Tanks, &m.Hovercrafts, &m.MissilesNuclear, &m.MissilesAntimatter,
			&m.ReconDrones, &m.Spies, &m.Turrets, &m.Satellites, &m.GlobalShieldActive)
	if err == sql.ErrNoRows {
		return &militaryRow{}, nil
	}
	return m, err
}

func saveMilitary(db *sql.DB, globalID string, m *militaryRow) error {
	_, err := db.Exec(`
INSERT INTO empire_military(empire_id, soldiers_normal, soldiers_super, soldiers_mega,
    tanks, hovercrafts, missiles_nuclear, missiles_antimatter, recon_drones, spies,
    turrets, satellites, global_shield)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(empire_id) DO UPDATE SET
    soldiers_normal=excluded.soldiers_normal,
    soldiers_super=excluded.soldiers_super,
    soldiers_mega=excluded.soldiers_mega,
    tanks=excluded.tanks,
    hovercrafts=excluded.hovercrafts,
    missiles_nuclear=excluded.missiles_nuclear,
    missiles_antimatter=excluded.missiles_antimatter,
    recon_drones=excluded.recon_drones,
    spies=excluded.spies,
    turrets=excluded.turrets,
    satellites=excluded.satellites,
    global_shield=excluded.global_shield`,
		globalID,
		m.SoldiersNormal, m.SoldiersSuper, m.SoldiersMega,
		m.Tanks, m.Hovercrafts,
		m.MissilesNuclear, m.MissilesAntimatter,
		m.ReconDrones, m.Spies,
		m.Turrets, m.Satellites, m.GlobalShieldActive)
	return err
}

// attackPower sums the offensive strength of all military units.
func attackPower(m *militaryRow) int {
	return m.SoldiersNormal*1 + m.SoldiersSuper*3 + m.SoldiersMega*8 +
		m.Tanks*15 + m.Hovercrafts*20
}

// defensePower sums military strength plus static defensive installations.
func defensePower(m *militaryRow) int {
	base := m.SoldiersNormal*1 + m.SoldiersSuper*3 + m.SoldiersMega*8 +
		m.Tanks*15 + m.Hovercrafts*20
	shield := 0
	if m.GlobalShieldActive > 0 {
		shield = 200
	}
	return base + m.Turrets*20 + m.Satellites*50 + shield
}

// ---- recruit unit definitions ----

type unitDef struct {
	key      string
	field    string // field name for display
	name     string
	costCr   int
	requires string // tech ID or "" for none
	reqBldg  string // building check: "" or "intel"
}

var unitDefs = []unitDef{
	{"1", "soldiers_normal", "Normal Soldiers", 50, "", ""},
	{"2", "soldiers_super", "SuperHuman Soldiers", 200, TechSuperhuman, ""},
	{"3", "soldiers_mega", "MegaHuman Soldiers", 500, TechMegahuman, ""},
	{"4", "tanks", "Tanks", 400, TechTank, ""},
	{"5", "hovercrafts", "Hovercrafts", 600, TechHovercraft, ""},
	{"6", "missiles_nuclear", "Nuclear Missiles", 800, TechNuclear, ""},
	{"7", "missiles_antimatter", "Antimatter Missiles", 1500, TechAntimatter, ""},
	{"8", "recon_drones", "Recon Drones", 150, "", ""},
	{"9", "spies", "Spies", 250, "", "intel"},
}

func unitCount(m *militaryRow, field string) int {
	switch field {
	case "soldiers_normal":
		return m.SoldiersNormal
	case "soldiers_super":
		return m.SoldiersSuper
	case "soldiers_mega":
		return m.SoldiersMega
	case "tanks":
		return m.Tanks
	case "hovercrafts":
		return m.Hovercrafts
	case "missiles_nuclear":
		return m.MissilesNuclear
	case "missiles_antimatter":
		return m.MissilesAntimatter
	case "recon_drones":
		return m.ReconDrones
	case "spies":
		return m.Spies
	}
	return 0
}

func addUnits(m *militaryRow, field string, n int) {
	switch field {
	case "soldiers_normal":
		m.SoldiersNormal += n
	case "soldiers_super":
		m.SoldiersSuper += n
	case "soldiers_mega":
		m.SoldiersMega += n
	case "tanks":
		m.Tanks += n
	case "hovercrafts":
		m.Hovercrafts += n
	case "missiles_nuclear":
		m.MissilesNuclear += n
	case "missiles_antimatter":
		m.MissilesAntimatter += n
	case "recon_drones":
		m.ReconDrones += n
	case "spies":
		m.Spies += n
	}
}

// ---- pvp log (defender's inbox) ----

type pvpLogEntry struct {
	AttackerName  string `json:"attacker_name"`
	AttackerWorld string `json:"attacker_world"`
	EventType     string `json:"event_type"` // "ground", "missile", "spy"
	Outcome       string `json:"outcome"`    // "victory", "defeat"
	Detail        string `json:"detail"`
}

func writePvPLog(db *sql.DB, defenderID string, e pvpLogEntry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
INSERT INTO pvp_log(empire_id, event_type, outcome, detail, created_at)
VALUES(?,?,?,?,?)`,
		defenderID, e.EventType, e.Outcome, string(data), time.Now().Unix())
	return err
}

func loadPvPLog(db *sql.DB, globalID string) ([]pvpLogEntry, error) {
	rows, err := db.Query(`
SELECT detail FROM pvp_log WHERE empire_id=? ORDER BY created_at ASC`, globalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []pvpLogEntry
	for rows.Next() {
		var detail string
		if err := rows.Scan(&detail); err != nil {
			return nil, err
		}
		var e pvpLogEntry
		if json.Unmarshal([]byte(detail), &e) == nil {
			entries = append(entries, e)
		}
	}
	return entries, rows.Err()
}

func clearPvPLog(db *sql.DB, globalID string) error {
	_, err := db.Exec(`DELETE FROM pvp_log WHERE empire_id=?`, globalID)
	return err
}

func countPvPLog(db *sql.DB, globalID string) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM pvp_log WHERE empire_id=?`, globalID).Scan(&n)
	return n
}

// ---- per-target attack limits ----

func attackLimitOK(db *sql.DB, attackerID, targetID string, dayIndex int, col string, max int) bool {
	var count int
	_ = db.QueryRow(
		// col is a trusted constant, not user input
		fmt.Sprintf(`SELECT COALESCE(%s, 0) FROM attack_limits WHERE attacker_id=? AND target_id=? AND day_index=?`, col),
		attackerID, targetID, dayIndex).Scan(&count)
	return count < max
}

func incAttackLimit(db *sql.DB, attackerID, targetID string, dayIndex int, col string) error {
	_, err := db.Exec(
		fmt.Sprintf(`
INSERT INTO attack_limits(attacker_id, target_id, day_index, %s) VALUES(?,?,?,1)
ON CONFLICT(attacker_id, target_id, day_index) DO UPDATE SET %s=%s+1`, col, col, col),
		attackerID, targetID, dayIndex)
	return err
}

func attackCounts(db *sql.DB, attackerID, targetID string, dayIndex int) (ground, missile, spy int) {
	db.QueryRow(`SELECT COALESCE(ground_count,0), COALESCE(missile_count,0), COALESCE(spy_count,0)
FROM attack_limits WHERE attacker_id=? AND target_id=? AND day_index=?`,
		attackerID, targetID, dayIndex).Scan(&ground, &missile, &spy)
	return
}

// ---- target listing ----

type empireStub struct {
	GlobalID   string
	WorldName  string
	EmpireName string
}

func listTargets(db *sql.DB, selfID string) ([]empireStub, error) {
	// Hide empires whose player has not logged in for 14+ days.
	cutoff := time.Now().Unix() - 14*86400
	rows, err := db.Query(`
SELECT e.global_id, e.world_name, e.empire_name
FROM empires e
JOIN players p ON p.global_id = e.global_id
WHERE e.global_id != ? AND e.world_name != '' AND p.last_seen >= ?
ORDER BY e.empire_name ASC`, selfID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []empireStub
	for rows.Next() {
		var s empireStub
		if err := rows.Scan(&s.GlobalID, &s.WorldName, &s.EmpireName); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// purgeInactiveEmpires deletes all game data for local players absent 30+ days.
// Called from Run() on the first login of each new day.
func purgeInactiveEmpires(db *sql.DB, nodeID string) error {
	cutoff := time.Now().Unix() - 30*86400
	rows, err := db.Query(
		`SELECT p.global_id FROM players p
		 WHERE p.home_node=? AND p.last_seen < ?
		   AND EXISTS (SELECT 1 FROM empires WHERE global_id=p.global_id)`,
		nodeID, cutoff)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		for _, stmt := range []string{
			`DELETE FROM empires WHERE global_id=?`,
			`DELETE FROM empire_regions WHERE empire_id=?`,
			`DELETE FROM empire_buildings WHERE empire_id=?`,
			`DELETE FROM empire_mines WHERE empire_id=?`,
			`DELETE FROM empire_tech WHERE empire_id=?`,
			`DELETE FROM mineral_store WHERE empire_id=?`,
			`DELETE FROM empire_military WHERE empire_id=?`,
			`DELETE FROM pvp_log WHERE empire_id=?`,
		} {
			_, _ = db.Exec(stmt, id)
		}
	}
	return nil
}

// ---- scoring ----

type rankEntry struct {
	EmpireName string
	WorldName  string
	Score      int
}

func computeScore(e *empireState, m *militaryRow, techCount int) int {
	milScore := attackPower(m) + m.Turrets*5 + m.Satellites*10
	return e.Population +
		milScore*10 +
		int(float64(e.Money)*0.1) +
		techCount*500
}

func localRankings(db *sql.DB) ([]rankEntry, error) {
	rows, err := db.Query(`SELECT global_id, world_name, empire_name, money, population FROM empires WHERE world_name != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct {
		globalID, world, empire string
		money, pop              int
	}
	var empires []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.globalID, &r.world, &r.empire, &r.money, &r.pop); err != nil {
			return nil, err
		}
		empires = append(empires, r)
	}
	_ = rows.Close()

	var rankings []rankEntry
	for _, r := range empires {
		e := &empireState{GlobalID: r.globalID, WorldName: r.world, EmpireName: r.empire, Money: r.money, Population: r.pop}
		m, _ := loadMilitary(db, r.globalID)
		var tc int
		db.QueryRow(`SELECT COUNT(*) FROM empire_tech WHERE empire_id=? AND researched=1`, r.globalID).Scan(&tc)
		rankings = append(rankings, rankEntry{
			EmpireName: r.empire,
			WorldName:  r.world,
			Score:      computeScore(e, m, tc),
		})
	}

	// Sort descending by score (simple insertion sort — list is small).
	for i := 1; i < len(rankings); i++ {
		for j := i; j > 0 && rankings[j].Score > rankings[j-1].Score; j-- {
			rankings[j], rankings[j-1] = rankings[j-1], rankings[j]
		}
	}
	return rankings, nil
}
