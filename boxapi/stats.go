package boxapi

import (
	"context"

	v2rayapi "github.com/sagernet/sing-box/experimental/v2rayapi"
	"github.com/sagernet/sing-box/option"
)

type StatsService = v2rayapi.StatsService

func NewStatsService(options option.V2RayStatsServiceOptions) *StatsService {
	return v2rayapi.NewStatsService(options)
}

// QueryStats NekoRay style API
func QueryStats(ss *StatsService, name string) int64 {
	if ss == nil {
		return 0
	}
	resp, err := ss.GetStats(context.TODO(), &v2rayapi.GetStatsRequest{Name: name, Reset_: true})
	if err != nil || resp.Stat == nil {
		return 0
	}
	return resp.Stat.Value
}
