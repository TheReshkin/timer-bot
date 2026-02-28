package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ДОработать, пока тут нет полезных метрик
const (
	ComponentName = "timer_bot"
)

var (
	// CommandUsageCounter counts how many times each command is used
	CommandUsageCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "command_usage_total",
			Help: "Total number of times each command is used",
		},
		[]string{"command"},
	)
)
