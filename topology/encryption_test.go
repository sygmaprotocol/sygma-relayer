// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package topology_test

import (
	"encoding/json"
	"testing"

	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/stretchr/testify/suite"
)

type AESEncryptionTestSuite struct {
	suite.Suite
	aesEncryption *topology.AESEncryption
}

func TestRunAESEncryptionTestSuite(t *testing.T) {
	suite.Run(t, new(AESEncryptionTestSuite))
}

func (s *AESEncryptionTestSuite) SetupTest() {
	cipherKey := []byte("asuperstrong32bitpasswordgohere!")
	s.aesEncryption, _ = topology.NewAESEncryption(cipherKey)
}

func (s *AESEncryptionTestSuite) Test_Decryption() {
	expectedTopology := topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "peerAddress1"},
			{PeerAddress: "peerAddress2"},
		},
		Threshold: "2",
	}

	encryptedData := "313233343536373831323334353637389343bf912f861f9fd58c1bb5ef6236a5e41a4a3f5698c7f01859a38f90c2d943092158bb485f5cc3653aa8dbf254fea4a0d15bc48c70565aed8057d5a1e2f4de999163b035ec30ce989295535197b6fed02b9584962793058b"
	decryptedData := s.aesEncryption.Decrypt(encryptedData)

	decryptedTopology := topology.RawTopology{}
	json.Unmarshal(decryptedData, &decryptedTopology)

	s.Equal(expectedTopology, decryptedTopology)
}
