package dominion

import (
	"database/sql"
	"fmt"
	"math/rand"
)

// groundAssault resolves a ground attack. Mutates unit counts and money in place;
// caller is responsible for persisting both sides.
// Returns a message for the attacker.
func groundAssault(attacker, defender *empireState, atm, dfm *militaryRow) string {
	ap := attackPower(atm)
	if ap == 0 {
		return noticeBad("You have no combat units to attack with.")
	}
	dp := defensePower(dfm)

	// Apply defender's home-field advantage: effective defense = dp * rand(0.8, 1.2).
	mult := 0.8 + rand.Float64()*0.4
	effDp := int(float64(dp) * mult)

	if ap > effDp {
		loot := defender.Money / 4
		defender.Money -= loot
		attacker.Money += loot
		applyCasualties(atm, 0.15)
		applyCasualties(dfm, 0.30)
		return noticeWin(fmt.Sprintf(
			"VICTORY! Seized %s. Your forces took 15%% casualties.", fmtCredits(loot)))
	}

	wipeForces(atm)
	applyCasualties(dfm, 0.05)
	return noticeBad("DEFEAT. Your attacking force was repelled and destroyed.")
}

func applyCasualties(m *militaryRow, rate float64) {
	shrink := func(n int) int {
		lost := int(float64(n) * rate)
		if n > 0 && lost == 0 {
			lost = 1
		}
		if lost > n {
			return 0
		}
		return n - lost
	}
	m.SoldiersNormal = shrink(m.SoldiersNormal)
	m.SoldiersSuper = shrink(m.SoldiersSuper)
	m.SoldiersMega = shrink(m.SoldiersMega)
	m.Tanks = shrink(m.Tanks)
	m.Hovercrafts = shrink(m.Hovercrafts)
}

func wipeForces(m *militaryRow) {
	m.SoldiersNormal = 0
	m.SoldiersSuper = 0
	m.SoldiersMega = 0
	m.Tanks = 0
	m.Hovercrafts = 0
}

// ballisticStrike resolves a missile attack. Mutates defender's empire and military.
// missileType: "nuclear" or "antimatter".
// Returns (message for attacker, was intercepted).
func ballisticStrike(db *sql.DB, defender *empireState, dfm *militaryRow, missileType string) (string, bool) {
	// Global Shield blocks all ballistic unconditionally.
	if dfm.GlobalShieldActive > 0 {
		return noticeBad("INTERCEPTED — Global Shield blocked the missile."), true
	}

	// Satellites intercept at 20% each, capped at 95%.
	interceptChance := float64(dfm.Satellites) * 0.20
	if interceptChance > 0.95 {
		interceptChance = 0.95
	}
	if interceptChance > 0 && rand.Float64() < interceptChance {
		return noticeBad(fmt.Sprintf(
			"INTERCEPTED by orbital satellites (%.0f%% chance).", interceptChance*100)), true
	}

	switch missileType {
	case "nuclear":
		loss := defender.Population / 10
		if loss < 1 {
			loss = 1
		}
		defender.Population -= loss
		if defender.Population < 1 {
			defender.Population = 1
		}
		regionDamage(db, defender.GlobalID)
		return noticeWin(fmt.Sprintf(
			"Nuclear strike hit. Killed %d population; one region damaged.", loss)), false
	case "antimatter":
		applyCasualties(dfm, 0.20)
		return noticeWin("Antimatter strike hit. 20%% of enemy military destroyed."), false
	}
	return noticeWin("Missile detonated."), false
}

// regionDamage deactivates one randomly chosen activated region.
func regionDamage(db *sql.DB, globalID string) {
	_, _ = db.Exec(`
UPDATE empire_regions SET activated = MAX(0, activated - 1)
WHERE empire_id = ? AND activated > 0
  AND region_type = (
      SELECT region_type FROM empire_regions
      WHERE empire_id = ? AND activated > 0
      ORDER BY RANDOM() LIMIT 1
  )`, globalID, globalID)
}

// spyScout reveals the target's military and treasury. Always succeeds; spy returns.
func spyScout(target *empireState, tfm *militaryRow) string {
	shieldStr := "offline"
	if tfm.GlobalShieldActive > 0 {
		shieldStr = "ACTIVE"
	}
	return fmt.Sprintf(
		"SCOUT REPORT — %s of %s\n"+
			"  Credits: %s on hand, %s banked\n"+
			"  Population: %d\n"+
			"  Soldiers: %d normal / %d super / %d mega\n"+
			"  Vehicles: %d tanks / %d hovercrafts\n"+
			"  Missiles: %d nuclear / %d antimatter\n"+
			"  Defense: %d turrets / %d satellites  Shield: %s\n"+
			"  Attack Power: %d   Defense Power: %d",
		target.EmpireName, target.WorldName,
		fmtCredits(target.Money), fmtCredits(target.MoneyBank),
		target.Population,
		tfm.SoldiersNormal, tfm.SoldiersSuper, tfm.SoldiersMega,
		tfm.Tanks, tfm.Hovercrafts,
		tfm.MissilesNuclear, tfm.MissilesAntimatter,
		tfm.Turrets, tfm.Satellites, shieldStr,
		attackPower(tfm), defensePower(tfm),
	)
}

// spySabotage attempts sabotage. Returns (message, spyConsumed).
// 50% success; spy consumed on failure.
func spySabotage(db *sql.DB, target *empireState, tfm *militaryRow) (string, bool) {
	if rand.Float64() >= 0.5 {
		return noticeBad("SABOTAGE FAILED: spy was captured."), true
	}

	// Success: pick a random sabotage target.
	switch rand.Intn(3) {
	case 0:
		if tfm.Turrets > 0 {
			tfm.Turrets--
			return noticeWin("SABOTAGE SUCCESS: spy destroyed a Ground Turret and returned."), false
		}
		// Fallthrough to region damage if no turrets.
		fallthrough
	case 1:
		regionDamage(db, target.GlobalID)
		return noticeWin("SABOTAGE SUCCESS: spy sabotaged a production region and returned."), false
	default:
		if atk := attackPower(tfm); atk > 0 {
			applyCasualties(tfm, 0.10)
			return noticeWin("SABOTAGE SUCCESS: spy eliminated 10% of enemy forces and returned."), false
		}
		return noticeWin("SABOTAGE SUCCESS: spy infiltrated but found no useful targets."), false
	}
}
