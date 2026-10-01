package buffs

import "github.com/wowsims/forever/sim/core"

// AirTotemCategory is the one air totem a party holds. Since client build 70009 Windfury Totem is a
// party aura like Grace of Air, and the build's patch notes allow one air totem per party, so a
// Windfury Totem and a Grace of Air Totem are never both up. Each air totem aura joins this category
// as a single-aura effect that carries no stat or proc of its own: the category only decides which
// aura stays up, and the aura that loses is deactivated, which removes whatever it granted.
//
// Windfury Weapon is not a member. It disables only the Windfury Totem proc its own wielder would get
// (WindfuryTotemCategory) and leaves Grace of Air alone, so a Windfury Weapon works under either totem.
const AirTotemCategory = "AirTotem"

// What each air totem bids for the slot. A higher bid replaces a lower one that is up and is refused
// by a higher one that is up. A totem the player casts outbids the one a party assumption supplies:
// the cast is the later and deliberate placement, the same way a new air totem replaces the shaman's
// previous one (Shaman.AirTotemAura). When two party buffs are both set, Windfury Totem outbids Grace
// of Air; MythicSim's worker never sends both, so this only fixes the outcome for a request that does.
const (
	AirTotemBidPartyGraceOfAir float64 = iota + 1
	AirTotemBidPartyWindfury
	AirTotemBidCastGraceOfAir
	AirTotemBidCastWindfury
)

// JoinAirTotemSlot makes aura an air totem that bids bid for the party's one slot.
func JoinAirTotemSlot(aura *core.Aura, bid float64) {
	aura.NewExclusiveEffect(AirTotemCategory, true, core.ExclusiveEffect{Priority: bid})
}
