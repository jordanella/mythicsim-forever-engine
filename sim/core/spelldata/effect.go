package spelldata

import (
	"math"
	"time"

	"github.com/wowsims/forever/sim/core"
)

// Whether the effect's class mask names the spell: the spell is in the effect's family and carries one of
// the mask's bits. An effect with an empty mask names none, which the client uses to switch an effect off.
func (e *Effect) Covers(s *Spell) bool {
	return e.ClassFlags.Matches(s.ClassFlags)
}

// The client's own number, before any unit the aura gives it.
func (e *Effect) BaseValue() float64 {
	return e.BasePoints
}

// The client states a percentage as an integer: Improved Righteous Fury reads 16, not 0.16.
func (e *Effect) Percent() float64 {
	return e.BasePoints / 100
}

// The client states rage on a 0-1000 bar: a -30 cost modifier is 3 rage, Charge's energize of 150 is
// 15.
func (e *Effect) Tenths() float64 {
	return e.BasePoints / 10
}

// The value read as a time, which is the unit a duration or delay modifier states it in.
func (e *Effect) TimeValue() time.Duration {
	return core.DurationFromMillis(e.BasePoints)
}

// EffectAuraPeriod: how long one tick of the aura lasts.
func (e *Effect) Period() time.Duration {
	return core.DurationFromMillis(e.PeriodMs)
}

func (e *Effect) Coeff() float64 {
	return e.SPCoef
}

func (e *Effect) APCoeff() float64 {
	return e.APCoef
}

// The spell this effect fires, or Nil where it fires nothing.
func (e *Effect) Trigger() *Spell {
	return Find(e.TriggerID)
}

// The amount at a caster level: the base points plus the per-level gain over the spell's own level,
// stopping at the level the spell stops scaling at. Both levels are the owning spell's, stamped onto
// the effect at generation, so an effect answers the same amount however the caller reached it.
//
// The base keeps the client's own EffectBasePointsF, which is fractional on about one effect in 55,
// and the per-level gain is added on top of the fraction before the whole is floored.
//
// float32 is load-bearing: EffectRealPointsPerLevel is a float32 widened into the DB
// (3.79999995231628), and multiplying in float64 moves the result off the tooltip on six rows. The
// client resolves the amount to a whole number, which is the floor below.
func (e *Effect) Average(level int32) float64 {
	// MaxLevel 0 is the client's "no cap", which a spell with no SpellLevels row carries.
	lvl := level
	if e.MaxLevel > 0 && int32(e.MaxLevel) < lvl {
		lvl = int32(e.MaxLevel)
	}

	delta := lvl - int32(e.SpellLevel)
	if delta < 0 {
		delta = 0
	}

	base := float32(e.BasePoints) + float32(float32(delta)*float32(e.PPL))
	return math.Floor(float64(base))
}

// The low end of the roll. EffectVariance is the spread around the average, so an effect that does not
// roll answers the average at both ends.
func (e *Effect) Min(level int32) float64 {
	return e.Average(level) * (1 - e.Variance/2)
}

func (e *Effect) Max(level int32) float64 {
	return e.Average(level) * (1 + e.Variance/2)
}

// The amount for one cast: rolled where the client states a spread, the average where it does not.
func (e *Effect) Roll(sim *core.Simulation, level int32) float64 {
	average := e.Average(level)
	if e.Variance == 0 {
		return average
	}
	return sim.Roll(average*(1-e.Variance/2), average*(1+e.Variance/2))
}
