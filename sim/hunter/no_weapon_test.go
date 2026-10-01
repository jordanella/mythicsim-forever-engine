package hunter

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// A hunter with an empty ranged slot, such as a Build Lab character before any gear is picked, has a
// zero-speed ranged weapon. Its auto shot used to reschedule itself at the same instant forever, so
// the iteration never ended and the process ran out of memory. An empty slot has nothing to swing.
func TestHunterWithNoRangedWeaponFinishes(t *testing.T) {
	for _, c := range []struct {
		name  string
		gear  *proto.EquipmentSpec
		style string
	}{
		{"no gear at all", &proto.EquipmentSpec{}, "sv_melee"},
		{"a two-hander and no bow", &proto.EquipmentSpec{Items: []*proto.ItemSpec{
			{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: 12784},
		}}, "sv_melee"},
		{"a bow and no melee weapon", &proto.EquipmentSpec{Items: []*proto.ItemSpec{
			{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: 18713},
		}}, "sv_melee"},
	} {
		t.Run(c.name, func(t *testing.T) {
			player := &proto.Player{
				Name: "Hunter", Class: proto.Class_ClassHunter, Race: proto.Race_RaceHuman,
				TalentsString: SurvivalMeleeTalents,
				Equipment:     c.gear,
				Rotation:      core.GetAplRotation("../../ui/specs/hunter/dps/apls", c.style).Rotation,
				Spec: &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{
					ClassOptions: &proto.HunterOptions{
						Ammo: proto.HunterOptions_Doomshot, QuiverBonus: proto.HunterOptions_Speed15,
						PetType: proto.HunterOptions_Cat, PetAttackSpeed: proto.HunterOptions_OneTwo, PetUptime: 1,
					},
				}}},
			}
			encounter := core.MakeSingleTargetEncounter(0)
			encounter.Duration = 30
			done := make(chan *proto.RaidSimResult, 1)
			go func() {
				done <- core.RunRaidSim(&proto.RaidSimRequest{
					Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
					Encounter:  encounter,
					SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1},
				})
			}()
			select {
			case res := <-done:
				if res.Error != nil {
					t.Fatal(res.Error.Message)
				}
			case <-time.After(20 * time.Second):
				t.Fatal("the sim did not finish: a weaponless slot is swinging at zero interval")
			}
		})
	}
}
