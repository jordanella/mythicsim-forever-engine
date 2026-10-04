package core

// Compiling each priority item's readiness check.
//
// An item can run when its condition holds and its action is ready. For a cast, two cheaper checks
// can rule that out first: the spell's timers (on cooldown or behind the GCD) and, for an energy,
// rage or focus cost, whether it can be afforded. Which is worth asking first depends on how the
// rotation plays: a cooldown-bound spec is mostly turned away by timers, a rogue by energy, a Fury
// warrior by its own conditions. The first iteration asks every check and counts which turn each item
// away; the next compiles each item into a function that asks them in the cheapest order for those
// counts, and later iterations only call it. Every check only reads (a mana cost, which also opens
// and closes out-of-mana stretches, is left to the cast check), and the cast check still comes last,
// so the order never changes an answer, only how quickly "not ready" is found.

// What an item counted in its first iteration.
type aplReadyCounts struct {
	timerSpell *Spell // the spell, when it has a cooldown
	costSpell  *Spell // the spell, when it costs energy, rage or focus

	checks           int64
	timerRejects     int64
	costRejects      int64
	conditionRejects int64
}

// Gives an item its first readiness function. With calibrate, a cast with a check worth trying first
// counts its checks until compileReady; without, it asks its cooldown first, as for an item never
// asked. Other items get the plain condition-then-action check.
func (action *APLAction) prepareReady(calibrate bool) {
	var spell *Spell
	switch impl := action.impl.(type) {
	case *APLActionCastSpell:
		spell = impl.spell
	case *APLActionCastFriendlySpell:
		spell = impl.spell
	case *APLActionChannelSpell:
		spell = impl.spell
	}
	counts := &aplReadyCounts{}
	if spell != nil && (spell.CD.Timer != nil || spell.SharedCD.Timer != nil) {
		counts.timerSpell = spell
	}
	if spell != nil && spell.Cost != nil {
		switch spell.Cost.ResourceCostImpl.(type) {
		case *EnergyCost, *RageCost, *FocusCost:
			counts.costSpell = spell
		}
	}
	action.counts = nil
	if !calibrate || (counts.timerSpell == nil && counts.costSpell == nil) {
		action.ready = action.compiledReady(counts.timerSpell, nil, false)
		return
	}
	action.counts = counts
	action.ready = action.countingReady
}

// The first iteration's check: asks the timers, the cost and the condition independently and counts
// each that says no. The answer is the same as the plain check's, since timers or cost that say no
// mean the cast check would too, and the cast check runs only when all three pass.
func (action *APLAction) countingReady(sim *Simulation) bool {
	counts := action.counts
	counts.checks++
	blocked := false
	if counts.timerSpell != nil && counts.timerSpell.timersBlockQueue(sim) {
		counts.timerRejects++
		blocked = true
	}
	if counts.costSpell != nil && counts.costSpell.cannotAffordNonMana() {
		counts.costRejects++
		blocked = true
	}
	conditionHolds := action.condition == nil || action.condition.GetBool(sim)
	if !conditionHolds {
		counts.conditionRejects++
	}
	return !blocked && conditionHolds && action.impl.IsReady(sim)
}

// Rough costs of each check, in units of a timer comparison. A condition costs about one per value
// in its tree; the cast check runs the timers, the unit and target checks, the cast conditions and the
// cost.
const (
	readyCostTimers = 1.0
	readyCostCost   = 2.0
	readyCostCast   = 8.0
	readyCostImpl   = 4.0
)

