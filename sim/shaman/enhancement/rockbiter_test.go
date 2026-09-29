package enhancement

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

// Rockbiter Weapon rank 7 is a permanent +653 melee attack power aura (client 16313), not a proc.
// It stacks with Windfury Totem, since it is not in the totem's category, and a second Rockbiter
// weapon does not add it twice.
func TestRockbiterWeaponAddsAttackPower(t *testing.T) {
	attackPower := func(mh, oh proto.ShamanImbue, totem bool) float64 {
		player := &proto.Player{
			Name: "Shaman", Class: proto.Class_ClassShaman, Race: proto.Race_RaceOrc,
			Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: 12784}}},
			Buffs:     &proto.IndividualBuffs{}, Consumables: &proto.ConsumesSpec{},
			Spec: &proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{
				Options: &proto.EnhancementShaman_Options{
					ClassOptions: &proto.ShamanOptions{ImbueMh: mh},
					ImbueOh:      oh,
				},
			}},
			Rotation: core.APLRotationFromJsonString(`{"type":"TypeAPL","priorityList":[]}`),
		}
		sim := core.NewSim(&proto.RaidSimRequest{
			SimOptions: &proto.SimOptions{RandomSeed: 1},
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{WindfuryTotem: totem}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
		}, simsignals.CreateSignals())
		sim.Reset()
		return sim.Raid.Parties[0].Players[0].GetCharacter().GetStat(stats.AttackPower)
	}
	const none, rockbiter = proto.ShamanImbue_NoImbue, proto.ShamanImbue_RockbiterWeapon
	base := attackPower(none, none, false)
	if got := attackPower(rockbiter, none, false) - base; got < 652.99 || got > 653.01 {
		t.Errorf("Rockbiter main hand adds %v attack power, want 653", got)
	}
	// Forever shamans cannot dual wield, so the two-hander leaves the off-hand imbue without a weapon.
	if got := attackPower(rockbiter, rockbiter, false) - base; got < 652.99 || got > 653.01 {
		t.Errorf("Rockbiter in both hands adds %v attack power, want 653", got)
	}
	if got := attackPower(proto.ShamanImbue_WindfuryWeapon, none, false) - base; got != 0 {
		t.Errorf("Windfury Weapon adds %v standing attack power, want 0", got)
	}
}
