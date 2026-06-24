package dominion

import (
	"database/sql"
)

// ---- region constants ----

const (
	RegionAgricultural = "agricultural"
	RegionIndustrial   = "industrial"
	RegionDesert       = "desert"
	RegionUrban        = "urban"
	RegionRiver        = "river"
	RegionOcean        = "ocean"
	RegionVolcanic     = "volcanic"
	RegionWasteland    = "wasteland"
)

var regionTypes = []string{
	RegionAgricultural, RegionIndustrial, RegionRiver, RegionUrban,
	RegionDesert, RegionOcean, RegionVolcanic, RegionWasteland,
}

// Base activation cost per region type.
var regionBaseCost = map[string]int{
	RegionAgricultural: 200,
	RegionIndustrial:   300,
	RegionRiver:        250,
	RegionUrban:        280,
	RegionDesert:       150,
	RegionOcean:        220,
	RegionVolcanic:     350,
	RegionWasteland:    120,
}

// nextRegionCost returns the cost to develop one more region of the given type.
// Cost increases by 50% of base for each region already owned.
func nextRegionCost(owned int, rtype string) int {
	base := regionBaseCost[rtype]
	return base + (base/2)*owned
}

// ---- mine constants ----

const (
	MineGold   = "gold"
	MineSilver = "silver"
	MineIron   = "iron"
	MineNickel = "nickel"
	MineCopper = "copper"
)

var mineTypes = []string{MineGold, MineSilver, MineIron, MineNickel, MineCopper}

// Units produced per miner per day.
var mineYield = map[string]int{
	MineGold: 1, MineSilver: 2, MineIron: 10, MineNickel: 5, MineCopper: 8,
}

// Credits per unit when selling.
var mineralPrice = map[string]int{
	MineGold: 50, MineSilver: 30, MineIron: 8, MineNickel: 12, MineCopper: 10,
}

// Starting deposit for a newly purchased mine.
var mineralDeposit = map[string]int{
	MineGold: 100, MineSilver: 200, MineIron: 1000, MineNickel: 500, MineCopper: 800,
}

// Cost to purchase a new mine of each type.
var mineBuyCost = map[string]int{
	MineGold: 2000, MineSilver: 1200, MineIron: 400, MineNickel: 600, MineCopper: 500,
}

// ---- tech constants ----

const (
	TechFission      = "energy_fission"
	TechFusion       = "energy_fusion"
	TechSuperhuman   = "soldiers_superhuman"
	TechMegahuman    = "soldiers_megahuman"
	TechTank         = "vehicle_tank"
	TechHovercraft   = "vehicle_hovercraft"
	TechNuclear      = "ballistic_nuclear"
	TechAntimatter   = "ballistic_antimatter"
	TechTurrets      = "defense_turrets"
	TechSatellites   = "defense_satellites"
	TechGlobalShield = "defense_global_shield"
	TechIntel        = "espionage_intel"
	TechHyperdrive   = "travel_hyperdrive"
)

// Research point costs per tech.
var techCost = map[string]int{
	TechFission:      200,
	TechFusion:       500,
	TechSuperhuman:   300,
	TechMegahuman:    600,
	TechTank:         250,
	TechHovercraft:   400,
	TechNuclear:      350,
	TechAntimatter:   700,
	TechTurrets:      150,
	TechSatellites:   400,
	TechGlobalShield: 800,
	TechIntel:        300,
	TechHyperdrive:   600,
}

// ---- structs ----

type regionRow struct {
	RegionType   string
	Quantity     int
	Activated    int
	ActivateCost int
}

type buildingsRow struct {
	MinersGuild         int
	FishingGuild        int
	FishersAssigned     int
	ConstructionFactory int
	ResearchLab         int
	IntelBuilding       int
	Lottery             int
}

type mineRow struct {
	MineType       string
	NumMines       int
	MinersAssigned int
	MineralLeft    int
}

// ---- load functions ----

func loadRegions(db *sql.DB, globalID string) (map[string]*regionRow, error) {
	rows, err := db.Query(
		`SELECT region_type, quantity, activated, activate_cost FROM empire_regions WHERE empire_id=?`,
		globalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]*regionRow)
	for rows.Next() {
		r := &regionRow{}
		if err := rows.Scan(&r.RegionType, &r.Quantity, &r.Activated, &r.ActivateCost); err != nil {
			return nil, err
		}
		m[r.RegionType] = r
	}
	return m, rows.Err()
}

