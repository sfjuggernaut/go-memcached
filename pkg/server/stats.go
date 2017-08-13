package server

import (
	"expvar"
	"time"
)

var (
	StatsNumCas    = expvar.NewInt("num_cas")
	StatsNumDelete = expvar.NewInt("num_delete")
	StatsNumGet    = expvar.NewInt("num_get")
	StatsNumGets   = expvar.NewInt("num_gets")
	StatsNumSet    = expvar.NewInt("num_set")

	StatsErrNumUnsupportedCmds = expvar.NewInt("err_num_unsupported_cmds")
)

// uptime returns time.Duration since server started
func (s *Server) uptime() time.Duration {
	return time.Since(s.startTime)
}

func (s *Server) getStats() map[string]string {
	stats := make(map[string]string)
	expvar.Do(func(variable expvar.KeyValue) {
		if variable.Key != "cmdline" && variable.Key != "memstats" {
			stats[variable.Key] = variable.Value.String()
		}
	})

	stats["start_time"] = s.startTime.String()
	stats["uptime"] = s.uptime().String()

	return stats
}
