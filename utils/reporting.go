package utils

import (
	"database/sql"
	"fmt"
	"github.com/jung-kurt/gofpdf"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Metric struct {
	ServerName             string
	AgentId                string
	IPAddress              string
	CpuUsage               float64
	MemoryUsage            float64
	CurrentDiskUtilization float64
	TotalDiskSpace         float64
	RecordedAt             time.Time
}

type SystemCount struct {
	Status string // Active, Inactive, Scheduled
	Count  int
}

// Colors for the PDF
var (
	headerBg    = []int{0, 32, 96}     // Deep Navy Blue
	titleBg     = []int{0, 51, 153}    // Royal Blue
	tableBg     = []int{240, 244, 248} // Light Blue-Gray
	alertColor  = []int{231, 76, 60}   // Red
	normalColor = []int{46, 204, 113}  // Green
)

func GenerateReport(db *sql.DB) string {
	//Query data
	//query := "\n\t\tSELECT ServerName, CpuUsage, MemoryUsage, CurrentDiskUtilization, RecordedAt \n\t\tFROM ITInfrastructureMetrics \n\t\tWHERE RecordedAt >= DATEADD(HOUR, -1, GETDATE())"
	//
	//rows, err := db.Query(query)
	//if err != nil {
	//	log.Fatalf("Query failed: %s", err.Error())
	//}
	//
	//defer func(rows *sql.Rows) {
	//	err := rows.Close()
	//	if err != nil {
	//		log.Fatalf("DB Closure failed: %s", err.Error())
	//	}
	//}(rows)

	var metrics []Metric

	metrics = append(metrics, Metric{
		ServerName:             "Test Server",
		IPAddress:              "127.0.0.1",
		CpuUsage:               50,
		MemoryUsage:            1024,
		CurrentDiskUtilization: 2048,
		RecordedAt:             time.Now(),
	})

	metrics = append(metrics, Metric{
		ServerName:             "Test Server 2",
		IPAddress:              "127.0.0.2",
		CpuUsage:               40,
		MemoryUsage:            1024,
		CurrentDiskUtilization: 2048,
		RecordedAt:             time.Now(),
	})

	metrics = append(metrics, Metric{
		ServerName:             "Test Server 3",
		IPAddress:              "127.0.0.3",
		CpuUsage:               40,
		MemoryUsage:            1024,
		CurrentDiskUtilization: 2048,
		RecordedAt:             time.Now(),
	})

	metrics = append(metrics, Metric{
		ServerName:             "Test Server 4",
		IPAddress:              "127.0.0.4",
		CpuUsage:               40,
		MemoryUsage:            1024,
		CurrentDiskUtilization: 2048,
		RecordedAt:             time.Now(),
	})

	//var headers []string
	//for rows.Next() {
	//	cols, err_ := rows.Columns()
	//	if err_ != nil {
	//		log.Fatalf("Error scanning cols: %s", err)
	//	}
	//
	//	headers = append(headers, cols...)
	//
	//	var m Metric
	//	if err := rows.Scan(&m.ServerName, &m.CpuUsage, &m.MemoryUsage, &m.CurrentDiskUtilization, &m.RecordedAt); err != nil {
	//		log.Fatalf("Error scanning row: %s", err.Error())
	//	}
	//	metrics = append(metrics, m)
	//}

	tableHeaders := []string{"ServerName", "CpuUsage", "MemoryUsage", "CurrentDiskUtilization"}

	// Generate PDF
	var filePath = GeneratePDF(metrics, tableHeaders)

	return filePath
}

func Header(pdf *gofpdf.Fpdf, reportTime string) {
	// Add logo placeholder
	pdf.SetFillColor(headerBg[0], headerBg[1], headerBg[2])
	pdf.Rect(0, 0, 210, 25, "F")
	logoPath := "public/work2.png"
	orgCompanyPath := "public/nibsslogo.png"

	// Company Logo placeholder
	//imageOptions := gofpdf.ImageOptions{
	//	ImageType: "PNG",
	//	ReadDpi:   true,
	//}
	//
	//// Get the width of the page to position the image
	//pageWidth, _ := pdf.GetPageSize()
	//logoWidth := 30.0 // Adjust the width as needed
	//margin := 10.0    // Margin from the edge

	pdf.Image(orgCompanyPath, 5, 5, 18, 0, false, "", 0, "")
	//pdf.ImageOptions(orgCompanyPath, pageWidth-logoWidth-margin, margin, logoWidth, 0, false, imageOptions, 0, "")

	// Title
	pdf.SetY(-1)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.Text(25, 10, "IT Infrastructure Report")

	pdf.SetFont("Arial", "B", 7)
	pdf.Text(25, 15, fmt.Sprintf("Generated at %s", reportTime))

	// Or Top Right (uncomment this line to switch)
	pdf.Image(logoPath, 170, 5, 35, 0, false, "", 0, "")

}

func ChartSection(pdf *gofpdf.Fpdf, metrics []Metric) {
	pdf.AddPage()

	// Section Title
	pdf.SetFillColor(titleBg[0], titleBg[1], titleBg[2])
	pdf.Rect(10, 50, 190, 10, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 10)
	pdf.Text(15, 57, "System Performance Overview")

	// Draw usage bars
	startY := 70
	for _, metric := range metrics {
		// Server name
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "B", 7)
		pdf.Text(15, float64(startY), metric.ServerName)

		// CPU Usage Bar
		UsageBar(pdf, 60, float64(startY)-3, metric.CpuUsage)
		pdf.SetFont("Arial", "", 8)
		pdf.Text(170, float64(startY), fmt.Sprintf("%.1f%%", metric.CpuUsage))

		startY += 15
	}
}

