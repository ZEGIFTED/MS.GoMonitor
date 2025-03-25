package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

func SyncNetworkMetrics(db *sql.DB, Metrics []mstypes.NetworkDeviceMetric) error {
	// Begin Sync Transaction
	tx, err := db.Begin()

	if err != nil {
		log.Printf("Error starting SyncMetrics transaction: %v", err)
		return err
	}

	defer func(tx *sql.Tx) {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}(tx)

	netSyncQuery := `
		MERGE INTO NetworkMetricData AS target
		USING (VALUES %s) AS source (DeviceName, Uptime, Interfaces, Description, CPUUsage, MemoryUsage, InboundTraffic, OutboundTraffic, LastPoll, AgentID, AgentHostName, AgentHostAddress, OS, AgentVersion, SDKVersion)
		ON target.AgentID = source.AgentID
		WHEN MATCHED THEN
			UPDATE SET 
				DeviceName = source.DeviceName,
				Uptime = source.Uptime,
				Interfaces = source.Interfaces,
				Description = source.Description,
				CPUUsage = source.CPUUsage,
				MemoryUsage = source.MemoryUsage,
				InboundTraffic = source.InboundTraffic,
				OutboundTraffic = source.OutboundTraffic,
				LastPoll = GETDATE()
		WHEN NOT MATCHED THEN 
			INSERT (DeviceName, Uptime, Interfaces, Description, CPUUsage, MemoryUsage, InboundTraffic, OutboundTraffic, LastPoll) 
			VALUES (source.DeviceName, source.Uptime, source.Interfaces, source.Description, source.CPUUsage, source.MemoryUsage, source.InboundTraffic, source.OutboundTraffic, source.LastPoll);
	`

	// Prepare the metric values for the query
	var metricValues []string
	for _, metric := range Metrics {
		metricValues = append(metricValues, fmt.Sprintf("('%s', '%s', '%s', '%s', %f, %f, %f, %f)",
			metric.DeviceName,
			metric.Uptime,
			metric.Interfaces,
			metric.Description,
			metric.CPUUsage,
			metric.MemoryUsage,
			metric.InboundTraffic,
			metric.OutboundTraffic,
		))
	}

	if len(metricValues) > 0 {
		_, err = tx.Exec(fmt.Sprintf(netSyncQuery, strings.Join(metricValues, ",")))
		if err != nil {
			return fmt.Errorf("error upserting network device metrics: %v", err)
		}
	}

	return nil
}
