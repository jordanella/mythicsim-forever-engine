package core

import (
	"time"

	"github.com/wowsims/forever/sim/core/proto"
)

// Caching conditions that depend only on the fight's time.
//
// A condition such as "Remaining Time <= 30s OR Remaining Time >= 140s", common on cooldowns held for
// the end of a fight, gives the same answer for minutes at a time, yet a rotation waiting for rage or
// energy asks it on every look. Each comparison of the remaining or current time with a constant can
// change its answer only at the one time where the two are equal, so a condition built from such
// comparisons, constants, And, Or and Not keeps its answer until the next of those times. The cache
// asks the condition as before and reuses the answer until then, so the answers never change.

// A condition, or part of one, that depends only on the fight's time.
type aplValueTimeCache struct {
	DefaultAPLValueImpl
	inner  APLValue
	leaves []*APLValueCompare

	value bool
	until time.Duration // value holds before this time
}

func (cache *aplValueTimeCache) Type() proto.APLValueType {
	return proto.APLValueType_ValueTypeBool
}

// Reports no inner values, so the rotation's check-cost estimate counts the cache as one value.
func (cache *aplValueTimeCache) GetInnerValues() []APLValue {
	return nil
}

func (cache *aplValueTimeCache) GetBool(sim *Simulation) bool {
	if sim.CurrentTime < cache.until {
		return cache.value
	}
	cache.value = cache.inner.GetBool(sim)
	cache.until = NeverExpires
	for _, leaf := range cache.leaves {
		cache.until = min(cache.until, nextTimeComparisonChange(sim, leaf))
	}
	return cache.value
}

func (cache *aplValueTimeCache) String() string {
	return cache.inner.String()
}

func (cache *aplValueTimeCache) reset() {
	cache.until = -NeverExpires
}

// The first time after now at which a comparison of the remaining or current time with a constant
// can change its answer, or now when it may change at any time.
func nextTimeComparisonChange(sim *Simulation, compare *APLValueCompare) time.Duration {
	timeValue, constValue := compare.lhs, compare.rhs
	if _, ok := timeValue.(*APLValueConst); ok {
		timeValue, constValue = constValue, timeValue
	}
	c := constValue.GetDuration(sim)

	// The time at which the two sides are equal: the answer is fixed before it, at it, and after it.
	var equalAt time.Duration
	switch timeValue.(type) {
	case *APLValueRemainingTime:
		if sim.Encounter.EndFightAtHealth > 0 && sim.Encounter.DurationIsEstimate {
			// Estimated from damage done after the first few seconds.
			return sim.CurrentTime
		}
		equalAt = sim.Duration - c
	case *APLValueCurrentTime:
		equalAt = c
	}

	switch {
	case sim.CurrentTime < equalAt:
		return equalAt
	case sim.CurrentTime == equalAt:
		return equalAt + 1
	default:
		return NeverExpires
	}
}

// The comparisons a value's answer depends on, when it depends only on the fight's time.
func timeOnlyComparisons(value APLValue) ([]*APLValueCompare, bool) {
	switch value := value.(type) {
	case *APLValueConst:
		return nil, value.valType == proto.APLValueType_ValueTypeBool
	case *APLValueCompare:
		if isTimeComparison(value) {
			return []*APLValueCompare{value}, true
		}
	case *APLValueAnd:
		return timeOnlyComparisonsOf(value.vals)
	case *APLValueOr:
		return timeOnlyComparisonsOf(value.vals)
	case *APLValueNot:
		return timeOnlyComparisons(value.val)
	}
	return nil, false
}

func timeOnlyComparisonsOf(values []APLValue) ([]*APLValueCompare, bool) {
	var leaves []*APLValueCompare
	for _, value := range values {
		valueLeaves, ok := timeOnlyComparisons(value)
		if !ok {
			return nil, false
		}
		leaves = append(leaves, valueLeaves...)
	}
	return leaves, true
}

// Whether a comparison is of the remaining or current time with a constant duration.
func isTimeComparison(compare *APLValueCompare) bool {
	if compare.lhs.Type() != proto.APLValueType_ValueTypeDuration {
		return false
	}
	isTime := func(value APLValue) bool {
		switch value.(type) {
		case *APLValueRemainingTime, *APLValueCurrentTime:
			return true
		}
		return false
	}
	isConst := func(value APLValue) bool {
		_, ok := value.(*APLValueConst)
		return ok
	}
	return (isTime(compare.lhs) && isConst(compare.rhs)) || (isConst(compare.lhs) && isTime(compare.rhs))
}

// Rewrites a condition so each largest part that depends only on the fight's time is cached, and
// drops constant operands that cannot change an And or Or. Returns the condition to use.
func (rot *APLRotation) cacheTimeOnlyConditions(value APLValue) APLValue {
	if value == nil {
		return nil
	}
	if leaves, ok := timeOnlyComparisons(value); ok && len(leaves) > 0 {
		cache := &aplValueTimeCache{inner: value, leaves: leaves}
		cache.reset()
		rot.timeCaches = append(rot.timeCaches, cache)
		return cache
	}

	switch value := value.(type) {
	case *APLValueAnd:
		value.vals = rot.cacheTimeOnlyOperands(value.vals, true)
		if len(value.vals) == 1 {
			return value.vals[0]
		}
	case *APLValueOr:
		value.vals = rot.cacheTimeOnlyOperands(value.vals, false)
		if len(value.vals) == 1 {
			return value.vals[0]
		}
	case *APLValueNot:
		value.val = rot.cacheTimeOnlyConditions(value.val)
	}
	return value
}

// Rewrites the operands of an And (identity true) or Or (identity false) in place, dropping any
// constant equal to the identity, which cannot change the answer. Operands that need no change keep
// their slice, so a rotation without time-only conditions is left exactly as built.
func (rot *APLRotation) cacheTimeOnlyOperands(values []APLValue, identity bool) []APLValue {
	isIdentity := func(value APLValue) bool {
		constValue, ok := value.(*APLValueConst)
		return ok && constValue.valType == proto.APLValueType_ValueTypeBool && constValue.boolVal == identity
	}
	dropped := 0
	for i, value := range values {
		if isIdentity(value) {
			dropped++
			continue
		}
		values[i] = rot.cacheTimeOnlyConditions(value)
	}
	if dropped == 0 || dropped == len(values) {
		return values
	}
	kept := make([]APLValue, 0, len(values)-dropped)
	for _, value := range values {
		if !isIdentity(value) {
			kept = append(kept, value)
		}
	}
	return kept
}
