package config

import (
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type LoadFromEnvTestSuite struct {
	suite.Suite
}

func TestRunLoadFromEnvTestSuite(t *testing.T) {
	suite.Run(t, new(LoadFromEnvTestSuite))
}

func (s *LoadFromEnvTestSuite) SetupSuite()    {}
func (s *LoadFromEnvTestSuite) TearDownSuite() {}
func (s *LoadFromEnvTestSuite) SetupTest() {
	os.Clearenv()
}
func (s *LoadFromEnvTestSuite) TearDownTest() {}

func (s *LoadFromEnvTestSuite) Test_ValidRelayerConfig() {
	_ = os.Setenv("CBH_RELAYER_OPENTELEMETRYCOLLECTORURL", "test.opentelemetry.url")
	_ = os.Setenv("CBH_RELAYER_LOGLEVEL", "info")
	_ = os.Setenv("CBH_RELAYER_LOGFILE", "test.log")
	_ = os.Setenv("CBH_RELAYER_HEALTHPORT", "4000")

	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_KEY", "test-pk")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_KEYSHAREPATH", "/cfg/keyshares/0.keyshare")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_THRESHOLD", "3")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_PORT", "9000")

	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_TOPOLOGYCONFIGURATION_ACCESSKEY", "test-access-key")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_TOPOLOGYCONFIGURATION_SECKEY", "test-sec-key")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_TOPOLOGYCONFIGURATION_DOCUMENTNAME", "test-doc-name")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_TOPOLOGYCONFIGURATION_BUCKETREGION", "test-bucket-region")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_TOPOLOGYCONFIGURATION_BUCKETNAME", "test-bucket-name")
	_ = os.Setenv("CBH_RELAYER_MPCCONFIG_TOPOLOGYCONFIGURATION_SERVICEADDRESS", "test-service-address")

	_ = os.Setenv("CBH_RELAYER_BULLYCONFIG_PINGWAITTIME", "2s")
	_ = os.Setenv("CBH_RELAYER_BULLYCONFIG_PINGBACKOFF", "2s")
	_ = os.Setenv("CBH_RELAYER_BULLYCONFIG_PINGINTERVAL", "2s")
	_ = os.Setenv("CBH_RELAYER_BULLYCONFIG_ELECTIONWAITTIME", "2s")
	_ = os.Setenv("CBH_RELAYER_BULLYCONFIG_BULLYWAITTIME", "2s")

	env, err := loadFromEnv()

	s.Nil(err)
	s.Equal(relayer.RawRelayerConfig{
		OpenTelemetryCollectorURL: "test.opentelemetry.url",
		LogLevel:                  "info",
		LogFile:                   "test.log",
		HealthPort:                "4000",
		MpcConfig: relayer.RawMpcRelayerConfig{
			KeysharePath: "/cfg/keyshares/0.keyshare",
			Key:          "test-pk",
			Threshold:    "3",
			Port:         "9000",
			TopologyConfiguration: relayer.TopologyConfiguration{
				AccessKey:      "test-access-key",
				SecKey:         "test-sec-key",
				DocumentName:   "test-doc-name",
				BucketRegion:   "test-bucket-region",
				BucketName:     "test-bucket-name",
				ServiceAddress: "test-service-address",
			},
		},
		BullyConfig: relayer.RawBullyConfig{
			PingWaitTime:     "2s",
			PingBackOff:      "2s",
			PingInterval:     "2s",
			ElectionWaitTime: "2s",
			BullyWaitTime:    "2s",
		},
	}, env.RelayerConfig)
}

func (s *LoadFromEnvTestSuite) Test_ValidChainConfig() {
	_ = os.Setenv("CBH_DOM_1", "{\n      \"id\": 1,\n      \"from\": \"0xff93B45308FD417dF303D6515aB04D9e89a750Ca\",\n      \"name\": \"evm1\",\n      \"type\": \"evm\",\n      \"endpoint\": \"ws://evm1-1:8546\",\n      \"bridge\": \"0xd606A00c1A39dA53EA7Bb3Ab570BBE40b156EB66\",\n      \"erc20Handler\": \"0x3cA3808176Ad060Ad80c4e08F30d85973Ef1d99e\",\n      \"erc721Handler\": \"0x75dF75bcdCa8eA2360c562b4aaDBAF3dfAf5b19b\",\n      \"genericHandler\": \"0xe1588E2c6a002AE93AeD325A910Ed30961874109\",\n      \"gasLimit\": 9000000,\n      \"maxGasPrice\": 20000000000,\n      \"blockConfirmations\": 2\n    }")
	_ = os.Setenv("CBH_DOM_2", "{\n      \"id\": 2,\n      \"from\": \"0xff93B45308FD417dF303D6515aB04D9e89a750Ca\",\n      \"name\": \"evm2\",\n      \"type\": \"evm\",\n      \"endpoint\": \"ws://evm2-1:8546\",\n      \"bridge\": \"0xd606A00c1A39dA53EA7Bb3Ab570BBE40b156EB66\",\n      \"erc20Handler\": \"0x3cA3808176Ad060Ad80c4e08F30d85973Ef1d99e\",\n      \"erc721Handler\": \"0x75dF75bcdCa8eA2360c562b4aaDBAF3dfAf5b19b\",\n      \"genericHandler\": \"0xe1588E2c6a002AE93AeD325A910Ed30961874109\",\n      \"gasLimit\": 9000000,\n      \"maxGasPrice\": 20000000000,\n      \"blockConfirmations\": 2\n    }")

	env, err := loadFromEnv()

	s.Nil(err)
	s.Equal([]map[string]interface{}{
		{
			"id":                 float64(1),
			"type":               "evm",
			"bridge":             "0xd606A00c1A39dA53EA7Bb3Ab570BBE40b156EB66",
			"erc721Handler":      "0x75dF75bcdCa8eA2360c562b4aaDBAF3dfAf5b19b",
			"gasLimit":           9e+06,
			"maxGasPrice":        2e+10,
			"from":               "0xff93B45308FD417dF303D6515aB04D9e89a750Ca",
			"name":               "evm1",
			"endpoint":           "ws://evm1-1:8546",
			"erc20Handler":       "0x3cA3808176Ad060Ad80c4e08F30d85973Ef1d99e",
			"genericHandler":     "0xe1588E2c6a002AE93AeD325A910Ed30961874109",
			"blockConfirmations": float64(2),
		},
		{
			"id":                 float64(2),
			"type":               "evm",
			"bridge":             "0xd606A00c1A39dA53EA7Bb3Ab570BBE40b156EB66",
			"erc721Handler":      "0x75dF75bcdCa8eA2360c562b4aaDBAF3dfAf5b19b",
			"gasLimit":           9e+06,
			"maxGasPrice":        2e+10,
			"from":               "0xff93B45308FD417dF303D6515aB04D9e89a750Ca",
			"name":               "evm2",
			"endpoint":           "ws://evm2-1:8546",
			"erc20Handler":       "0x3cA3808176Ad060Ad80c4e08F30d85973Ef1d99e",
			"genericHandler":     "0xe1588E2c6a002AE93AeD325A910Ed30961874109",
			"blockConfirmations": float64(2),
		}}, env.ChainConfigs)
}
