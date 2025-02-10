package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
)

// InfrastructureReportMetrics represents infrastructure metrics from the database
type InfrastructureReportMetrics struct {
	Timestamp   time.Time
	ServerName  string
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
	NetworkIn   float64
	NetworkOut  float64
	ActiveUsers int
	AlertCount  int
}

// Define the database connection string
const (
	server   = "your_server_name"
	user     = "your_username"
	password = "your_password"
	database = "your_database_name"
)

// Define the SQL query to retrieve the infrastructure data
const query = `
        -- Your SQL query to retrieve infrastructure data
        -- Example:
        SELECT 
                ServerName, 
                OSName, 
                OSVersion, 
                CPUCount, 
                MemoryTotal 
        FROM 
                Servers 
`

func fetchReportingMetrics(db *sql.DB, start, end time.Time) ([]InfrastructureReportMetrics, error) {
	query := ``

	rows, err := db.QueryContext(context.Background(), query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	defer rows.Close()

	var metrics []InfrastructureReportMetrics
	for rows.Next() {
		var m InfrastructureReportMetrics
		err := rows.Scan(
			&m.Timestamp,
			&m.ServerName,
			&m.CPUUsage,
			&m.MemoryUsage,
			&m.DiskUsage,
			&m.NetworkIn,
			&m.NetworkOut,
			&m.ActiveUsers,
			&m.AlertCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func generatePDFReport(metrics []InfrastructureReportMetrics, reportTime time.Time) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAuthor("MS", true)
	pdf.SetTitle("Hourly IT Infrastructure Report", true)
	pdf.AddPage()

	// Set up header
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "IT Infrastructure Report")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(190, 10, fmt.Sprintf("Generated at: %s", reportTime.Format("2006-01-02 15:04:05")))
	pdf.Ln(15)

	// Table headers
	headers := []string{"Server", "CPU %", "Memory %", "Disk %", "Network In", "Network Out", "Users", "Alerts"}
	pdf.SetFont("Arial", "B", 10)

	// Calculate column widths
	colWidths := []float64{40, 20, 20, 20, 25, 25, 20, 20}

	// Draw headers
	for i, header := range headers {
		pdf.Cell(colWidths[i], 10, header)
	}
	pdf.Ln(-1)

	// Table content
	pdf.SetFont("Arial", "", 10)
	for _, m := range metrics {
		pdf.Cell(colWidths[0], 10, m.ServerName)
		pdf.Cell(colWidths[1], 10, fmt.Sprintf("%.1f", m.CPUUsage))
		pdf.Cell(colWidths[2], 10, fmt.Sprintf("%.1f", m.MemoryUsage))
		pdf.Cell(colWidths[3], 10, fmt.Sprintf("%.1f", m.DiskUsage))
		pdf.Cell(colWidths[4], 10, fmt.Sprintf("%.2f MB", m.NetworkIn))
		pdf.Cell(colWidths[5], 10, fmt.Sprintf("%.2f MB", m.NetworkOut))
		pdf.Cell(colWidths[6], 10, fmt.Sprintf("%d", m.ActiveUsers))
		pdf.Cell(colWidths[7], 10, fmt.Sprintf("%d", m.AlertCount))
		pdf.Ln(-1)
	}

	// Add summary section
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 10, "Summary")
	pdf.Ln(10)

	// Calculate and add summary statistics
	var totalCPU, totalMem, totalDisk float64
	totalAlerts := 0
	for _, m := range metrics {
		totalCPU += m.CPUUsage
		totalMem += m.MemoryUsage
		totalDisk += m.DiskUsage
		totalAlerts += m.AlertCount
	}

	numMetrics := float64(len(metrics))
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(190, 10, fmt.Sprintf("Average CPU Usage: %.1f%%", totalCPU/numMetrics))
	pdf.Ln(8)
	pdf.Cell(190, 10, fmt.Sprintf("Average Memory Usage: %.1f%%", totalMem/numMetrics))
	pdf.Ln(8)
	pdf.Cell(190, 10, fmt.Sprintf("Average Disk Usage: %.1f%%", totalDisk/numMetrics))
	pdf.Ln(8)
	pdf.Cell(190, 10, fmt.Sprintf("Total Alerts: %d", totalAlerts))

	// Save the PDF
	filename := fmt.Sprintf("infra_report_%s.pdf", reportTime.Format("20060102_150405"))

	fmt.Println("Hourly IT Infrastructure Report generated successfully:", filename)

	//currentTime := time.Now().Format("2006-01-02_15-04-05")
	//filename := fmt.Sprintf("infrastructure_report_%s.pdf", currentTime)
	//err = os.WriteFile(filename, buf.Bytes(), 0644)
	//if err != nil {
	//	log.Fatal("Failed to save PDF:", err)
	//}
	return pdf.OutputFileAndClose(filename)
}
