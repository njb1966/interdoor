package dominion

import (
	"database/sql"
	"math"
)

// energyRate returns energy produced per activated industrial region per day
// based on the empire's current energy tech level.
func energyRate(db *sql.DB, globalID string) int {
	if hasTech(db, globalID, TechFusion) {
		return 2000
	}
	if hasTech(db, globalID, TechFission) {
		return 500
	}
	return 100 // Fossil (default)
}

// runProductionTick applies one day's production to the empire.
// Called by Run() when ResetIfNewDay returns true.
func runProductionTick(db *sql.DB, globalID string) error {
	e, err := loadEmpire(db, globalID)
	if err != nil {
		return err
	}
	regions, err := loadRegions(db, globalID)
	if err != nil {
		return err
	}
	buildings, err := loadBuildings(db, globalID)
	if err != nil {
		return err
	}
	mines, err := loadMines(db, globalID)
	if err != nil {
		return err
	}

	agr := activated(regions, RegionAgricultural)
	riv := activated(regions, RegionRiver)
	ind := activated(regions, RegionIndustrial)

	// 1. Food production
	foodProd := agr*500 + riv*300 + buildings.FishersAssigned*50
	e.FoodStorage += foodProd

	// 2. Food consumption
	consumed := int(math.Round(float64(e.Population) * 0.1))
	e.FoodStorage -= consumed

	// 3. Population growth/shrink based on food balance
	if e.FoodStorage > 0 {
		growth := int(math.Round(float64(e.Population) * 0.01))
		if growth < 1 {
			growth = 1
		}
		e.Population += growth
	} else {
		loss := int(math.Round(float64(e.Population) * 0.005))
		if loss < 1 {
			loss = 1
		}
		e.Population -= loss
		if e.Population < 1 {
			e.Population = 1
		}
		e.FoodStorage = 0
	}

	// 4. Energy production
	e.Energy += ind * energyRate(db, globalID)

	// 4b. Global Shield energy drain (1000/day; auto-drops at 0).
	mil, _ := loadMilitary(db, globalID)
	if mil != nil && mil.GlobalShieldActive > 0 {
		e.Energy -= 1000
		if e.Energy < 0 {
			e.Energy = 0
			mil.GlobalShieldActive = 0
			_ = saveMilitary(db, globalID, mil)
		}
	}

	// 5. Mine output → mineral store
	for _, mine := range mines {
		if mine.NumMines == 0 || mine.MinersAssigned == 0 || mine.MineralLeft == 0 {
			continue
		}
		yield := mine.MinersAssigned * mineYield[mine.MineType]
		if yield > mine.MineralLeft {
			yield = mine.MineralLeft
		}
		mine.MineralLeft -= yield
		if _, err := db.Exec(
			`UPDATE empire_mines SET mineral_left=? WHERE empire_id=? AND mine_type=?`,
			mine.MineralLeft, globalID, mine.MineType); err != nil {
			return err
		}
		if err := addMineralStore(db, globalID, mine.MineType, yield); err != nil {
			return err
		}
	}

	// 6. Research points from labs
	e.ResearchPts += buildings.ResearchLab * 10

	// 7. Building points from factories
	e.BuildingPts += buildings.ConstructionFactory * 5

	return updateEmpire(db, e)
}

func activated(regions map[string]*regionRow, rtype string) int {
	if r, ok := regions[rtype]; ok {
		return r.Activated
	}
	return 0
}
