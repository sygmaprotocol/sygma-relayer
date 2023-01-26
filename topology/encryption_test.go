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
	cipherKey := []byte("v8y/B?E(H+MbQeTh")
	s.aesEncryption, _ = topology.NewAESEncryption(cipherKey)
}

func (s *AESEncryptionTestSuite) Test_Decryption() {
	expectedTopology := topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "2",
	}

	encryptedData := "12345678123456786c49ea9bdd0d37f3ad3d266b4ef5d6ef243027b25bd5092091bcd26488e185c4f466c6795535593dc41b03fcd8997985a78bc784c9f561bac89683a0170e5632ec0fc9237a97ebe38f783067d7f0d19dfe708349ca10759e6091228de7899ee10d679c8b444132bfd8106e1d28e944facb21be60182b7c069f264244ab545871ee6d15a1f070cacada34647bc7d2404384b3ee54b9058ec14ae9e017610f392adeb05d33d524e10043908887d932e5a974c8200639c0dc8d77e1cfb65ecbd2f9c731c61212d1a928b5436f3540cfbd981070b5567ced664ef20cc795ebb792231df08f05987a5d9458664d34666995fb15a969440dfd28db35fbd79f9e11cbcfd42409259c4bb1006c0907d2d4b170698e90452ead9ab7f4e41309fe8c586ccee54cc9cfaf3ac22d00b6c6f583a3f7a1fe3ddd470aa12ad9cf63f072798dadb5a21004529fec4a5914d68a18fd0b3fc33079d4ff09af44416b732f024b75b40dd0"
	decryptedData := s.aesEncryption.Decrypt(encryptedData)

	decryptedTopology := topology.RawTopology{}
	_ = json.Unmarshal(decryptedData, &decryptedTopology)

	s.Equal(expectedTopology, decryptedTopology)
}
