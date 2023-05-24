package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestOk = promauto.NewCounter(prometheus.CounterOpts{
		Name: "request_ok_total",
		Help: "total requests processed successfully",
	})
	RequestFail = promauto.NewCounter(prometheus.CounterOpts{
		Name: "request_fail_total",
		Help: "total requests failed",
	})
	RequestLimit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "request_limit_total",
		Help: "total requests limited",
	})
)
