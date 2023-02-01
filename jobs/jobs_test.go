package jobs_test

import (
	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/ChainSafe/sygma-relayer/jobs"
	"github.com/stretchr/testify/suite"
	"testing"
)

type CronJobsTestSuite struct {
	suite.Suite
}

func TestRunCronJobsTestSuite(t *testing.T) {
	suite.Run(t, new(CronJobsTestSuite))
}

func (s *CronJobsTestSuite) Test_CreateCronJobs_CommunicationHealth() {
	tests := []struct {
		configJobs []relayer.CronJob
		expected   map[string]string
	}{
		{
			configJobs: []relayer.CronJob{
				{Id: "communication-health", Frequency: "*/10 * * * *"},
			},
			expected: map[string]string{
				"communication-health": "*/10 * * * *",
			},
		},
		{
			configJobs: []relayer.CronJob{
				{Id: "communication-health", Frequency: ""},
			},
			expected: map[string]string{
				"communication-health": "*/5 * * * *",
			},
		},
		{
			configJobs: []relayer.CronJob{},
			expected: map[string]string{
				"communication-health": "*/5 * * * *",
			},
		},
	}
	for _, test := range tests {
		result := jobs.SetDefaultJobs(test.configJobs)
		s.Equal(len(test.expected), len(result))
		for id, expected := range test.expected {
			s.Equal(expected, result[id])
		}
	}
}
