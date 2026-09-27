// Rebuild the embedded binary databases after a reviewed JSON data migration.
package main

import (
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
)

func main() {
	for _, name := range []string{"db", "leftover_db"} {
		path := filepath.Join("assets/database", name)
		raw, err := os.ReadFile(path + ".json")
		if err != nil {
			panic(err)
		}
		db := &proto.UIDatabase{}
		if err = protojson.Unmarshal(raw, db); err != nil {
			panic(err)
		}
		raw, err = pb.MarshalOptions{Deterministic: true}.Marshal(db)
		if err != nil {
			panic(err)
		}
		if err = os.WriteFile(path+".bin", raw, 0644); err != nil {
			panic(err)
		}
	}
}
