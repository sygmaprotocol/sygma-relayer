package common_test

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/chainbridge-core/tss/common"
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
