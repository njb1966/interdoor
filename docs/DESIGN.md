# DORNHAVEN: THE LOW

> Status: Historical game-specific source material. The root numbered InterDoor docs are canonical for Phase 1 framework, protocol, operator, and architecture decisions. Do not use this file as Phase 1 framework authority unless it has been explicitly reconciled.

## Game Design Document — v0.1

---

## PREMISE

Dornhaven is a fictional Hanseatic trading city on the Baltic coast. Founded centuries ago by a consortium of merchant families, it has prospered without interruption through wars, plagues, and economic collapses that leveled its neighbors. No one questions this anymore. Prosperity is simply what Dornhaven does.

Beneath the modern city lies the Low — a vast, layered underground built from centuries of forgotten infrastructure. Bricked-over cellars, flooded merchant vaults, smuggler passages, guild halls that predate the city's oldest records. The Low is populated by people who have been *balanced out* of the world Above.

They weren't banished. They weren't punished. They were *subtracted*. The Ledger — the self-sustaining system that enforces the terms of an ancient civic contract called the Charter — determined that the math required their removal. One day they existed Above. The next, no one remembered them. Their apartments were rented to strangers. Their names vanished from records. They woke up in the dark, in the Low, with nothing.

The player is one of these people. The game is about surviving the Low, building a life in it, uncovering the history of the Charter, understanding the Ledger, and eventually confronting the system itself — to destroy it, rewrite it, replace it, or simply escape it.

---

## TONE

Dark-whimsical. Gaiman's sense of hidden worlds and old bargains, filtered through Adams's deadpan absurdism. The game describes dangerous situations with dry, observational wit. NPCs are eccentric in ways that seem charming until you realize they're also terrifying. The humor comes from juxtaposition — the mundane and the impossible, presented with equal weight.

**Example NPC dialogue:**
> "You'll want to avoid the third passage on the left. Not because of anything dangerous, mind you — it's perfectly safe. It's just that it isn't there on Tuesdays, and you seem like the sort of person who'd be halfway through when Tuesday arrives."

**Example exploration text:**
> The tunnel opens into what was once a merchant's counting house. The abacus on the desk still works, which is more than can be said for the merchant, who has been dead for approximately three hundred years but whose skeleton maintains an expression of profound irritation at being interrupted.

The tone is never campy, never winking at the player. The world takes itself seriously. The humor emerges from the world being what it is.

---

## THE LOW — DISTRICTS

The Low is organized into distinct districts, each with its own character, dangers, and gameplay functions. Districts are layered vertically (closer to the surface = safer, deeper = more dangerous) and spread horizontally (different districts at the same depth, controlled by different factions).

### The Threshold

**Depth:** Surface-adjacent
**Danger:** Minimal
**Function:** Tutorial / arrival zone

The transitional space between Above and the Low. Not quite a district — more of a condition. New arrivals wake up here, disoriented, in whatever forgotten crawlspace or bricked-over basement the Ledger deposited them in. A few residents make it their business to find newcomers before something else does. Their motives vary.

The Threshold is where players learn the basic mechanics: movement, interaction, the favor economy, and the uncomfortable truth of their situation. It's brief — a day or two of play — before the player makes their way to the Lanternmarket and the game proper begins.

**Key NPCs:** The Lamplighter (a guide figure who orients newcomers — cheerful, helpful, and evasive about their own history), scavengers who offer "help" at unfavorable exchange rates.

---

### The Lanternmarket

**Depth:** Upper Low
**Danger:** Low (enforced)
**Function:** Central hub / economy / social space

A vast underground plaza where hundreds of lanterns hang from chains bolted into a ceiling so high it disappears into shadow. The commercial heart of the Low. Traders, merchants, contract brokers, information sellers, and favor dealers operate from stalls, alcoves, and repurposed merchant vaults.

