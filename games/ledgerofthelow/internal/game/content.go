package game

import "math/rand"

// Item is a gear entry. Bonus applies to attack (weapons) or defense (armor).
type Item struct {
	ID    string
	Name  string
	Bonus int
}

var items = map[string]Item{
	"wpn_rusted_shiv":     {"wpn_rusted_shiv", "Rusted Shiv", 3},
	"wpn_bone_mallet":     {"wpn_bone_mallet", "Bone Mallet", 6},
	"wpn_dock_hook":       {"wpn_dock_hook", "Dock Hook", 9},
	"wpn_warden_blade":    {"wpn_warden_blade", "Warden's Blade", 15},
	"arm_scavengers_wrap": {"arm_scavengers_wrap", "Scavenger's Wrap", 1},
	"arm_layered_rags":    {"arm_layered_rags", "Layered Rags", 5},
	"arm_vault_plate":     {"arm_vault_plate", "Vault Plate", 14},
	"arm_deep_harness":    {"arm_deep_harness", "Deep Harness", 8},
}

// TradeGood is a scavenged item with barter value but no combat use.
type TradeGood struct {
	ID     string
	Name   string
	Weight int
}

var tradeGoods = map[string]TradeGood{
	"trd_brass_fittings": {"trd_brass_fittings", "brass fittings", 8},
	"trd_lantern_glass":  {"trd_lantern_glass", "lantern glass", 5},
	"trd_bone_buttons":   {"trd_bone_buttons", "bone buttons", 3},
	"trd_copper_wire":    {"trd_copper_wire", "copper wire", 6},
	"trd_chalk_dust":     {"trd_chalk_dust", "chalk dust", 2},
	"trd_wound_cloth":    {"trd_wound_cloth", "wound cloth", 4},
	"trd_black_salt":     {"trd_black_salt", "black salt", 7},
	"trd_old_ink":        {"trd_old_ink", "old ink", 5},
	"trd_vault_brass":    {"trd_vault_brass", "vault brass", 9},
}

// lootDrops is the pool a combat win rolls against (one good per win).
var lootDrops = []string{
	"trd_brass_fittings",
	"trd_lantern_glass",
	"trd_bone_buttons",
	"trd_copper_wire",
	"trd_chalk_dust",
	"trd_wound_cloth",
	"trd_black_salt",
	"trd_old_ink",
}

func goodName(id string) string {
	if g, ok := tradeGoods[id]; ok {
		return g.Name
	}
	return id
}

func goodWeight(id string) int {
	if g, ok := tradeGoods[id]; ok {
		return g.Weight
	}
	return 0
}

// Ware is something Maren sells. Price is in barter units; Slot is where it equips.
type Ware struct {
	ItemID string
	Price  int
	Slot   string // weapon | armor
}

var marenWares = []Ware{
	{"wpn_bone_mallet", 10, "weapon"},
	{"wpn_dock_hook", 16, "weapon"},
	{"arm_layered_rags", 18, "armor"},
	{"arm_vault_plate", 40, "armor"},
}

func itemBonus(id string) int {
	if it, ok := items[id]; ok {
		return it.Bonus
	}
	return 0
}

func itemName(id string) string {
	if it, ok := items[id]; ok {
		return it.Name
	}
	return "nothing"
}

// Creature is a Warrens combatant.
type Creature struct {
	ID       string
	Name     string
	Band     int
	HP       int
	Strength int
	Defense  int
	Intro    string
}

var creaturesByBand = map[int][]Creature{
	1: {
		{"crt_gutter_eel", "gutter eel", 1, 12, 7, 3,
			"Something long unknots itself from the black water, decides you are food, and is only partly wrong."},
		{"crt_brick_rat", "brick rat", 1, 10, 6, 2,
			"A rat the size of a regret picks its way out of the masonry and squares up."},
		{"crt_drain_spider", "drain spider", 1, 14, 8, 4,
			"Eight eyes, no hurry. It has been waiting in the overflow drain since before you arrived. Possibly before you were born."},
		{"crt_mold_crawler", "mold crawler", 1, 16, 5, 6,
			"Low and slow. It doesn't look like a threat until it is one. That's the whole strategy."},
	},
	2: {
		{"crt_brick_haunt", "brick haunt", 2, 22, 11, 7,
			"A draught-shape in the bricked-over doorway. It wants you to leave. It will settle for the fight."},
		{"crt_vault_guardian", "vault guardian", 2, 20, 10, 10,
			"Whatever kept the vaults locked once, it's still at it. The lock is gone. The guardian isn't."},
		{"crt_debt_collector", "debt collector", 2, 18, 14, 5,
			"He has a clipboard and a broken nose and the look of a man who has collected from worse than you. He may be right."},
	},
	3: {
		{"crt_tally_shade", "tally shade", 3, 34, 16, 11,
			"It wears the outline of someone balanced out mid-sentence. The numbers on its skin are still trying to finish the equation."},
		{"crt_deep_mason", "deep mason", 3, 42, 13, 16,
			"Whatever built the lower tunnels is still here, still building. It objects to you watching."},
		{"crt_hunger_blind", "hunger blind", 3, 28, 19, 7,
			"No eyes. Feeds on lantern light and anything warm enough to cast a shadow. You qualify on both counts."},
	},
	4: {
		{"crt_obligation_wraith", "obligation wraith", 4, 46, 22, 12,
			"You owe something you have forgotten. It has not. It found you down here, where all debts eventually surface."},
		{"crt_ward_echo", "ward echo", 4, 36, 18, 17,
			"It moves like you. It fights like you would, if you were better at it. The Low made this from watching you."},
		{"crt_the_archivist", "Archivist", 4, 52, 20, 15,
			"Old. Patient. Cataloguing. You are in its records now. It prefers its subjects still."},
	},
}

