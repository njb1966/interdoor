package game

import (
	"math"
	"math/rand"
)

// Damage implements the CORE_LOOP §1.2 per-round formula:
//
//	raw      = max(1, attackPower - defensePower/2)
//	variance = random 0.8 .. 1.2
//	damage   = max(1, round(raw * variance))
//	crit     = random < luck%  -> damage *= 1.5
//
// It returns the damage dealt and whether the blow was a crit.
func Damage(attackPower, defensePower, luck int, r *rand.Rand) (int, bool) {
	raw := attackPower - defensePower/2
	if raw < 1 {
		raw = 1
	}
	variance := 0.8 + r.Float64()*0.4
	dmg := int(math.Round(float64(raw) * variance))
	if dmg < 1 {
		dmg = 1
	}
	crit := r.Intn(100) < luck
	if crit {
		dmg = dmg * 3 / 2
	}
	return dmg, crit
}

// fleeChance is 40% + luck*2% per CORE_LOOP §1.2 (depth penalty omitted in B1.1).
func fleeChance(luck int) int { return 40 + luck*2 }
