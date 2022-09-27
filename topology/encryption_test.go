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

func (s *AESEncryptionTestSuite) Test_EncryptionDecryption() {
	rawTopology := topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "peerAddress1"},
			{PeerAddress: "peerAddress2"},
		},
		Threshold: "2",
	}
	data, _ := json.Marshal(rawTopology)

	encryptedData := s.aesEncryption.Encrypt(data)
	decryptedData := s.aesEncryption.Decrypt(encryptedData)

	s.Equal(data, decryptedData)
}