The Market operates under the Market Truce — a rare point of agreement among all factions that violence within the Market is forbidden. Enforcement is communal and immediate. This makes the Lanternmarket the safest place in the Low, and therefore the place where everything important happens. Deals, alliances, betrayals (subtle ones), recruitment, and gossip.

**Gameplay:** This is the player's primary hub. Trade goods, post and accept contracts, negotiate favors, check the debt board (public record of outstanding obligations), interact with other players, and access faction representatives. The Market is where you spend resources; everything else is where you earn them.

**Key NPCs:** The Broker (manages contracts and mediates disputes — scrupulously fair, which in the Low is its own form of menace), various merchants with rotating inventories, faction recruiters.

---

### The Rafters

**Depth:** Upper Low
**Danger:** Low-moderate
**Function:** Residential / faction headquarters / social

The highest level of the Low, pressed up against the underside of the modern city. Old cellars, converted warehouse foundations, bricked-over basements. You can hear the city Above through the ceiling — footsteps, traffic, the muffled thump of music from a bar whose patrons have no idea what's beneath them.

The Rafters are where established Low folk live. People who've been here long enough to have proper quarters, furniture, even a semblance of normalcy. It's politically dense — faction headquarters are here, and territorial boundaries are negotiated constantly. The Rafters are safe if you know whose territory you're in and behave accordingly. Dangerous if you don't.

**Gameplay:** Faction quest hubs, player housing (earned through reputation), social interaction spaces. This is where the political game happens — faction reputation quests, territorial disputes, alliance building.

**Key locations:** Each faction maintains a headquarters in the Rafters (detailed in the Factions section).

---

### The Gutters

**Depth:** Mid-Low
**Danger:** Moderate-high
**Function:** Transit / exploration / maritime threats

The waterways. Flooded tunnels that connect to the old harbor infrastructure, smuggler passages that once ran to the sea, underground canals that predate the city's written records. Water flows through the Low constantly — some of it rainwater seeping from Above, some of it from sources deeper down that no one has identified.

The Gutters are the fastest way to travel between distant parts of the Low, but they're dangerous. Things live in the water. The passages flood unpredictably. Navigation requires knowledge or a guide, and guides charge accordingly.

**Gameplay:** Transit system between districts (faster than walking through the Warrens but riskier). Water-themed exploration encounters. Smuggling runs (high-value contract work). The Ferrymen faction controls most Gutter routes.

**Key NPCs:** Ferrymen guides (hired at varying rates depending on route danger), Gutter-dwellers (people too paranoid or antisocial for the Rafters who live on platforms above the waterline).

---

### The Warrens

**Depth:** Mid-Low to Deep
**Danger:** High
**Function:** Primary exploration / combat / scavenging