// Turns the first iteration's counts into a compiled check. Of the cheap checks this item has, it
// tries each subset in each order ahead of the condition, and keeps the one with the lowest expected
// cost, treating the checks as independent. An item never asked keeps the cooldown-first order.
func (action *APLAction) compileReady() {
	counts := action.counts
	if counts == nil {
		return
	}
	action.counts = nil
	hasTimers, hasCost := counts.timerSpell != nil, counts.costSpell != nil
	if counts.checks == 0 {
		action.ready = action.compiledReady(counts.timerSpell, nil, false)
		return
	}

	n := float64(counts.checks)
	pTimers, pCost, pCondition := float64(counts.timerRejects)/n, float64(counts.costRejects)/n, float64(counts.conditionRejects)/n
	costCondition := 0.0
	if action.condition != nil {
		costCondition = float64(1 + len(action.GetAllAPLValues()))
	}
	costLast := readyCostImpl
	if _, ok := action.impl.(*APLActionCastSpell); ok {
		costLast = readyCostCast
	}

	type step struct{ cost, reject float64 }
	expected := func(steps ...step) float64 {
		total, reach := 0.0, 1.0
		for _, s := range steps {
			total += reach * s.cost
			reach *= 1 - s.reject
		}
		return total + reach*costLast
	}
	timers, cost, condition := step{readyCostTimers, pTimers}, step{readyCostCost, pCost}, step{costCondition, pCondition}

	type plan struct {
		timers, cost, costFirst bool
		expected                float64
	}
	best := plan{expected: expected(condition)}
	consider := func(p plan, steps ...step) {
		if p.expected = expected(steps...); p.expected < best.expected {
			best = p
		}
	}
	if hasTimers {
		consider(plan{timers: true}, timers, condition)
	}
	if hasCost {
		consider(plan{cost: true}, cost, condition)
	}
	if hasTimers && hasCost {
		consider(plan{timers: true, cost: true}, timers, cost, condition)
		consider(plan{timers: true, cost: true, costFirst: true}, cost, timers, condition)
	}
	var timerSpell, costSpell *Spell
	if best.timers {
		timerSpell = counts.timerSpell
	}
	if best.cost {
		costSpell = counts.costSpell
	}
	action.ready = action.compiledReady(timerSpell, costSpell, best.costFirst)
}

// The compiled check for one order: the timers of timerSpell and the cost of costSpell, each when
// given, in the order costFirst picks, then the condition, then the action.
func (action *APLAction) compiledReady(timerSpell, costSpell *Spell, costFirst bool) func(*Simulation) bool {
	impl, condition := action.impl, action.condition
	timers, cost := timerSpell != nil, costSpell != nil
	switch {
	case !timers && !cost && condition == nil:
		return impl.IsReady
	case !timers && !cost:
		return func(sim *Simulation) bool {
			return condition.GetBool(sim) && impl.IsReady(sim)
		}
	case timers && !cost && condition == nil:
		return func(sim *Simulation) bool {
			return !timerSpell.timersBlockQueue(sim) && impl.IsReady(sim)
		}
	case timers && !cost:
		return func(sim *Simulation) bool {
			return !timerSpell.timersBlockQueue(sim) && condition.GetBool(sim) && impl.IsReady(sim)
		}
	case !timers && condition == nil:
		return func(sim *Simulation) bool {
			return !costSpell.cannotAffordNonMana() && impl.IsReady(sim)
		}
	case !timers:
		return func(sim *Simulation) bool {
			return !costSpell.cannotAffordNonMana() && condition.GetBool(sim) && impl.IsReady(sim)
		}
	case costFirst && condition == nil:
		return func(sim *Simulation) bool {
			return !costSpell.cannotAffordNonMana() && !timerSpell.timersBlockQueue(sim) && impl.IsReady(sim)
		}
	case costFirst:
		return func(sim *Simulation) bool {
			return !costSpell.cannotAffordNonMana() && !timerSpell.timersBlockQueue(sim) && condition.GetBool(sim) && impl.IsReady(sim)
		}
	case condition == nil:
		return func(sim *Simulation) bool {
			return !timerSpell.timersBlockQueue(sim) && !costSpell.cannotAffordNonMana() && impl.IsReady(sim)
		}
	default:
		return func(sim *Simulation) bool {
			return !timerSpell.timersBlockQueue(sim) && !costSpell.cannotAffordNonMana() && condition.GetBool(sim) && impl.IsReady(sim)
		}
	}
}

// Gives every item its first readiness function, once the rotation is built and its conditions final.
func (rot *APLRotation) prepareReadyChecks() {
	for _, action := range rot.allPrepullActions() {
		action.prepareReady(false)
	}
	for _, action := range rot.allAPLActions() {
		action.prepareReady(true)
	}
}

// Compiles every item that counted its checks, once the first iteration is over.
func (rot *APLRotation) compileReadyChecks() {
	if rot.readyCompiled {
		return
	}
	calibrating, counted := false, false
	for _, action := range rot.allAPLActions() {
		if action.counts != nil {
			calibrating = true
			if action.counts.checks > 0 {
				counted = true
				break
			}
		}
	}
	if !calibrating {
		// Nothing has checks worth ordering, so there is nothing to compile, now or later.
		rot.readyCompiled = true
		return
	}
	if !counted {
		return
	}
	for _, action := range rot.allAPLActions() {
		action.compileReady()
	}
	rot.readyCompiled = true
}
