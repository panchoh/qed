package cli

import (
	"encoding/hex"
	"fmt"
	"verifiabledata/balloon"

	"github.com/spf13/cobra"
)

func newMembershipCommand(ctx *Context) *cobra.Command {

	var version uint
	var verify bool
	var key, hyperDigest, historyDigest string

	cmd := &cobra.Command{
		Use:   "membership",
		Short: "Query for membership",
		Long: `Query for membership of an event to the authenticated data structure.
			It also verifies the proofs provided by the server if flag enabled.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if verify {
				if hyperDigest == "" {
					return fmt.Errorf("Error: trying to verify proof without hyper digest")
				}
				if historyDigest == "" {
					return fmt.Errorf("Error: trying to verify proof without history digest")
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Querying key [ %s ] with version [ %d ]\n", key, version)

			event := []byte(key)
			proof, err := ctx.client.Membership(event, version)
			if err != nil {
				return err
			}

			fmt.Printf("Received proof: %+v\n", proof)

			if verify {
				hdBytes, _ := hex.DecodeString(hyperDigest)
				htdBytes, _ := hex.DecodeString(historyDigest)
				commitment := &balloon.Commitment{htdBytes, hdBytes, uint(version)}

				fmt.Printf("Verifying with commitment: \n\tHyperDigest: %s\n\tHistoryDigest: %s\n\tVersion: %d\n",
					hyperDigest, historyDigest, version)
				if proof.Verify(commitment, event) {
					fmt.Println("Verify: OK")
				} else {
					fmt.Println("Verify: KO")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&key, "key", "", "Key to query")
	cmd.Flags().UintVar(&version, "version", 0, "Version to query")
	cmd.Flags().BoolVar(&verify, "verify", false, "Do verify received proof")
	cmd.Flags().StringVar(&hyperDigest, "hyperDigest", "", "Digest of the hyper tree")
	cmd.Flags().StringVar(&historyDigest, "historyDigest", "", "Digest of the history tree")
	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("version")

	return cmd
}