func loadBuildings(db *sql.DB, globalID string) (*buildingsRow, error) {
	b := &buildingsRow{}
	err := db.QueryRow(
		`SELECT miners_guild, fishing_guild, fishers_assigned, construction_factory, research_lab, intel_building, lottery
		 FROM empire_buildings WHERE empire_id=?`, globalID).
		Scan(&b.MinersGuild, &b.FishingGuild, &b.FishersAssigned,
			&b.ConstructionFactory, &b.ResearchLab, &b.IntelBuilding, &b.Lottery)
	if err == sql.ErrNoRows {
		return &buildingsRow{}, nil
	}
	return b, err
}

func loadMines(db *sql.DB, globalID string) (map[string]*mineRow, error) {
	rows, err := db.Query(
		`SELECT mine_type, num_mines, miners_assigned, mineral_left FROM empire_mines WHERE empire_id=?`,
		globalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]*mineRow)
	for rows.Next() {
		r := &mineRow{}
		if err := rows.Scan(&r.MineType, &r.NumMines, &r.MinersAssigned, &r.MineralLeft); err != nil {
			return nil, err
		}
		m[r.MineType] = r
	}
	return m, rows.Err()
}

func loadMineralStore(db *sql.DB, globalID string) (map[string]int, error) {
	rows, err := db.Query(
		`SELECT mineral_type, quantity FROM mineral_store WHERE empire_id=?`, globalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int)
	for rows.Next() {
		var t string
		var q int
		if err := rows.Scan(&t, &q); err != nil {
			return nil, err
		}
		m[t] = q
	}
	return m, rows.Err()
}

// ---- tech helpers ----

func hasTech(db *sql.DB, globalID, techID string) bool {
	var v int
	_ = db.QueryRow(
		`SELECT researched FROM empire_tech WHERE empire_id=? AND tech_id=?`,
		globalID, techID).Scan(&v)
	return v == 1
}

func setTech(db *sql.DB, globalID, techID string) error {
	_, err := db.Exec(`
INSERT INTO empire_tech(empire_id, tech_id, researched) VALUES(?,?,1)
ON CONFLICT(empire_id, tech_id) DO UPDATE SET researched=1`, globalID, techID)
	return err
}

// ---- save helpers ----

func updateEmpire(db *sql.DB, e *empireState) error {
	_, err := db.Exec(
		`UPDATE empires SET money=?, money_bank=?, population=?, food_storage=?, energy=?, research_pts=?, building_pts=?
		 WHERE global_id=?`,
		e.Money, e.MoneyBank, e.Population, e.FoodStorage, e.Energy, e.ResearchPts, e.BuildingPts, e.GlobalID)
	return err
}

// addRegion inserts or increments an empire's region row.
func addRegion(db *sql.DB, globalID, regionType string, cost int) error {
	_, err := db.Exec(`
INSERT INTO empire_regions(empire_id, region_type, quantity, activated, activate_cost)
VALUES(?, ?, 1, 1, ?)
ON CONFLICT(empire_id, region_type) DO UPDATE SET
    quantity=quantity+1, activated=activated+1, activate_cost=?`,
		globalID, regionType, cost, cost)
	return err
}

func saveBuildings(db *sql.DB, globalID string, b *buildingsRow) error {
	_, err := db.Exec(`
INSERT INTO empire_buildings(empire_id, miners_guild, fishing_guild, fishers_assigned,
    construction_factory, research_lab, intel_building, lottery)
VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(empire_id) DO UPDATE SET
    miners_guild=excluded.miners_guild, fishing_guild=excluded.fishing_guild,
    fishers_assigned=excluded.fishers_assigned,
    construction_factory=excluded.construction_factory,
    research_lab=excluded.research_lab, intel_building=excluded.intel_building,
    lottery=excluded.lottery`,
		globalID, b.MinersGuild, b.FishingGuild, b.FishersAssigned,
		b.ConstructionFactory, b.ResearchLab, b.IntelBuilding, b.Lottery)
	return err
}

func addMineralStore(db *sql.DB, globalID, mineralType string, qty int) error {
	_, err := db.Exec(`
INSERT INTO mineral_store(empire_id, mineral_type, quantity) VALUES(?,?,?)
ON CONFLICT(empire_id, mineral_type) DO UPDATE SET quantity=quantity+?`,
		globalID, mineralType, qty, qty)
	return err
}

func clearMineralStore(db *sql.DB, globalID, mineralType string) error {
	_, err := db.Exec(
		`UPDATE mineral_store SET quantity=0 WHERE empire_id=? AND mineral_type=?`,
		globalID, mineralType)
	return err
}
