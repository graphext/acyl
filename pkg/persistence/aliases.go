package persistence

import (
	"github.com/dollarshaveclub/acyl/pkg/config"
	"github.com/dollarshaveclub/acyl/pkg/metrics"
	"github.com/dollarshaveclub/acyl/pkg/models"
)

type QAType = models.QAType
type QAEnvironment = models.QAEnvironment
type QAEnvironments = models.QAEnvironments
type EnvironmentStatus = models.EnvironmentStatus
type RepoRevisionData = models.RepoRevisionData
type RefMap = models.RefMap
type QADestroyReason = models.QADestroyReason
type QAEnvironmentEvent = models.QAEnvironmentEvent

type ServerConfig = config.ServerConfig
type MetricsCollector = metrics.Collector

type AWSCreds = config.AWSCreds
type AWSConfig = config.AWSConfig
