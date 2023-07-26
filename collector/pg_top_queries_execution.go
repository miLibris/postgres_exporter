package collector

import (
	"context"
	"database/sql"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

const statTopQueriesExecutionTime = "stat_top_queries"

func init() {
	registerCollector(statTopQueriesExecutionTime, defaultEnabled, NewPGStatTopQueriesExecutionTime)
}

type PGStatTopQueriesExecutionTime struct {
	log log.Logger
}

func NewPGStatTopQueriesExecutionTime(config collectorConfig) (Collector, error) {
	return &PGStatTopQueriesExecutionTime{log: config.logger}, nil
}

var (
	statTotalExecutionTime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "total_seconds"),
		"Total time spent in the statement, in milliseconds",
		[]string{"queryid", "query", "user"},
		prometheus.Labels{},
	)
	statMinimumExecutionTime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "min_time_seconds"),
		"Minimum time spent in the statement, in milliseconds",
		[]string{"queryid", "query", "user"},
		prometheus.Labels{},
	)

	statMaximumExecutionTime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "max_time_seconds"),
		"Maximum time spent in the statement, in milliseconds",
		[]string{"queryid", "query", "user"},
		prometheus.Labels{},
	)

	statMeanExecutionTime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "mean_time"),
		"Mean time spent in the statement, in milliseconds",
		[]string{"queryid", "query", "user"},
		prometheus.Labels{},
	)

	statExecutedCalls = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "calls"),
		"Number of times executed",
		[]string{"queryid"},
		prometheus.Labels{},
	)

	statTopQueryExecutionQuery = `SELECT
			queryid,
			query
			pg_get_userbyid(userid) as user,
			mean_time / 1000.0 as mean_time,
			total_time / 1000.0 as total_seconds,
			min_time / 1000.0 as min_time_seconds,
			max_time / 1000.0 as max_time_seconds,
			calls
		FROM pg_stat_statements
		ORDER BY total_seconds DESC
		LIMIT 10;`
)

func (PGStatTopQueriesExecutionTime) Update(ctx context.Context, instance *instance, ch chan<- prometheus.Metric) error {
	db := instance.getDB()
	rows, err := db.QueryContext(ctx, statTopQueryExecutionQuery)

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var queryid, query, user sql.NullString
		var meanTime, totalSeconds, minTimeSeconds, maxTimeSeconds sql.NullFloat64
		var calls sql.NullInt64

		if err := rows.Scan(&queryid, &query, &user, &meanTime, &totalSeconds, &minTimeSeconds, &maxTimeSeconds, &calls); err != nil {
			return err
		}

		queryIdLabel := "unknown"
		if queryid.Valid {
			queryIdLabel = queryid.String
		}

		queryLabel := "unknown"
		if query.Valid {
			queryIdLabel = query.String
		}

		userLabel := "unknown"
		if user.Valid {
			userLabel = user.String
		}

		totalSecondsMetric := 0.0
		if totalSeconds.Valid {
			totalSecondsMetric = totalSeconds.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statTotalExecutionTime,
			prometheus.CounterValue,
			totalSecondsMetric,
			queryIdLabel,
			queryLabel,
			userLabel,
		)

		meanTimeSecondsMetric := 0.0
		if meanTime.Valid {
			meanTimeSecondsMetric = meanTime.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statMeanExecutionTime,
			prometheus.CounterValue,
			meanTimeSecondsMetric,
			queryIdLabel,
			queryLabel,
			userLabel,
		)

		minTimeSecondsMetric := 0.0
		if minTimeSeconds.Valid {
			minTimeSecondsMetric = minTimeSeconds.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statMinimumExecutionTime,
			prometheus.CounterValue,
			minTimeSecondsMetric,
			queryIdLabel,
			queryLabel,
			userLabel,
		)

		maxTimeSecondsMetric := 0.0
		if maxTimeSeconds.Valid {
			maxTimeSecondsMetric = maxTimeSeconds.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statMaximumExecutionTime,
			prometheus.CounterValue,
			maxTimeSecondsMetric,
			queryIdLabel,
			queryLabel,
			userLabel,
		)

		callsMetric := 0.0
		if calls.Valid {
			callsMetric = float64(calls.Int64)
		}

		ch <- prometheus.MustNewConstMetric(
			statExecutedCalls,
			prometheus.CounterValue,
			callsMetric,
			queryIdLabel,
			queryLabel,
			userLabel,
		)
	}
	return nil
}
