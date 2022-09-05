package common_test

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/sygma-relayer/tss/common"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/suite"
)

type IsParticipantTestSuite struct {
	suite.Suite
}

func TestRunIsParticipantTestSuite(t *testing.T) {
	suite.Run(t, new(IsParticipantTestSuite))
}

func (s *IsParticipantTestSuite) Test_ValidParticipant() {
	party1 := tss.NewPartyID("id1", "id1", big.NewInt(1))
	party2 := tss.NewPartyID("id2", "id2", big.NewInt(2))
	parties := tss.SortedPartyIDs{party1, party2}

	isParticipant := common.IsParticipant(party1, parties)

	s.Equal(true, isParticipant)
}

func (s *IsParticipantTestSuite) Test_InvalidParticipant() {
	party1 := tss.NewPartyID("id1", "id1", big.NewInt(1))
	party2 := tss.NewPartyID("id2", "id2", big.NewInt(2))
	invalidParty := tss.NewPartyID("invalid", "id3", big.NewInt(3))
	parties := tss.SortedPartyIDs{party1, party2}

	isParticipant := common.IsParticipant(invalidParty, parties)

	s.Equal(false, isParticipant)
}

type PeersFromPartiesTestSuite struct {
	suite.Suite
}

func TestRunPeersFromPartiesTestSuite(t *testing.T) {
	suite.Run(t, new(PeersFromPartiesTestSuite))
}

func (s *PeersFromPartiesTestSuite) Test_NoParties() {
	peers, err := common.PeersFromParties([]*tss.PartyID{})

	s.Nil(err)
	s.Equal(peers, []peer.ID{})
}

func (s *PeersFromPartiesTestSuite) Test_InvalidParty() {
	party1 := common.CreatePartyID("invalid")

	_, err := common.PeersFromParties([]*tss.PartyID{party1})

	s.NotNil(err)
}

func (s *PeersFromPartiesTestSuite) Test_ValidParties() {
	party1 := common.CreatePartyID("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	party2 := common.CreatePartyID("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	peerID1, _ := peer.Decode(party1.Id)
	peerID2, _ := peer.Decode(party2.Id)

	peers, err := common.PeersFromParties([]*tss.PartyID{party1, party2})

	s.Nil(err)
	s.Equal(peers, []peer.ID{peerID1, peerID2})
}

type PeersFromIDSTestSuite struct {
	suite.Suite
}

func TestRunPeersFromIDSTestSuite(t *testing.T) {
	suite.Run(t, new(PeersFromIDSTestSuite))
}

func (s *PeersFromIDSTestSuite) Test_NoIDS() {
	peers, err := common.PeersFromIDS([]string{})

	s.Nil(err)
	s.Equal(peers, []peer.ID{})
}

func (s *PeersFromIDSTestSuite) Test_InvalidParty() {
	_, err := common.PeersFromIDS([]string{"invalid"})

	s.NotNil(err)
}

func (s *PeersFromIDSTestSuite) Test_ValidIDS() {
	party1 := common.CreatePartyID("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	party2 := common.CreatePartyID("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	peerID1, _ := peer.Decode(party1.Id)
	peerID2, _ := peer.Decode(party2.Id)

	peers, err := common.PeersFromIDS([]string{party1.Id, party2.Id})

	s.Nil(err)
	s.Equal(peers, []peer.ID{peerID1, peerID2})
}

type SortPeersForSessionTestSuite struct {
	suite.Suite
}

func TestRunSortPeersForSessionTestSuite(t *testing.T) {
	suite.Run(t, new(SortPeersForSessionTestSuite))
}

func (s *SortPeersForSessionTestSuite) Test_NoPeers() {
	sortedPeers := common.SortPeersForSession([]peer.ID{}, "sessioniD")

	s.Equal(sortedPeers, common.SortablePeerSlice{})
}

func (s *SortPeersForSessionTestSuite) Test_ValidPeers() {
	peer1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	peer2, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	peer3, _ := peer.Decode("QmYayosTHxL2xa4jyrQ2PmbhGbrkSxsGM1kzXLTT8SsLVy")
	peers := []peer.ID{peer3, peer2, peer1}

	sortedPeers := common.SortPeersForSession(peers, "sessionID")

	s.Equal(sortedPeers, common.SortablePeerSlice{
		common.PeerMsg{SessionID: "sessionID", ID: peer1},
		common.PeerMsg{SessionID: "sessionID", ID: peer2},
		common.PeerMsg{SessionID: "sessionID", ID: peer3},
	})
}

type PartiesFromPeersTestSuite struct {
	suite.Suite
}

func TestRunPartiesFromPeersTestSuite(t *testing.T) {
	suite.Run(t, new(PartiesFromPeersTestSuite))
}

func (s *PartiesFromPeersTestSuite) Test_ValidPeers() {
	party1 := common.CreatePartyID("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	party1.Index = 1
	party2 := common.CreatePartyID("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	party2.Index = 0
	peerID1, _ := peer.Decode(party1.Id)
	peerID2, _ := peer.Decode(party2.Id)

	sortedParties := common.PartiesFromPeers([]peer.ID{peerID1, peerID2})

	s.Equal(sortedParties, tss.SortedPartyIDs{party2, party1})
}

type ExcludedPeersTestSuite struct {
	suite.Suite
}

func TestRunExcludedPeersTestSuite(t *testing.T) {
	suite.Run(t, new(PartiesFromPeersTestSuite))
}

func (s *PartiesFromPeersTestSuite) Test_ExcludePeers_Excluded() {
	peerID1, _ := peer.Decode("QmcW3oMdSqoEcjbyd51auqC23vhKX6BqfcZcY2HJ3sKAZR")
	peerID2, _ := peer.Decode("QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54")
	peerID3, _ := peer.Decode("QmYayosTHxL2xa4jyrQ2PmbhGbrkSxsGM1kzXLTT8SsLVy")
	peers := []peer.ID{peerID1, peerID2, peerID3}
	excludedPeers := []peer.ID{peerID3, peerID2}

	includedPeers := common.ExcludePeers(peers, excludedPeers)

	s.Equal(includedPeers, peer.IDSlice{peerID1})
}
