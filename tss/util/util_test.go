package util_test

import (
	"testing"

	"github.com/ChainSafe/sygma-relayer/tss/util"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type IsParticipantTestSuite struct {
	suite.Suite
}

func TestRunIsParticipantTestSuite(t *testing.T) {
	suite.Run(t, new(IsParticipantTestSuite))
}

func (s *IsParticipantTestSuite) Test_ValidParticipant() {
	peerID1 := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"
	peerID2 := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF56"
	peers := peer.IDSlice{peer.ID(peerID1), peer.ID(peerID2)}

	isParticipant := util.IsParticipant(peer.ID(peerID1), peers)

	s.Equal(true, isParticipant)
}

func (s *IsParticipantTestSuite) Test_InvalidParticipant() {
	peerID1 := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF54"
	peerID2 := "QmZHPnN3CKiTAp8VaJqszbf8m7v4mPh15M421KpVdYHF56"
	peers := peer.IDSlice{peer.ID(peerID2)}

	isParticipant := util.IsParticipant(peer.ID(peerID1), peers)

	s.Equal(false, isParticipant)
}
