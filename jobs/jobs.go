package jobs

import (
	"errors"
	"github.com/ChainSafe/sygma-relayer/comm"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/config/relayer"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
)

var defaultJobs = map[string]string{
	"communication-health": "*/5 * * * *", // every 5 min
}

func CreateCronJobs(jobs []relayer.CronJob, h host.Host) *cron.Cron {
	c := cron.New()
	for id, frequency := range SetDefaultJobs(jobs) {
		var err error
		switch id {
		case "communication-health":
			{
				healthComm := p2p.NewCommunication(h, "p2p/health")
				err = c.AddFunc(frequency, func() {
					communicationErrors := comm.ExecuteCommHealthCheck(healthComm, h.Peerstore().Peers())
					for _, cerr := range communicationErrors {
						log.Err(cerr).Msg("communication error")
					}
				})
			}
		default:
			err = errors.New("unknown job configured")
		}

		if err == nil {
			log.Info().Msgf("successfully started job: %s", id)
		} else {
			log.Err(err).Msgf("unable to start job: %s", id)
		}
	}
	return c
}

func SetDefaultJobs(configJobs []relayer.CronJob) map[string]string {
	jobs := map[string]string{}
	for id, defaultFrequency := range defaultJobs {
		if defaultFrequency == "" {
			continue
		}

		f := defaultFrequency

		for _, job := range configJobs {
			if id == job.Id {
				if job.Frequency == "" {
					continue // use default value if empty
				}
				f = job.Frequency // use value from config
			}
		}
		jobs[id] = f
	}
	return jobs
}