func creatureForBand(band int, pick func(n int) int) Creature {
	for b := band; b >= 1; b-- {
		if cs, ok := creaturesByBand[b]; ok && len(cs) > 0 {
			return cs[pick(len(cs))]
		}
	}
	return creaturesByBand[1][0]
}

// ---- text pools ----

var attackHits = []string{
	"You find the gap and push through it.",
	"Your blow connects. Something gives.",
	"You drive in hard. The thing reconsiders.",
	"A solid strike. It will remember that.",
	"You don't miss. They'll wish you had.",
	"You press the advantage. It lands.",
	"Clean hit. Not clean enough, but it counts.",
	"You read it right and it costs them.",
	"You make something hurt that has not hurt in a while.",
	"A good angle. Everything lands where you meant it to.",
	"The moment opens. You fill it.",
	"Hard and direct. The way things get settled down here.",
	"They move. You move better.",
	"It connects with something like authority.",
}

var attackCrits = []string{
	"You find the seam. Whatever holds it together stops.",
	"Perfect timing. Ugly result. Yours.",
	"The kind of hit you can't plan. You take it anyway.",
	"It won't recover from that. You make sure of it.",
	"The dark itself seems to flinch.",
	"Everything goes right, all at once. You won't question it.",
	"That is the kind of hit the Low tells stories about. Briefly.",
	"Whatever it was built for, it wasn't built for that.",
	"You don't know where that came from. It doesn't matter.",
	"Something fundamental gives way.",
}

var creatureHits = []string{
	"It answers. You feel it.",
	"You're not the only one who can hit.",
	"Pain. Brief and educational.",
	"It finds you before you can stop it.",
	"The gap you left was smaller than you thought.",
	"Something gets through your guard.",
	"It doesn't fight fair. It just fights.",
	"You are reminded you are not alone down here.",
	"The thing knows its business.",
	"That's going to cost you later. It's costing you now.",
	"Whatever it is, it is not slow.",
	"You absorb it. There is no other option.",
	"The Low watches. The Low does not intervene.",
	"A fair return on the hit you landed.",
}

var fleeHits = []string{
	"You turn to go and the dark takes its toll.",
	"The gap closes as you reach for it.",
	"It catches you on the way out.",
	"Leaving costs you something. It always does.",
	"You make it most of the way before it disagrees.",
	"Retreat has a price. You're paying it.",
	"The thing objects to your exit.",
	"Running costs you something. File that under lessons learned.",
}

var fleeSuccessLines = []string{
	"You slip away from the %s into the dark.",
	"You leave the %s with your life and not much else.",
	"The %s lets you go, or stops caring which. Same result.",
	"The %s has other business. You take the opening.",
	"Dignity is for the surface. You keep what you carry.",
	"You and the %s reach an understanding: you leave, it allows it.",
	"The %s doesn't follow. You don't test that assumption.",
	"Running from a %s. Not your finest day. Still breathing, though.",
}

var fightWinLines = []string{
	"The %s is dead.",
	"The %s stops. You don't wait to see if it starts again.",
	"The %s folds and doesn't get up.",
	"The %s has had enough. Permanently.",
	"Whatever the %s was trying, it didn't work.",
	"The %s goes quiet. The Low goes quiet. Then just the Low.",
	"The %s is settled. You add it to the accounts.",
	"What the %s had, it has no longer. You saw to that.",
	"The %s ends. The dark notes it and moves on.",
	"You and the %s are done. The %s is more done.",
}

