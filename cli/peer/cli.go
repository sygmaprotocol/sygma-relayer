// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package peer

import (
	"github.com/spf13/cobra"
)

var PeerCLI = &cobra.Command{
	Use:   "peer",
	Short: "libp2p peer related commands",
}

func init() {
	PeerCLI.AddCommand(peerInfoCMD)
	PeerCLI.AddCommand(generateKeyCMD)
}
