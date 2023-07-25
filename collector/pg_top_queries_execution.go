package collector

import (
	"context"
	"database/sql"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

const statTopQueriesExecutionTime = "stat_top_queries"

func init() {
	registerCollector(statTopQueriesExecutionTime, true, NewPGStatTopQueriesExecutionTime)
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
		[]string{"queryid", "datname", "user"},
		prometheus.Labels{},
	)
	statMinimumExecutionTime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "min_time_seconds"),
		"Minimum time spent in the statement, in milliseconds",
		[]string{"queryid", "datname", "user"},
		prometheus.Labels{},
	)

	statMaximumExecutionTime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "max_time_seconds"),
		"Maximum time spent in the statement, in milliseconds",
		[]string{"queryid", "datname", "user"},
		prometheus.Labels{},
	)

	statMeanExecutionTime = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, statTopQueriesExecutionTime, "mean_time"),
		"Mean time spent in the statement, in milliseconds",
		[]string{"queryid", "datname", "user"},
		prometheus.Labels{},
	)

	statTopQueryExecutionQuery = `SELECT
		pg_get_userbyid(userid) as user,
		pg_database.datname,
		pg_stat_statements.queryid,
		pg_stat_statements.mean_time / 1000.0 as mean_time,
		pg_stat_statements.total_time / 1000.0 as total_seconds,
		pg_stat_statements.min_time / 1000.0 as min_time_seconds,
		pg_stat_statements.max_time / 1000.0 as max_time_seconds,
		pg_stat_statements.calls
		FROM pg_stat_statements
		JOIN pg_database
			ON pg_database.id = pg_stat_statements.dbid
		ORDER BY total_seconds DESC
		LIMIT 100;`
)

func (PGStatTopQueriesExecutionTime) Update(ctx context.Context, instance *instance, ch chan<- prometheus.Metric) error {
	db := instance.getDB()
	rows, err := db.QueryContext(ctx, statTopQueryExecutionQuery)

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var user, datName, queryid sql.NullString
		var meanTime, totalSeconds, minTimeSeconds, maxTimeSeconds sql.NullFloat64
		var calls sql.NullInt64

		if err := rows.Scan(&user, &datName, &meanTime, &totalSeconds, &minTimeSeconds, &maxTimeSeconds, &calls); err != nil {
			return err
		}

		userLabel := "unknown"
		if user.Valid {
			userLabel = user.String
		}

		datnameLabel := "unknown"
		if datName.Valid {
			datnameLabel = datName.String
		}

		queryIdLabel := "unknown"
		if queryid.Valid {
			queryIdLabel = queryid.String
		}

		totalSecondsMetric := 0.0
		if totalSeconds.Valid {
			totalSecondsMetric = totalSeconds.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statTotalExecutionTime,
			prometheus.CounterValue,
			totalSecondsMetric,
			userLabel, datnameLabel, queryIdLabel,
		)

		meanTimeSecondsMetric := 0.0
		if meanTime.Valid {
			meanTimeSecondsMetric = meanTime.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statMeanExecutionTime,
			prometheus.CounterValue,
			meanTimeSecondsMetric,
			userLabel, datnameLabel, queryIdLabel,
		)

		minTimeSecondsMetric := 0.0
		if minTimeSeconds.Valid {
			minTimeSecondsMetric = minTimeSeconds.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statMinimumExecutionTime,
			prometheus.CounterValue,
			minTimeSecondsMetric,
			userLabel, datnameLabel, queryIdLabel,
		)

		maxTimeSecondsMetric := 0.0
		if maxTimeSeconds.Valid {
			maxTimeSecondsMetric = maxTimeSeconds.Float64
		}
		ch <- prometheus.MustNewConstMetric(
			statMaximumExecutionTime,
			prometheus.CounterValue,
			maxTimeSecondsMetric,
			userLabel, datnameLabel, queryIdLabel,
		)

		callsMetric := 0.0
		if calls.Valid {
			callsMetric = float64(calls.Int64)
		}

		ch <- prometheus.MustNewConstMetric(
			statTotalExecutionTime,
			prometheus.CounterValue,
			callsMetric,
			userLabel, datnameLabel, queryIdLabel,
		)
	}
	return nil
}