Twisting, unmapped passages that extend in every direction. The Warrens are the Low's wilderness — the spaces between settled districts where the infrastructure is old, unstable, and only partially understood. Corridors that don't appear on any map. Rooms sealed for centuries. Passages that seem to rearrange themselves between visits (whether this is the Ledger's doing or simple structural instability is a matter of debate).

This is where players go to scavenge, fight, and find things that were lost. The Warrens are dangerous but rewarding — the deeper you go, the older the infrastructure, the more valuable the salvage, and the worse the things that live there.

**Gameplay:** The core exploration/combat loop. Each expedition into the Warrens is procedurally structured — a series of encounters (combat, discovery, environmental hazard, NPC interaction) determined by depth, player capability, and randomization. This is the "forest fights" equivalent, but with more variety and escalating danger the deeper you push.

**Encounter types:**
- **Creatures** — Things that live in the dark. Not animals, not people, not easily categorized. Some are territorial, some are predatory, some are just strange.
- **Scavenge sites** — Abandoned rooms, collapsed vaults, forgotten caches. Risk/reward — some are trapped, some are claimed by something.
- **Environmental hazards** — Collapses, gas pockets, flooded sections, unstable floors. Navigable with the right gear or knowledge.
- **Other Walkers** — NPCs or other players exploring the same area. Encounters can be cooperative, competitive, or hostile.
- **Anomalies** — Places where the Ledger's influence is visible. Walls with numbers scratched into them. Doors that only open for people carrying specific debts. Passages that reroute based on your obligation profile. These are clues to how the system works.

---

### The Vaults

**Depth:** Mid-Low
**Danger:** Variable (faction-dependent)
**Function:** Faction strongholds / high-value resources

Old merchant vaults, treasury rooms, guild halls from Dornhaven's golden age. Massive, well-built spaces designed to store wealth — and now repurposed by factions as strongholds, armories, and treasuries. Each major faction controls at least one Vault.

Entering a Vault without invitation is a political act with consequences. Being invited into a Vault is a sign of trust and faction standing.

**Gameplay:** High-tier faction content. Access gated by reputation. Inside, players find faction-specific merchants, advanced quests, exclusive resources, and lore relevant to each faction's understanding of the Charter and the Ledger.

---

### The Archive

**Depth:** Deep Low
**Danger:** High (not from combat)
**Function:** Endgame research / lore / Charter investigation

Somewhere in the deep Low, there exists a vast repository of records. Not the Ledger itself — but the accumulated bureaucratic residue of centuries. Old contracts, forgotten debt instruments, maps of passages that no longer exist, municipal records from eras Above that have been completely forgotten. The Archive is where the Charter's history lives — in fragments, scattered across rooms that may not be in the same order they were yesterday.

The Archive is not guarded by people or creatures. It's guarded by the Ledger's attention. The deeper you research, the more the Ledger notices you. This manifests as increasing strangeness — your debts fluctuating without cause, passages rerouting to delay you, NPCs forgetting conversations you had with them. The Ledger doesn't attack. It *adjusts*.

**Gameplay:** The endgame progression system. Players spend actions on research sessions that yield Knowledge fragments — pieces of the Charter's history, the Ledger's mechanisms, the identities of the original signatories, and the nature of the bargain itself. Enough fragments unlock understanding of each aspect of the system. Full understanding unlocks the path to the Deep and the endgame confrontation.

Research is not combat. It's investigation — choosing which threads to follow, deciding which sources to trust (some records have been altered, by whom?), and managing the Ledger's increasing attention as you get closer to the truth.

---

### The Deep

**Depth:** Below the Low
**Danger:** Extreme
**Function:** Endgame zone

Below the Low. The oldest passages. Pre-city. Possibly pre-human. The architecture here doesn't match anything from Dornhaven's history — the stonework is older, the proportions are wrong, and the geometry doesn't always behave.

The Deep is where the Ledger's heart is. Not a room with a glowing book — something more distributed, more structural. The walls themselves are the Ledger here. Every surface is covered in faintly luminous marks — numbers, names, obligations, balances — shifting and updating in real time. The Deep is the Ledger made visible.

**Gameplay:** The final sequence. Reaching the Deep requires sufficient Knowledge, faction support, resources, and allies. The confrontation with the Ledger is not combat — it's a series of choices informed by everything the player has learned. The outcome depends on what the player knows, who they've allied with, and what they're willing to sacrifice.

---

## FACTIONS

Four factions control the Low's politics. Players can build reputation with multiple factions simultaneously but must ultimately align with one to access endgame content. Faction alignment determines the player's endgame options.

### The Ledgermen

**Philosophy:** The system works. Work within it.
**Territory:** The largest and most stable Vault in the Rafters.
**Tone:** Bureaucratic, polite, quietly terrifying.

The Ledgermen believe the Charter and the Ledger are just — or at least, necessary. They argue that the Low exists because the math requires it, and that fighting the math is like fighting gravity. Instead, they've built power within the system, accumulating favors and managing debts with surgical precision. They're the Low's establishment — the closest thing it has to a government.

They're not evil. They genuinely believe that stability within the system is the only alternative to chaos. They point out, correctly, that every attempt to challenge the Ledger has ended badly. They offer newcomers protection, structure, and a path to a comfortable life in the Low — in exchange for loyalty and, of course, favors.

**Gameplay benefits:** Best trade networks, most stable territory, access to Ledger-adjacent information (they understand the debt economy better than anyone). High-tier Ledgermen can manipulate debt instruments in ways other factions can't.

**Endgame path:** Reform. Rewrite the Charter's terms from within. The Ledger persists but operates differently.

---

### The Remnants

**Philosophy:** Understand before you act.
**Territory:** The Archive (or the parts they've been able to access and hold).
**Tone:** Scholarly, cautious, obsessive, occasionally insufferable.

The Remnants are historians, archivists, and scholars who believe that knowledge is the only real power in the Low. They've been studying the Charter and the Ledger for generations — accumulating fragments, building theories, arguing among themselves about interpretations. They control access to the Archive and guard it jealously.

They're paralyzed by their own thoroughness. They always need more data, more fragments, more certainty before they'll act. This makes them infuriating allies but invaluable sources of knowledge. They know more about the Charter than anyone — they just can't agree on what to do about it.

**Gameplay benefits:** Access to the Archive, Knowledge fragments, lore quests, research tools. The Remnants accelerate the investigative endgame significantly. They also identify and translate Anomalies in the Warrens that other factions overlook.

**Endgame path:** Understanding. Confront the Ledger with complete knowledge of the Charter's terms and exploit a loophole or flaw in the original agreement.

---

### The Tally

**Philosophy:** Burn it down.
**Territory:** A fortified section of the Warrens, deliberately deep and hard to find.
**Tone:** Angry, idealistic, reckless, alive.

The Tally wants the Charter voided and the Ledger destroyed, full stop. They see the system as fundamentally unjust — a machine that feeds on human lives to keep a city of oblivious people comfortable — and they believe that any action against it is morally justified. They don't care about the consequences for Above. They don't care about the theoretical risks. They care about the people who are trapped Below right now.

They're the most welcoming faction for newcomers, because fresh anger is useful and because they remember what it felt like. They're also the most likely to get you killed — their operations are aggressive, their targets are ambitious, and their casualty rate is high. But they're the only faction that treats the situation with the urgency it deserves.

**Gameplay benefits:** Best combat training and equipment, aggressive quests with high rewards, access to parts of the Warrens and the Deep that other factions avoid. The Tally doesn't wait for permission.

**Endgame path:** Destruction. Void the Charter by force. Destroy the Ledger's mechanism. Accept the consequences for Above and Below alike.

---

### The Ferrymen

**Philosophy:** Keep moving. Owe no one. Trust the current.
**Territory:** The Gutters — every navigable waterway in the Low.
**Tone:** Laconic, pragmatic, privately philosophical, unreliable narrators.

The Ferrymen control transit through the Low. They navigate the Gutters, maintain passage routes, and carry goods, people, and information between districts. Everyone needs them, so nobody can afford to antagonize them. They maintain strict neutrality in faction disputes — not out of principle, but because neutrality is their business model.

Beneath the pragmatism, the Ferrymen have their own theory about the Low: that the waterways are older than the Ledger, older than the Charter, older than Dornhaven. They believe the Low existed before the city, and that the founders didn't create the Charter — they found it, and foolishly activated something that was already here. The Ferrymen's endgame isn't about destroying or reforming the system. It's about navigating past it entirely.

**Gameplay benefits:** Mobility. Ferrymen-aligned players travel between districts faster and cheaper. They also get access to intelligence — the Ferrymen hear everything — and smuggling contracts that are unavailable to other factions. They can access parts of the Gutters that connect to... somewhere else.

**Endgame path:** Escape. Don't destroy the Ledger, don't reform it — find the passage that goes *around* it. The Ferrymen believe there's a way out that doesn't require confronting the system at all. Whether they're right is the gamble.

---

## THE ECONOMY

### Currency: Favors and Debts

The Low has no money. Coins from Above are worthless — who would you spend them with? Instead, the economy runs on **favors** (things people owe you) and **debts** (things you owe others).

A favor is a transferable, trackable obligation. If you help someone, they owe you a favor. You can call it in (demand a service), trade it (transfer the obligation to someone else), or hold it (accumulate power through uncalled debts). Favors have *weight* based on who owes them — a favor from a faction leader is worth more than a favor from a newcomer.

Debts are the inverse. When you receive help, goods, information, or safe passage, you incur a debt. Debts must eventually resolve — the Ledger tracks them, and unresolved debts have consequences. Your **Debt Load** is a core game metric: too much debt makes you vulnerable to Ledger attention and faction pressure; too little means you haven't been participating in the economy and lack the connections to progress.

### The Tension

To progress, you must trade. To trade, you must incur debts. The Ledger tracks all debts. Therefore, *progression itself feeds the system you're trying to understand and potentially destroy*. This is by design — mechanically and narratively. The player cannot avoid engaging with the Ledger's economy. The question is how they manage that engagement.

### Reputation

Separate from favors and debts, each player has a **Reputation** score with each faction and an overall **Standing** in the Low. Reputation determines access — what quests are available, which NPCs will talk to you, which territories you can enter. Standing determines visibility — how well-known you are, which affects both opportunities and danger.

High Standing means better opportunities but more Ledger attention. Low Standing means anonymity but limited access. Managing this balance is part of the strategy.

---

## DAILY ACTION LOOP

Each day, the player has a fixed number of **Actions** (default: 12, tunable for balance). Actions are spent on activities across the Low's districts. A day's play session should take 10-20 minutes.

### Action Costs

| Activity | Actions | Location |
|----------|---------|----------|
| Explore the Warrens (shallow) | 2 | Warrens |
| Explore the Warrens (deep) | 3 | Warrens |
| Navigate the Gutters | 2 | Gutters |
| Trade at the Lanternmarket | 1 | Lanternmarket |
| Faction quest (minor) | 2 | Varies |
| Faction quest (major) | 4 | Varies |
| Research at the Archive | 3 | Archive |
| Rest / recover from injury | 1 | Rafters |
| Accept or post a contract | 0 | Lanternmarket |
| Player-to-player interaction | 0 | Any |

### Daily Rhythm

The Ledger resets available actions at a fixed time (server midnight). Unspent actions do not roll over. This creates the daily habit loop: log in, decide how to spend your actions, execute, log out, anticipate tomorrow.

The world changes between sessions. Other players act. NPC storylines advance. The Warrens shift. Contracts expire. Debts accrue interest. The game is alive when you're not playing, which creates the anticipation that drives daily returns.

### Death and Consequence

If a player dies (health reaches zero from combat, environmental hazard, or faction violence):

- **Carried inventory is lost.** Whatever you had on you is gone — scavenged by whatever killed you or absorbed by the Warrens.
- **Banked resources persist.** Anything stored in your quarters (Rafters) or a faction Vault is safe.
- **Debts persist.** Death does not clear your ledger. The Ledger remembers.
- **You wake up in the Threshold the next day.** Not as a newcomer — you're still you, still known — but you have to make your way back to wherever you were, which costs actions.
- **Reputation takes a small hit.** Dying isn't shameful, but it isn't impressive either.

Death stings without being devastating. It costs you time and resources but not your character or your progress toward the endgame. The threat is real enough to make survival matter without punishing players so hard they quit.

---

## PROGRESSION

Progression is not level-based. There is no XP bar. Instead, players advance across four parallel tracks:

### 1. Reputation (per faction)

A numeric score with each of the four factions, earned through quests, favors, and factional service. Reputation gates access to content, NPCs, territories, and eventually endgame paths.

Reputation can be lost through opposing factional actions. Full reputation with one faction may reduce standing with its rivals.

### 2. Standing (global)

How well-known you are in the Low overall. Standing increases through notable actions — completing high-profile contracts, surviving deep Warren expeditions, influencing faction politics, accumulating significant favor networks.

Standing is a double-edged metric. High Standing opens doors but also attracts Ledger attention and factional scrutiny. A player can deliberately manage their Standing — sometimes anonymity is more valuable than fame.

### 3. Knowledge (endgame progression)

Fragments of information about the Charter, the Ledger, and the Deep. Earned through Archive research, Anomaly investigation in the Warrens, faction lore quests, and NPC conversations. Knowledge fragments assemble into understanding of specific aspects of the system:

- **The Charter's Terms** — What was actually agreed to, and by whom.
- **The Ledger's Mechanisms** — How it tracks debts, how it selects people for balancing, how it enforces.
- **The Signatories** — Who made the original bargain, and what they understood they were doing.
- **The Deep** — What's down there, how to reach it, what to expect.
- **The Flaw** — Whether the system has a vulnerability, and what it is.

Each knowledge area has multiple fragments. Assembling enough fragments in an area unlocks understanding. Full understanding across all areas unlocks the endgame sequence.

### 4. Resources

Gear, goods, materials, and consumables — the tangible assets that make survival easier. Better gear means deeper exploration, harder quests, and more options in dangerous situations. Resources are earned through scavenging, trading, and quest rewards.

Resources are deliberately finite and perishable. Gear degrades. Consumables are consumed. The economy keeps players engaged in the scavenging/trading loop even at high progression levels.

---

## ENDGAME

When a player has assembled sufficient Knowledge and faction reputation, they can attempt **the Descent** — the journey into the Deep to confront the Ledger.

The Descent is not a single action. It's a multi-day sequence requiring:

- Minimum Knowledge thresholds across all five areas
- Alignment with at least one faction at high reputation
- Sufficient resources and gear for an extended expedition
- (Optional) Allied players willing to make the journey together

### The Confrontation

At the heart of the Deep, the Ledger is not a book or an artifact. It's the architecture itself — walls covered in shifting marks, numbers, names, balances. The Deep is the Ledger made manifest.

The confrontation is not combat. It's a series of **choices** informed by what the player knows, who they've allied with, and what they're willing to sacrifice. The available endings depend on faction alignment:

**Void (The Tally path):** Destroy the Ledger's mechanism entirely. The Charter is nullified. Everyone in the Low is free — and the consequences for Dornhaven Above are... unknown. The player's character emerges Above, remembered again. The game resets for that character with a Void marker — subsequent playthroughs in a world where the Charter was destroyed (or wasn't, if other players chose differently on their nodes).

**Reform (The Ledgermen path):** Rewrite the Charter's terms. The Ledger persists but operates under new rules — rules the player defines, within constraints. A delicate negotiation with a system that doesn't want to change. The player's character becomes part of the new order. Subsequent playthroughs have access to the Reformed system's benefits and limitations.

**Exploit (The Remnants path):** Use complete knowledge of the Charter to find and leverage its flaw. Not destruction, not reform — subversion. The Ledger continues operating, but the player has found a way to exempt themselves and others from its calculations. Elegant, intellectual, and arguably selfish. The system persists, but the player is free of it.

**Passage (The Ferrymen path):** Don't confront the Ledger at all. Navigate past it, through the waterways that predate the Charter, to whatever lies on the other side. The player leaves the Low, leaves Dornhaven, leaves the game's geography entirely. The most mysterious ending — and the one that raises the most questions about what the Low actually is.

After any ending, the player may start a new character. The world state may be affected by the chosen ending (especially in federation mode, where node-wide consequences ripple).

---

## FEDERATION MODEL (Phase 2)

*This section describes the federated multiplayer system to be implemented after the single-node game is complete and proven.*

### Concept

Each federated node represents a **district** of a larger Low — or, more ambitiously, a different city's Low connected by deep passages. The fiction supports both interpretations.

### What Syncs

- **Player roster:** Who exists on which node. Players can see (but not interact with) players on other nodes until passage between nodes is established.
- **Debt network:** Debts can cross node boundaries. A favor owed to someone on another node is still tracked.
- **Event feed:** Major events on each node (faction shifts, endgame completions, notable achievements) are broadcast to all nodes.
- **Inter-node travel:** Players can spend significant actions to travel between nodes, enabling cross-node trade, PvP, and cooperation.

### Architecture

Hub-and-spoke. One node serves as the coordination hub. Other nodes register and sync state via REST API. The hub maintains the master debt ledger and event log. Nodes operate independently between syncs and reconcile on a schedule (configurable: real-time WebSocket for active nodes, polling for others).

### Narrative Integration

Federation isn't bolted on — it's the Low getting bigger. When a new node joins, in-fiction a new passage has opened in the deep Warrens connecting to another city's underground. The first players to traverse it are explorers. Trade routes follow. Then faction politics. Then conflict.

---

## TECHNICAL STACK (Preliminary)

- **Language:** Go
- **Connection:** SSH (primary), Telnet (legacy), WebSocket (browser terminal)
- **Database:** SQLite (single-node), PostgreSQL (federation hub)
- **Terminal rendering:** ANSI escape codes, 80x24 minimum
- **State sync:** REST API + optional WebSocket for real-time federation
- **Deployment:** Single binary, zero runtime dependencies

---

## DEVELOPMENT ROADMAP

### Phase 1: Core Game (Solo)
- [ ] Connection server (SSH + terminal handling)
- [ ] Character creation and persistence
- [ ] The Threshold (tutorial sequence)
- [ ] The Lanternmarket (hub + economy)
- [ ] The Warrens (exploration/combat loop)
- [ ] Daily action system
- [ ] Favor/debt economy
- [ ] Basic NPC interactions
- [ ] ANSI art: title screen, district headers, key scenes
- [ ] Death and recovery mechanics

### Phase 2: Depth
- [ ] The Rafters (faction headquarters, housing)
- [ ] The Gutters (transit system, water encounters)
- [ ] Faction system (reputation, quests, alignment)
- [ ] The Vaults (faction strongholds, gated content)
- [ ] NPC storylines with daily advancement
- [ ] Player-to-player interactions (messaging, favor trades, PvP)
- [ ] Contract system (player-posted jobs)
- [ ] Standing system and Ledger attention mechanics

### Phase 3: Endgame
- [ ] The Archive (research system, Knowledge fragments)
- [ ] Anomaly system in the Warrens
- [ ] Charter/Ledger lore content
- [ ] The Deep (endgame zone)
- [ ] Endgame confrontation sequence (all four paths)
- [ ] Post-ending game state changes
- [ ] Character reset / New Game+ mechanics

### Phase 4: Federation
- [ ] Node registration and authentication
- [ ] State sync protocol (REST API)
- [ ] Inter-node debt tracking
- [ ] Cross-node travel mechanics
- [ ] Event broadcast system
- [ ] Hub dashboard / admin interface
- [ ] Federation protocol documentation

### Phase 5: Community
- [ ] Public release and documentation
- [ ] Sysop setup guide
- [ ] Game extension API (custom content / IGM equivalent)
- [ ] ANSI art toolkit and contribution guide
- [ ] Node directory / network status page

---

## OPEN QUESTIONS

- What is the city's history in more detail? Who were the original Charter signatories? What did they bargain with (each other? something older?)?
- How literally does the Ledger manifest in daily gameplay? Is it a background system the player infers, or does it communicate directly?
- What creatures inhabit the Warrens and the Deep? What's the bestiary's tone — mundane-but-wrong, or overtly supernatural?
- Should the game support IGM-style extensions from the start, or is that a post-launch concern?
- What is the game's name as presented to players? "Dornhaven" is the city. "The Low" is the setting. The game title could be either, or something else entirely.

---

*Document version 0.1 — Initial design. Everything above is subject to iteration.*
