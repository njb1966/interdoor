package dominion

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"interdoor.net/interdoor/internal/engine"
)

// crossNodePayload is the JSON blob submitted to the hub when an attacker on
// this node strikes a player on another node. The defender's node receives it
// via IncomingPvP and resolves the combat locally.
type crossNodePayload struct {
	AttackType    string      `json:"attack_type"`            // "ground" or "ballistic"
	MissileType   string      `json:"missile_type,omitempty"` // "nuclear" or "antimatter"
	AttackerName  string      `json:"attacker_name"`
	AttackerWorld string      `json:"attacker_world"`
	AttackerMil   militaryRow `json:"attacker_mil"`
}

// pvpResolvedPayload is the event the DEFENDER's node emits after resolving an
// IncomingPvP request. The ATTACKER's node picks it up via RegisterGameHandlers.
type pvpResolvedPayload struct {
	RequestID        string `json:"request_id"`
	AttackerGlobalID string `json:"attacker_global_id"`
	VictimGlobalID   string `json:"victim_global_id"`
	Outcome          string `json:"outcome"` // "attacked", "repelled", "intercepted", "no_target"
	ResultText       string `json:"result_text"`
	VictimName       string `json:"victim_name"`
	VictimWorld      string `json:"victim_world"`
	LootAmount       int    `json:"loot_amount"`
}

// IncomingPvP resolves a cross-node attack on the DEFENDER's node.
// Called by the federation sync loop (Syncer.Tick) for each queued PvP request.
func (g *Dominion) IncomingPvP(store *engine.Store, reqID, attackerID, victimID string, payload json.RawMessage) error {
	var atk crossNodePayload
	if err := json.Unmarshal(payload, &atk); err != nil {
		return fmt.Errorf("IncomingPvP: %w", err)
	}
	db := store.DB()
	defender, err := loadEmpire(db, victimID)
	if err != nil {
		// No empire for this player on this node — emit to unblock the hub queue.
		return store.Emit("pvp.resolved", pvpResolvedPayload{
			RequestID:        reqID,
			AttackerGlobalID: attackerID,
			VictimGlobalID:   victimID,
			Outcome:          "no_target",
			ResultText:       "Target empire not found on this node.",
		})
	}
	dfm, _ := loadMilitary(db, victimID)
	atm := &atk.AttackerMil
	// fakeAttacker carries attacker identity for money bookkeeping in groundAssault.
	fakeAttacker := &empireState{
		GlobalID:   attackerID,
		EmpireName: atk.AttackerName,
		WorldName:  atk.AttackerWorld,
	}

	moneyBefore := defender.Money
	var result string
	switch atk.AttackType {
	case "ground":
		result = groundAssault(fakeAttacker, defender, atm, dfm)
		_ = updateEmpire(db, defender)
		_ = saveMilitary(db, victimID, dfm)
	case "ballistic":
		result, _ = ballisticStrike(db, defender, dfm, atk.MissileType)
		_ = updateEmpire(db, defender)
		_ = saveMilitary(db, victimID, dfm)
	default:
		return fmt.Errorf("IncomingPvP: unknown attack type %q", atk.AttackType)
	}

	lootAmount := moneyBefore - defender.Money
	if lootAmount < 0 {
		lootAmount = 0
	}

	outcome := "repelled"
	if strings.Contains(result, "VICTORY") || strings.Contains(result, "hit") {
		outcome = "attacked"
	} else if strings.Contains(result, "INTERCEPTED") {
		outcome = "intercepted"
	}

	_ = writePvPLog(db, victimID, pvpLogEntry{
		AttackerName:  atk.AttackerName,
		AttackerWorld: atk.AttackerWorld,
		EventType:     atk.AttackType,
		Outcome:       outcome,
		Detail:        fmt.Sprintf("[Cross-node] %s of %s: %s", atk.AttackerName, atk.AttackerWorld, result),
	})

	return store.Emit("pvp.resolved", pvpResolvedPayload{
		RequestID:        reqID,
		AttackerGlobalID: attackerID,
		VictimGlobalID:   victimID,
		Outcome:          outcome,
		ResultText:       result,
		VictimName:       defender.EmpireName,
		VictimWorld:      defender.WorldName,
		LootAmount:       lootAmount,
	})
}

