package core

import (
	"time"
)

type QueuedSpell struct {
	spell       *Spell
	target      *Unit
	queueAction *PendingAction

	// Stores the time at which QueueSpell() was called to ensure that only one spell can be queued up per timestep
	QueueInitiatedAt time.Duration
}

// Models the use of /cqs macros to change which spell should be cast at the last minute
func (qs *QueuedSpell) Cancel(sim *Simulation) {
	if (qs.queueAction != nil) && !qs.queueAction.consumed {
		qs.queueAction.Cancel(sim)
		qs.queueAction = nil
	}

	qs.QueueInitiatedAt = -NeverExpires
}

func (unit *Unit) CancelQueuedSpell(sim *Simulation) {
	if unit.QueuedSpell != nil {
		unit.QueuedSpell.Cancel(sim)
	}
}

func (qs *QueuedSpell) InitiateQueue(sim *Simulation, spell *Spell, target *Unit, executeAt time.Duration) {
	qs.Cancel(sim)
	qs.spell = spell
	qs.target = target

	if qs.queueAction == nil {
		pa := &PendingAction{
			NextActionAt: executeAt,
			Priority:     ActionPriorityGCD,

			OnAction: func(sim *Simulation) {
				qs.spell.Cast(sim, qs.target)
			},
		}
		qs.queueAction = pa
	} else {
		qs.queueAction.cancelled = false
		qs.queueAction.NextActionAt = executeAt
	}

	sim.AddPendingAction(qs.queueAction)
	qs.QueueInitiatedAt = sim.CurrentTime
}

func (unit *Unit) QueueSpell(sim *Simulation, spell *Spell, target *Unit, queueAt time.Duration) {
	if unit.QueuedSpell == nil {
		qs := &QueuedSpell{}
		unit.QueuedSpell = qs
	}

	fireAt := queueAt + time.Duration(1) // 1ns artificial delay guarantees last-second cancellation if desired
	unit.QueuedSpell.InitiateQueue(sim, spell, target, fireAt)

	if sim.Log != nil {
		unit.Log(sim, "Queueing up %s to cast at %s.", spell.ActionID, fireAt)
	}
}

// Enforce only one queued spell per timestep
func (unit *Unit) CanQueueSpell(sim *Simulation) bool {
	return (unit.QueuedSpell == nil) || (unit.QueuedSpell.QueueInitiatedAt != sim.CurrentTime)
}

// Returns whether the spell could be queued by the player at the current time using the
// game's spell queueing functionality. Assumes the maximum spell queue window of 400ms
// that the game allows.
func (spell *Spell) CanQueue(sim *Simulation, target *Unit) bool {
	if spell == nil {
		return false
	}

	// The timer checks come first: they are a few comparisons, and most rotation checks of a spell
	// find it on cooldown or behind the GCD. CanCompleteCast runs the cast conditions and the cost
	// check, and the mana cost check also opens and closes out-of-mana stretches, so those count
	// only spells that are otherwise ready.
	if spell.timersBlockQueue(sim) {
		return false
	}

	return spell.CanCompleteCast(sim, target, false)
}

// Whether the spell's timers alone keep it from being cast or queued now: an un-equipped item, a
// channel still running, or the GCD or its cooldown more than one spell queue window away.
func (spell *Spell) timersBlockQueue(sim *Simulation) bool {
	if spell.Flags.Matches(SpellFlagSwapped) {
		return true
	}

	// Apply SQW leniency to any pending hardcasts
	if spell.Flags.Matches(SpellFlagChanneled) && spell.Unit.Hardcast.Expires > sim.CurrentTime+MaxSpellQueueWindow {
		return true
	}

	// Apply SQW leniency to GCD timer
	if spell.DefaultCast.GCD > 0 && spell.Unit.GCD.TimeToReady(sim) > MaxSpellQueueWindow {
		return true
	}

	// Spells that are within one SQW of coming off cooldown can also be queued
	return MaxTimeToReady(spell.CD.Timer, spell.SharedCD.Timer, sim) > MaxSpellQueueWindow
}

// Whether the spell's energy, rage or focus cost alone keeps it from being cast now: the same answer
// as the cost check in CanCompleteCast, without writing CurCast. A mana cost is left to the cast
// check, which also opens and closes out-of-mana stretches.
func (spell *Spell) cannotAffordNonMana() bool {
	switch spell.Cost.ResourceCostImpl.(type) {
	case *EnergyCost:
		return spell.Unit.CurrentEnergy() < spell.Cost.GetCurrentCost()
	case *RageCost:
		return spell.Unit.CurrentRage() < spell.Cost.GetCurrentCost()
	case *FocusCost:
		return spell.Unit.currentFocus < spell.Cost.GetCurrentCost()
	}
	return false
}

// Helper function for APL checks to prevent infinite loops
func (spell *Spell) CanCastOrQueue(sim *Simulation, target *Unit) bool {
	return spell.Unit.CanQueueSpell(sim) && spell.CanQueue(sim, target)
}

func (spell *Spell) CastOrQueue(sim *Simulation, target *Unit) {
	if spell.CanCast(sim, target) {
		spell.Cast(sim, target)
	} else if spell.CanQueue(sim, target) {
		// Determine which timer the spell is waiting on
		queueTime := max(spell.Unit.Hardcast.Expires, BothTimersReadyAt(spell.CD.Timer, spell.SharedCD.Timer))

		if (spell.DefaultCast.GCD > 0) || spell.Flags.Matches(SpellFlagMCD) {
			queueTime = max(queueTime, spell.Unit.GCD.ReadyAt())
		}

		// Schedule the cast to go off without delay
		spell.Unit.QueueSpell(sim, spell, target, queueTime)
	} else {
		// Fallback to make sure there is always log output
		spell.Cast(sim, target)
	}
}
