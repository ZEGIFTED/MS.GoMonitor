package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

type Metric struct {
	AgentHostAddress string
	AgentHostName    string

	CpuUsage               float64
	MemoryUsage            float64
	CurrentDiskUtilization float64
	TotalStorageCapacity   string
	AgentAPI               string
}

func AgentDataSync() string {
	return "AgentDataSync"
}

func FetchMetricsReport(db *sql.DB) ([]Metric, []string, error) {
	rows, err := db.QueryContext(context.Background(), "EXECUTE ResourceUtilizationSP @StartDate = '', @EndDate = ''")
	if err != nil {
		log.Fatalf("Query failed: %s", err.Error())
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatalf("DB Closure failed: %s", err.Error())
		}
	}(rows)

	var metrics []Metric
	var headers []string
	var tableHeaders []string

	for rows.Next() {
		var resource Metric

		err := rows.Scan(
			&resource.AgentHostName,
			&resource.AgentHostAddress,
			&resource.CpuUsage,
			&resource.MemoryUsage,
			&resource.CurrentDiskUtilization,
			&resource.TotalStorageCapacity,
			&resource.AgentAPI,
		)

		cols, err_ := rows.Columns()
		if err_ != nil {
			log.Fatalf("Error scanning cols: %s", err)
		}

		headers = append(headers, cols...)

		for _, h := range headers {
			if h != "AgentAPIBaseURL" && h != "AgentAPI" {
				tableHeaders = append(tableHeaders, h)
			}
		}

		if err != nil {
			return nil, nil, fmt.Errorf("error scanning resource row: %v", err)
		}

		metrics = append(metrics, resource)
	}

	return metrics, tableHeaders, nil
}
