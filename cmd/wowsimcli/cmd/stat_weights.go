package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	rootCmd.AddCommand(newStatWeightsCommand())
}

// Stat weights use the same request and result protos as the browser API.
func newStatWeightsCommand() *cobra.Command {
	var inputPath, outputPath string
	var rejectUnknown bool
	command := &cobra.Command{
		Use: "statweights", Short: "calculate character stat weights and EP values",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			raw, err := os.ReadFile(inputPath)
			if err != nil {
				return fmt.Errorf("read stat weights request: %w", err)
			}
			request, err := loadStatWeightsRequest(raw, rejectUnknown)
			if err != nil {
				return err
			}
			result := core.StatWeights(request)
			raw, err = (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(result)
			if err != nil {
				return fmt.Errorf("marshal stat weights: %w", err)
			}
			if outputPath != "" {
				err = os.WriteFile(outputPath, raw, 0644)
			} else {
				_, err = command.OutOrStdout().Write(raw)
			}
			if err != nil {
				return fmt.Errorf("write stat weights: %w", err)
			}
			if result.Error != nil {
				return fmt.Errorf("stat weights failed: %s", result.Error.Message)
			}
			return nil
		},
	}
	command.Flags().StringVar(&inputPath, "infile", "", "input StatWeightsRequest in protojson format")
	command.Flags().StringVar(&outputPath, "outfile", "", "output file, defaults to stdout")
	command.Flags().BoolVar(&rejectUnknown, "strict", false, "reject unknown fields and enum names")
	_ = command.MarkFlagRequired("infile")
	return command
}

func loadStatWeightsRequest(raw []byte, strict bool) (*proto.StatWeightsRequest, error) {
	request := &proto.StatWeightsRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: !strict}).Unmarshal(raw, request); err != nil {
		return nil, fmt.Errorf("read stat weights request: %w", err)
	}
	if request.Player == nil || request.Encounter == nil || request.SimOptions == nil {
		return nil, fmt.Errorf("stat weights require player, encounter and simOptions")
	}
	if request.SimOptions.Iterations < 2 {
		return nil, fmt.Errorf("stat weights require at least two iterations")
	}
	if len(request.StatsToWeigh)+len(request.PseudoStatsToWeigh) == 0 {
		return nil, fmt.Errorf("stat weights require at least one stat to weigh")
	}
	return request, nil
}
