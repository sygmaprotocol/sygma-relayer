package topology

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
)

func NewFixedNetworkTopologyProvider() (NetworkTopologyProvider, error) {
	return fixedTopologyProvider{}, nil
}

type fixedTopologyProvider struct{}

func (p fixedTopologyProvider) NetworkTopology() (NetworkTopology, error) {

	fixedData := `{
    "peers": [
        {"peerAddress": "/dns4/relayer-0.relayer-0/tcp/9000/p2p/QmVuMSb6unWs2m22sgEQF97XvShbrd9JAkX7Kh2xQ9EYGC"},
        {"peerAddress": "/dns4/relayer-1.relayer-1/tcp/9000/p2p/QmcLn2tXGcYA1FUUWsRQoRGmWN17SncGuvjFL3h9azMRgB"},
        {"peerAddress": "/dns4/relayer-2.relayer-2/tcp/9000/p2p/QmVF5HpD7oPkRGFF62pJC6w2QQgD5fZ6qVAzupamugjsTC"},
        {"peerAddress": "/dns4/relayer-3.relayer-3/tcp/9000/p2p/QmZG9c35vUBehEDTkG1mLhw2J4jHG3VsYcJAuY1kqevohE"},
        {"peerAddress": "/dns4/relayer-4.relayer-4/tcp/9000/p2p/QmaFmSv7PkmCo5n4bDLRC8cvDkxDdnbw2sz9ZFNG3EaxHE"}
    ], "threshold": 3}`

	rawTopology := &RawTopology{}
	err := json.Unmarshal([]byte(fixedData), rawTopology)
	if err != nil {
		log.Err(err).Msg("unable to unmarshal topology data")
		return NetworkTopology{}, err
	}

	return ProcessRawTopology(rawTopology)
}