var fightDeathLines = []string{
	"The %s is the last thing you see. The dark does the rest.",
	"The %s finishes what it started. The Low collects.",
	"The %s wins. You were its day's work.",
	"The %s makes its point. You are the point.",
	"You do not dodge it this time. There is no this time.",
	"The %s had better numbers. The Ledger agrees.",
	"The %s was not what you expected. The Low knew.",
	"Something goes wrong. In the Warrens, that means one thing.",
}

// fightWinFlavors are appended to fight result pages to vary the aftermath.
var fightWinFlavors = []string{
	"The Low notes the transaction.",
	"Something in the deep shifts, briefly.",
	"The numbers, for once, are in your favor.",
	"The Ledger adds a mark that leans in your direction.",
	"The Warrens give back what they give back.",
	"Quiet, for a moment. Then just the Low again.",
}

// wakeAtThresholdLines are shown when the player dies and wakes at the Threshold.
var wakeAtThresholdLines = []string{
	"You wake in the Threshold, lighter than you were and owing the same.",
	"You open your eyes at the Threshold. The goods are gone. The debt stands.",
	"The Low delivers you to the Threshold. It doesn't send what you were carrying.",
	"You surface at the Threshold. The Warrens have balanced the books. Against you.",
}

// thresholdWakeFlavors are the second line after the wake message.
var thresholdWakeFlavors = []string{
	"The Ledger did not wait for you to agree.",
	"The Low takes what the Low takes.",
	"The tally was short. It isn't anymore.",
	"Whatever you were carrying is gone.",
}

// theLowAmbient are the default notice shown when entering the Lanternmarket.
var theLowAmbient = []string{
	" The lanterns sway, though there is no wind to sway them.",
	" Someone has chalked OWED above the archway. It has been there a long time.",
	" The Ledger is current. Somewhere, it is always current.",
	" You can hear the Warrens from here, if you listen the wrong way.",
	" The accounting runs like water down here. It finds every low place.",
	" A porter passes with a cart of things that don't bear examining. He nods.",
	" The hum of the market is the sound of debts compounding.",
	" The lamplighters change the wicks. The dark doesn't notice.",
	" Things are traded here that have no name on the surface. You don't ask.",
	" The market breathes. Not like something that needs to, but close enough.",
}

// thresholdDescs are [2]string pairs: first-line flavor + second-line transition.
var thresholdDescs = [][2]string{
	{
		" The quiet edge of the Low, where the newly-subtracted gather their wits",
		" before going down.  From here you can:",
	},
	{
		" You were balanced out. The Low found its equilibrium. Here you are.",
		" From here you can:",
	},
	{
		" Between the surface and the Warrens, there is this place. You are in it.",
		" From here you can:",
	},
	{
		" Every wanderer of the Low has stood here. Most of them went further.",
		" From here you can:",
	},
	{
		" The Low keeps your place. It is thoughtful that way, about what it keeps.",
		" From here you can:",
	},
}

var marenGreetings = []string{
	`"Take what you need. The Warrens will collect either way."`,
	`"You look like the Low's been at you. Good. Means it knows you're here."`,
	`"Business is bad when the living stop wanting. Lucky for me, they never do."`,
	`"Everything's got a price. The trick's knowing whether you're the buyer."`,
	`"You want it, take it. The tab covers what the scavenge can't. It always does."`,
	`"Don't give me that look. Everything I sell is priced fair. Fair for me."`,
	`"The Bonesetter's had a busy week. Good for her. Steady trade."`,
	`"I've been here since before the last reckoning. I'll be here after the next."`,
	`"Nothing I've got will keep you safe. Safer, maybe. That's all anyone offers."`,
	`"Pay the tab when you can. I'm patient. The Ledger is. The Warrens are not."`,
}

var bonesetterGreetings = []string{
	`"Come to be put back together, have you?"`,
	`"Another one. They always come eventually."`,
	`"Sit down. I've seen worse. Not much worse, but some."`,
	`"The Warrens do good work for me. Steady clientele."`,
	`"Don't thank me. Thank whatever stopped you going deeper."`,
}

var bonesetterHealthy = []string{
	`"Nothing broken on you I can charge for," the Bonesetter says, almost sorry.`,
	`"You're whole. Come back when you're not," says the Bonesetter, looking away.`,
	`"Nothing for me to do here. That'll change."`,
	`"Intact. Surprising. Come back when you're not."`,
}

// pickRandom returns a random element from pool using rng.
func pickRandom(rng *rand.Rand, pool []string) string {
	if len(pool) == 0 {
		return ""
	}
	return pool[rng.Intn(len(pool))]
}

// appendCombatLog appends entry to log, keeping at most max entries.
func appendCombatLog(log []string, entry string, max int) []string {
	log = append(log, entry)
	if len(log) > max {
		log = log[len(log)-max:]
	}
	return log
}
