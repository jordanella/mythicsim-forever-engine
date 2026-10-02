package enhancement

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// The shipped rotation casts Lightning Bolt at 5 Maelstrom Weapon stacks. A build without the talent
// has no such aura; the condition used to drop out and leave Lightning Bolt cast on every GCD.
func TestNoMaelstromWeaponNoLightningBolt(t *testing.T) {
	stacks := &proto.APLValue{Value: &proto.APLValue_Cmp{Cmp: &proto.APLValueCompare{
		Op:  proto.APLValueCompare_OpGe,
		Lhs: &proto.APLValue{Value: &proto.APLValue_AuraNumStacks{AuraNumStacks: &proto.APLValueAuraNumStacks{AuraId: core.ActionID{SpellID: 408505}.ToProto()}}},
		Rhs: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: "5"}}},
	}}}
	player := &proto.Player{
		Name: "enh", Class: proto.Class_ClassShaman, Race: proto.Race_RaceOrc,
		TalentsString: "05323303001-0530300310051-05005", // 21 Enhancement on the UI presets: no Maelstrom Weapon
		Equipment:     &proto.EquipmentSpec{Items: []*proto.ItemSpec{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: 17182}}},
		Spec:          &proto.Player_EnhancementShaman{EnhancementShaman: &proto.EnhancementShaman{Options: &proto.EnhancementShaman_Options{ClassOptions: &proto.ShamanOptions{}}}},
		Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{{Action: &proto.APLAction{
			Condition: stacks,
			Action:    &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: core.ActionID{SpellID: 15208}.ToProto()}},
		}}}},
	}
	raid := &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: &proto.PartyBuffs{}}}, Buffs: &proto.RaidBuffs{}, Debuffs: &proto.Debuffs{}, NumActiveParties: 1}
	res := core.RunRaidSim(&proto.RaidSimRequest{Raid: raid, Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{Iterations: 5, RandomSeed: 1}})
	if res.Error != nil {
		t.Fatal(res.Error.Message)
	}
	for _, a := range res.RaidMetrics.Parties[0].Players[0].Actions {
		for _, tgt := range a.Targets {
			if a.Id.GetSpellId() == 15208 && tgt.Casts > 0 {
				t.Fatalf("Lightning Bolt cast %d times without Maelstrom Weapon", tgt.Casts)
			}
		}
	}
}
