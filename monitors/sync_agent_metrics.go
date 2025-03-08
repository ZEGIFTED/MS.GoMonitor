package monitors

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	_ "github.com/microsoft/go-mssqldb"
	"log"
	"net/http"
	"strings"
)

func SyncMetrics(db *sql.DB, agentMetrics []AgentInfo, agentSyncURL string) error {
	// Begin Sync Transaction
	tx, err := db.Begin()

	if err != nil {
		log.Printf("Error starting SyncMetrics transaction: %v", err)

		return err
	}

	defer func(tx *sql.Tx) {
		if err != nil {
			tx.Rollback()
			//log.Printf("Error rolling back SyncMetrics transaction: %v", err)
		} else {
			tx.Commit()
			//if err != nil {
			//log.Printf("Error Committing SyncMetrics transaction: %v", err)
			//return
			//}
		}
	}(tx)

	// Upsert Agents in batch
	agentQuery := `
		MERGE INTO Agents AS target
		USING (VALUES %s) AS source (AgentID, AgentHostName, AgentHostAddress, OS, AgentVersion, SDKVersion)
		ON target.AgentID = source.AgentID
		WHEN MATCHED AND (target.AgentHostName <> source.AgentHostName OR target.AgentHostAddress <> source.AgentHostAddress OR target.OS <> source.OS OR 
						  target.AgentVersion <> source.AgentVersion OR target.SDKVersion <> source.SDKVersion) THEN
			UPDATE SET AgentHostName = source.AgentVersion, AgentHostAddress = source.AgentHostAddress, OS = source.OS, 
						AgentVersion = source.AgentVersion, SDKVersion = source.SDKVersion, LastSync = GETDATE()
		WHEN NOT MATCHED THEN 
			INSERT (AgentID, AgentHostName, AgentHostAddress, OS, AgentVersion, SDKVersion, LastSync) 
			VALUES (source.AgentID, source.AgentHostName, source.AgentHostAddress, source.OS, source.AgentVersion, source.SDKVersion, GETDATE());
	`

	var agentValues []string
	for _, agent := range agentMetrics {
		agentValues = append(agentValues, fmt.Sprintf("('%s', '%s', '%s', '%s', '%s', '%s')",
			agent.AgentID, agent.Name, agent.IPAddress, agent.OS, agent.Version, agent.SDKVersion))
	}

	if len(agentValues) > 0 {
		_, err = tx.Exec(fmt.Sprintf(agentQuery, strings.Join(agentValues, ",")))
		if err != nil {
			return fmt.Errorf("error upserting agents: %v", err)
		}
	}

	// Batch Insert SystemMetrics
	var metricValues []string
	for _, agent := range agentMetrics {
		for _, metric := range agent.Metrics {
			metricValues = append(metricValues, fmt.Sprintf("('%s', %d, %d, %f, %f)",
				agent.AgentID, metric.Timestamp, metric.TimestampMem, metric.CPUUsage, metric.MemoryUsage))
		}
	}

	if len(metricValues) > 0 {
		metricQuery := fmt.Sprintf(`
			INSERT INTO SystemMetricData (AgentID, Timestamp, TimestampMem, CPUUsage, MemoryUsage)
			VALUES %s;`, strings.Join(metricValues, ","))

		_, err = tx.Exec(metricQuery)
		if err != nil {
			return fmt.Errorf("error inserting system metrics: %v", err)
		}
	}

	// Batch Insert Disks
	var diskValues []string
	for _, agent := range agentMetrics {
		for _, disk := range agent.Disks {
			diskValues = append(diskValues, fmt.Sprintf("('%s', '%s', %d, %d, %d, '%s', '%s')",
				agent.AgentID, disk.Drive, disk.Size, disk.Free, disk.Used, disk.FormatFree, disk.FormatSize))
		}
	}

	if len(diskValues) > 0 {
		diskQuery := `
            MERGE INTO SystemDiskData AS target
            USING (VALUES %s) AS source (AgentID, Drive, Size, Free, Used, FormatSize, FormatFree)
            ON target.AgentID = source.AgentID AND target.Drive = source.Drive
            WHEN MATCHED THEN
                UPDATE SET Size = source.Size, Free = source.Free, Used = source.Used, FormatSize = source.FormatSize, FormatFree = source.FormatFree
            WHEN NOT MATCHED THEN
                INSERT (AgentID, Drive, Size, Free, Used, FormatSize, FormatFree)
                VALUES (source.AgentID, source.Drive, source.Size, source.Free, source.Used, source.FormatSize, source.FormatFree);
	`

		_, err = tx.Exec(fmt.Sprintf(diskQuery, strings.Join(diskValues, ", ")))
		if err != nil {
			return fmt.Errorf("error syncing disk metrics: %v", err)
		}
	}

	// Create a custom HTTP client with disabled SSL verification
	client := &http.Client{
		Timeout: constants.HTTPRequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	log.Println("Notifying Agent of Sync Completion", agentSyncURL)
	_, err_ := client.Get(agentSyncURL)
	if err_ != nil {
		return err
	}

	log.Println("Completed DB Data Sync...")
	return nil
}
