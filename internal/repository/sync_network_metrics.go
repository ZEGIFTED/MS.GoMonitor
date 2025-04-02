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

	// netSyncQuery := `
	// 		INSERT INTO network_metrics (device_name, device_ip, metric_name, metric_oid, metric_value, timestamp)
	// 		VALUES (?, ?, ?, ?, ?, NOW())
	// 		ON DUPLICATE KEY UPDATE metric_value = VALUES(metric_value), timestamp = NOW()
	// 	`

	netSyncQuery := `
		MERGE INTO NetworkDeviceMetricData AS target
		USING (VALUES %s) AS source (SystemMonitorId, DeviceName, MetricName, DeviceIP, MetricDescription, MetricValue, LastPoll)
		ON target.DeviceIP = source.DeviceIP
		WHEN MATCHED THEN
			UPDATE SET 
			SystemMonitorId = source.SystemMonitorId,
			DeviceName = source.DeviceName,
			MetricName = source.MetricName,
			DeviceIP = source.DeviceIP,
			MetricDescription = source.MetricDescription,
			MetricValue = source.MetricValue,
			LastPoll = source.LastPoll
		WHEN NOT MATCHED THEN 
			INSERT (SystemMonitorId, DeviceName, MetricName, DeviceIP, MetricDescription, MetricValue, LastPoll) 
			VALUES (
			source.SystemMonitorId,
			source.DeviceName,
			source.MetricName,
			source.DeviceIP,
			source.MetricDescription,
			source.MetricValue,
			source.LastPoll);
	`

	// Prepare the metric values for the query
	var metricValues []string
	for _, metric := range Metrics {
		metricValues = append(metricValues, fmt.Sprintf("('%s', '%s', '%s', '%s', '%s', '%s', CAST('%s' AS DATETIME))",
			metric.SystemMonitorId,
			metric.DeviceName,
			metric.MetricName,
			metric.DeviceIP,
			metric.MetricDescription,
			metric.MetricValue,
			metric.LastPoll,
		))
	}

	for v, i := range metricValues {
		log.Println(v, i)
	}

	if len(metricValues) > 0 {
		_, err = tx.Exec(fmt.Sprintf(netSyncQuery, strings.Join(metricValues, ",")))
		if err != nil {
			return fmt.Errorf("error upserting network device metrics: %v", err)
		}
	}

	return nil
}