// doRemoteGroundAssault submits a cross-node ground attack via the hub.
// Resources are spent locally; the result arrives via Galactic Dispatches.
func doRemoteGroundAssault(ctx *engine.Context, attacker *empireState, atm *militaryRow, victim engine.Player) string {
	if ctx.CrossNodeAttack == nil {
		return noticeBad("Not connected to the InterDOOR network.")
	}
	if attackPower(atm) == 0 {
		return noticeBad("You have no combat units to attack with.")
	}
	turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
	if turn.Actions < 1 {
		return noticeBad("No turns remaining today.")
	}
	if turn.Attacks < 1 {
		return noticeBad("No attacks remaining today.")
	}

	payload, _ := json.Marshal(crossNodePayload{
		AttackType:    "ground",
		AttackerName:  attacker.EmpireName,
		AttackerWorld: attacker.WorldName,
		AttackerMil:   *atm,
	})
	_, err := ctx.CrossNodeAttack(ctx.Player.GlobalID, victim.GlobalID, payload)
	if err != nil {
		log.Printf("cross-node ground %s->%s: %v", ctx.Player.GlobalID, victim.GlobalID, err)
		return noticeBad("Network error submitting attack.")
	}
	_ = ctx.Store.SpendActions(ctx.Player.GlobalID, 1)
	_ = ctx.Store.SpendAttack(ctx.Player.GlobalID)
	return noticeWin(fmt.Sprintf(
		"Ground forces dispatched toward %s's node. Check Galactic Dispatches for the result.", victim.Name))
}

// doRemoteMissileStrike submits a cross-node ballistic strike via the hub.
func doRemoteMissileStrike(ctx *engine.Context, attacker *empireState, atm *militaryRow, victim engine.Player, missileType string) string {
	if ctx.CrossNodeAttack == nil {
		return noticeBad("Not connected to the InterDOOR network.")
	}
	turn, _ := ctx.Store.LoadTurn(ctx.Player.GlobalID)
	if turn.Actions < 1 {
		return noticeBad("No turns remaining today.")
	}
	if turn.Attacks < 1 {
		return noticeBad("No attacks remaining today.")
	}
	switch missileType {
	case "nuclear":
		if atm.MissilesNuclear == 0 {
			return noticeBad("No nuclear missiles in arsenal.")
		}
	case "antimatter":
		if atm.MissilesAntimatter == 0 {
			return noticeBad("No antimatter missiles in arsenal.")
		}
	default:
		return noticeBad("Unknown missile type.")
	}

	payload, _ := json.Marshal(crossNodePayload{
		AttackType:    "ballistic",
		MissileType:   missileType,
		AttackerName:  attacker.EmpireName,
		AttackerWorld: attacker.WorldName,
		AttackerMil:   *atm,
	})
	_, err := ctx.CrossNodeAttack(ctx.Player.GlobalID, victim.GlobalID, payload)
	if err != nil {
		log.Printf("cross-node missile %s->%s: %v", ctx.Player.GlobalID, victim.GlobalID, err)
		return noticeBad("Network error submitting strike.")
	}
	// Consume the missile locally — the strike was launched regardless of outcome.
	switch missileType {
	case "nuclear":
		atm.MissilesNuclear--
	case "antimatter":
		atm.MissilesAntimatter--
	}
	_ = saveMilitary(ctx.Store.DB(), ctx.Player.GlobalID, atm)
	_ = ctx.Store.SpendActions(ctx.Player.GlobalID, 1)
	_ = ctx.Store.SpendAttack(ctx.Player.GlobalID)
	return noticeWin(fmt.Sprintf(
		"Missile launched toward %s's node. Check Galactic Dispatches for confirmation.", victim.Name))
}