func UsageBar(pdf *gofpdf.Fpdf, x, y, percentage float64) {
	width := 100.0
	height := 5.0

	// Background bar
	pdf.SetFillColor(tableBg[0], tableBg[1], tableBg[2])
	pdf.Rect(x, y, width, height, "F")

	// Usage bar
	if percentage > 80 {
		pdf.SetFillColor(alertColor[0], alertColor[1], alertColor[2])
	} else {
		pdf.SetFillColor(normalColor[0], normalColor[1], normalColor[2])
	}
	pdf.Rect(x, y, width*(percentage/100), height, "F")
}

func MetricsTable(pdf *gofpdf.Fpdf, metrics []Metric, tableHeaders []string) {
	pageWidth, _ := pdf.GetPageSize()
	marginLeft, _, marginRight, _ := pdf.GetMargins()
	tableWidth := pageWidth - marginLeft - marginRight

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.Text(pdf.GetX()+5, pdf.GetY(), "Infrastructure Report")

	// Table headers
	for _, header := range tableHeaders {
		pdf.CellFormat(40, 10, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Calculate column widths
	colWidths := []float64{40, 20, 20, 25, 25, 20}

	// Calculate proportional column widths based on full width
	//colWidths := []float64{
	//	tableWidth * 0.15, // Server name gets more space
	//	tableWidth * 0.15, // Server IP gets more space
	//	tableWidth * 0.12, // CPU
	//	tableWidth * 0.12, // Memory
	//	tableWidth * 0.12, // Memory
	//	tableWidth * 0.12, // Disk
	//	//tableWidth * 0.125, // Network In
	//	//tableWidth * 0.125, // Network Out
	//	//tableWidth * 0.1,  // Users
	//	//tableWidth * 0.1,  // Alerts
	//}

	// Draw table header
	pdf.SetFillColor(headerBg[0], headerBg[1], headerBg[2])
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 10)

	x := marginLeft
	for i, header := range tableHeaders {
		pdf.Rect(x, pdf.GetY(), colWidths[i], 10, "DF")
		pdf.Text(x+2, pdf.GetY()+5, header)
		x += colWidths[i]
	}
	pdf.Ln(10)

	// Add some content below the logo
	//pdf.SetFont("Arial", "", 10)
	//for _, m := range metrics {
	//	pdf.Cell(colWidths[0], 10, m.ServerName)
	//	pdf.Cell(colWidths[1], 10, m.IPAddress)
	//	pdf.Cell(colWidths[2], 10, fmt.Sprintf("%.1f", m.CpuUsage))
	//	pdf.Cell(colWidths[3], 10, fmt.Sprintf("%.1f", m.MemoryUsage))
	//	pdf.Cell(colWidths[4], 10, fmt.Sprintf("%.1f", m.CurrentDiskUtilization))
	//	//pdf.Cell(colWidths[5], 10, fmt.Sprintf("%.2f MB", m.CurrentDiskUtilization))
	//	//pdf.Cell(colWidths[6], 10, fmt.Sprintf("%.2f MB", m.NetworkOut))
	//	//pdf.Cell(colWidths[7], 10, fmt.Sprintf("%d", m.ActiveUsers))
	//	//pdf.Cell(colWidths[8], 10, fmt.Sprintf("%d", m.AlertCount))
	//	pdf.Ln(-1)
	//}

	// Table content
	//pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 7)
	rowColor := false

	for _, m := range metrics {
		x = marginLeft
		if rowColor {
			pdf.SetFillColor(tableBg[0], tableBg[1], tableBg[2])
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		if m.CpuUsage > 80 || m.MemoryUsage > 80 || m.CurrentDiskUtilization > 80 {
			pdf.SetTextColor(alertColor[0], alertColor[1], alertColor[2])
		} else {
			pdf.SetTextColor(0, 32, 96) // Deep blue for normal values
		}

		// Draw row background
		pdf.Rect(x, pdf.GetY(), tableWidth, 8, "F")

		// Helper function to format values
		formatValue := func(value float64) string {
			if value < 0 { // Check for invalid values
				return "N/A"
			}
			return fmt.Sprintf("%.1f", value)
		}

		// Draw cell content
		pdf.Text(x+2, pdf.GetY()+5, m.ServerName)
		x += colWidths[0]
		pdf.Text(x+2, pdf.GetY()+5, m.IPAddress)
		x += colWidths[0]
		pdf.Text(x+2, pdf.GetY()+5, formatValue(m.CpuUsage))
		x += colWidths[1]
		pdf.Text(x+2, pdf.GetY()+5, formatValue(m.MemoryUsage))
		x += colWidths[2]
		pdf.Text(x+2, pdf.GetY()+5, formatValue(m.CurrentDiskUtilization))
		//x += colWidths[3]
		//pdf.Text(x+2, pdf.GetY()+5, fmt.Sprintf("%.2f MB", m.NetworkIn))
		//x += colWidths[4]
		//pdf.Text(x+2, pdf.GetY()+7, fmt.Sprintf("%.2f MB", m.NetworkOut))
		//x += colWidths[5]
		//pdf.Text(x+2, pdf.GetY()+7, fmt.Sprintf("%d", m.ActiveUsers))
		//x += colWidths[6]
		//pdf.Text(x+2, pdf.GetY()+7, fmt.Sprintf("%d", m.AlertCount))

		pdf.Ln(10)
		rowColor = !rowColor
	}
}

// GeneratePDF metrics ServiceMonitorStatus
func GeneratePDF(metrics []Metric, headers []string) string {
	// Get current time and format it
	currentTime := time.Now()
	formattedTime := currentTime.Format("January 2, 2006 15:04:05")
	//.Format("2006-01-02 15:04:05")

	// Initialize a new PDF document (A4 size, portrait orientation)
	pdf := gofpdf.New("P", "mm", "A4", "")

	// First page - Overview
	pdf.AddPage()
	Header(pdf, formattedTime)

	pdf.SetY(30)
	pdf.SetFont("Arial", "", 21)
	// Active / Inactive System / Scheduled for maintainance

	//sections := map[string]string{
	//	"Active":    "Active Systems",
	//	"Inactive":  "Inactive Systems",
	//	"Scheduled": "Scheduled for Maintenance",
	//}

	//var counts []SystemCount
	counts := []SystemCount{
		{"Active", 50000},
		{"Inactive", 2000},
		{"Scheduled", 302},
		{"Acknowledged", 31},
	}

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(titleBg[0], titleBg[1], titleBg[2])
	pdf.Cell(190, 8, "System Status Summary")
	pdf.Ln(10)

	boxWidth := 45.0
	boxHeight := 16.0
	margin := 5.0
	startX := pdf.GetX()
	startY := pdf.GetY()

	for i, count := range counts {
		if i > 0 {
			pdf.SetX(startX + float64(i)*(boxWidth+margin))
		}

		// Determine colors
		var r, g, b int
		if count.Status == "Active" {
			r, g, b = 0, 102, 204 // Blue for active
		} else if count.Status == "Inactive" {
			r, g, b = 204, 0, 0 // Red for inactive
		} else if count.Status == "Scheduled" {
			r, g, b = 255, 165, 0 // Orange for scheduled
		} else if count.Status == "Acknowledged" {
			r, g, b = 102, 106, 109
		}

		// Draw rounded rectangle with border color matching count color
		//pdf.SetDrawColor(r, g, b)
		pdf.SetLineWidth(0.3)
		pdf.RoundedRect(pdf.GetX(), startY, boxWidth, boxHeight, 3, "1234", "D")

		// Status in black and capitalized
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "B", 9)
		pdf.SetXY(pdf.GetX(), startY+5)
		pdf.CellFormat(boxWidth, boxHeight/2, strings.ToUpper(count.Status), "0", 0, "C", false, 0, "")

		// Count with respective color and bigger font, centered
		pdf.SetTextColor(r, g, b)
		pdf.SetFont("Arial", "B", 10)
		pdf.SetXY(pdf.GetX()-32, startY+9)
		pdf.CellFormat(boxWidth, boxHeight/2, fmt.Sprintf("%d", count.Count), "0", 0, "C", false, 0, "")
	}

	//pdf.AddPage()
	// Table section
	pdf.SetY(70)
	MetricsTable(pdf, metrics, headers)

	// Chart section on new page
	pdf.SetY(30)
	ChartSection(pdf, metrics)

	// Add footer
	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(0, 10, fmt.Sprintf("All Rights Reserved. Company Name. Page %d", pdf.PageNo()), "0", 0, "C", false, 0, "")
	})

	// Sanitize filename to avoid issues with colons or other special characters
	//safeFilename := strings.ReplaceAll(formattedTime, ":", "-")

	// Construct the filename and path
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, os.ModePerm); err != nil {
		log.Fatalf("Error creating reports directory: %s", err.Error())
	}

	fileName := "Hourly_IT_Report.pdf"
	//fileName := "Hourly_IT_Report_" + formattedTime + ".pdf"

	filePath := filepath.Join(reportsDir, fileName)
	log.Printf("Writing report to %s", filePath)

	// Output the PDF to a file
	err := pdf.OutputFileAndClose(filePath)
	//if err != nil {
	//	log.Fatalf("Error creating PDF: %s", err)
	//}
	if err != nil {
		panic(err)
	}

	log.Println("PDF created successfully!")

	return filePath
}
