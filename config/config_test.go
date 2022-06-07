package config_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/chainbridge-core/config"
	"github.com/ChainSafe/chainbridge-core/config/relayer"
	"github.com/stretchr/testify/suite"
)

type GetConfigTestSuite struct {
	suite.Suite
}

func TestRunGetConfigTestSuite(t *testing.T) {
	suite.Run(t, new(GetConfigTestSuite))
}

func (s *GetConfigTestSuite) SetupSuite()    {}
func (s *GetConfigTestSuite) TearDownSuite() {}
func (s *GetConfigTestSuite) SetupTest()     {}
func (s *GetConfigTestSuite) TearDownTest()  {}

func (s *GetConfigTestSuite) Test_InvalidPath() {
	_, err := config.GetConfig("invalid")

	s.NotNil(err)
}

type ConfigTestCase struct {
	name       string
	inConfig   config.RawConfig
	shouldFail bool
	errorMsg   string
	outConfig  config.Config
}

func (s *GetConfigTestSuite) TestConfigurationProcessing() {
	testCases := []ConfigTestCase{
		{
			name: "missing chain type",
			inConfig: config.RawConfig{
				ChainConfigs: []map[string]interface{}{{
					"name": "chain1",
				}},
				RelayerConfig: relayer.RawRelayerConfig{
					OpenTelemetryCollectorURL: "",
					LogLevel:                  "",
					LogFile:                   "",
					MpcConfig: relayer.RawMpcRelayerConfig{
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey: "access-key",
							SecKey:    "sec-key",
						},
					},
					BullyConfig: relayer.RawBullyConfig{
						PingWaitTime:     "1s",
						PingBackOff:      "1s",
						PingInterval:     "1s",
						ElectionWaitTime: "1s",
					},
				},
			},
			shouldFail: true,
			errorMsg:   "chain 'type' must be provided for every configured chain",
			outConfig:  config.Config{},
		},
		{
			name: "invalid relayer type",
			inConfig: config.RawConfig{
				RelayerConfig: relayer.RawRelayerConfig{
					LogLevel: "invalid",
					MpcConfig: relayer.RawMpcRelayerConfig{
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey: "access-key",
							SecKey:    "sec-key",
						},
					},
				},
				ChainConfigs: []map[string]interface{}{{
					"name": "chain1",
				}},
			},
			shouldFail: true,
			errorMsg:   "unknown log level: invalid",
			outConfig:  config.Config{},
		},
		{
			name: "invalid bully config",
			inConfig: config.RawConfig{
				RelayerConfig: relayer.RawRelayerConfig{
					LogLevel: "info",
					MpcConfig: relayer.RawMpcRelayerConfig{
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey: "access-key",
							SecKey:    "sec-key",
						},
						Port: 2020,
					},
					BullyConfig: relayer.RawBullyConfig{
						PingWaitTime:     "2z",
						PingBackOff:      "",
						PingInterval:     "",
						ElectionWaitTime: "",
					},
				},
				ChainConfigs: []map[string]interface{}{{
					"name": "chain1",
				}},
			},
			shouldFail: true,
			errorMsg:   "unable to parse bully ping wait time: time: unknown unit \"z\" in duration \"2z\"",
			outConfig:  config.Config{},
		},
		{
			name: "invalid topology config",
			inConfig: config.RawConfig{
				RelayerConfig: relayer.RawRelayerConfig{
					LogLevel: "info",
					MpcConfig: relayer.RawMpcRelayerConfig{
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey: "access-key",
							SecKey:    "",
						},
						Port: 2020,
					},
				},
				ChainConfigs: []map[string]interface{}{{
					"name": "chain1",
				}},
			},
			shouldFail: true,
			errorMsg:   "topology configuration secret key not provided",
			outConfig:  config.Config{},
		},
		{
			name: "set default values in config",
			inConfig: config.RawConfig{
				RelayerConfig: relayer.RawRelayerConfig{
					// LogLevel: use default value,
					// LogFile: use default value
					MpcConfig: relayer.RawMpcRelayerConfig{
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey: "access-key",
							SecKey:    "sec-key",
						},
						// Port: use default value,
					},
				},
				ChainConfigs: []map[string]interface{}{{
					"type": "evm",
					"name": "evm1",
				}},
			},
			shouldFail: false,
			errorMsg:   "unable to parse bully ping wait time: time: unknown unit \"z\" in duration \"2z\"",
			outConfig: config.Config{
				RelayerConfig: relayer.RelayerConfig{
					LogLevel:                  1,
					LogFile:                   "out.log",
					OpenTelemetryCollectorURL: "",
					MpcConfig: relayer.MpcRelayerConfig{
						Port: 9000,
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey:      "access-key",
							SecKey:         "sec-key",
							DocumentName:   "topology.json",
							BucketRegion:   "us-east-1",
							BucketName:     "mpc-topology",
							ServiceAddress: "buckets.chainsafe.io",
						},
					},
					BullyConfig: relayer.BullyConfig{
						PingWaitTime:     1 * time.Second,
						PingBackOff:      1 * time.Second,
						PingInterval:     1 * time.Second,
						ElectionWaitTime: 2 * time.Second,
						BullyWaitTime:    25 * time.Second,
					},
				},
				ChainConfigs: []map[string]interface{}{{
					"type": "evm",
					"name": "evm1",
				}},
			},
		},
		{
			name: "valid config",
			inConfig: config.RawConfig{
				RelayerConfig: relayer.RawRelayerConfig{
					LogLevel: "debug",
					LogFile:  "custom.log",
					MpcConfig: relayer.RawMpcRelayerConfig{
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey:  "access-key",
							SecKey:     "sec-key",
							BucketName: "test-mpc-bucket",
						},
						Port:         2020,
						KeysharePath: "./share.key",
						KeystorePath: "./key.pk",
						Threshold:    5,
					},
					BullyConfig: relayer.RawBullyConfig{
						PingWaitTime:     "1s",
						PingBackOff:      "1s",
						PingInterval:     "1s",
						ElectionWaitTime: "1s",
						BullyWaitTime:    "1s",
					},
				},
				ChainConfigs: []map[string]interface{}{{
					"type": "evm",
					"name": "evm1",
				}},
			},
			shouldFail: false,
			errorMsg:   "unable to parse bully ping wait time: time: unknown unit \"z\" in duration \"2z\"",
			outConfig: config.Config{
				RelayerConfig: relayer.RelayerConfig{
					LogLevel:                  0,
					LogFile:                   "custom.log",
					OpenTelemetryCollectorURL: "",
					MpcConfig: relayer.MpcRelayerConfig{
						Port:         2020,
						KeysharePath: "./share.key",
						KeystorePath: "./key.pk",
						Threshold:    5,
						TopologyConfiguration: relayer.TopologyConfiguration{
							AccessKey:      "access-key",
							SecKey:         "sec-key",
							DocumentName:   "topology.json",
							BucketRegion:   "us-east-1",
							BucketName:     "test-mpc-bucket",
							ServiceAddress: "buckets.chainsafe.io",
						},
					},
					BullyConfig: relayer.BullyConfig{
						PingWaitTime:     time.Second,
						PingBackOff:      time.Second,
						PingInterval:     time.Second,
						ElectionWaitTime: time.Second,
						BullyWaitTime:    time.Second,
					},
				},
				ChainConfigs: []map[string]interface{}{{
					"type": "evm",
					"name": "evm1",
				}},
			},
		},
	}

	for _, t := range testCases {
		s.Run(t.name, func() {
			file, _ := json.Marshal(t.inConfig)
			_ = ioutil.WriteFile("test.json", file, 0644)

			conf, err := config.GetConfig("test.json")

			_ = os.Remove("test.json")

			if t.shouldFail {
				s.NotNil(err)
				s.Equal(t.errorMsg, err.Error())
			} else {
				s.Nil(err)
				s.Equal(t.outConfig, conf)
			}
		})
	}
}
