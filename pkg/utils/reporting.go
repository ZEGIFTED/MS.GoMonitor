package utils

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/internal"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/jung-kurt/gofpdf"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SystemCount struct {
	Status string // Active, Inactive, Scheduled
	Count  int
}

func GenerateReport(db *sql.DB) (string, string) {
	var metrics, tableHeaders, err = internal.FetchMetricsReport(db)

	if err != nil {
		log.Println("Error fetching metrics report", err)
	}

	// Generate PDF
	var filePath = GeneratePDF(metrics, tableHeaders)
	var csvFilePath = GenerateCSV(metrics, tableHeaders)

	return filePath, csvFilePath
}

func Header(pdf *gofpdf.Fpdf, reportTime string) {
	// Add logo placeholder
	pageWidth, _ := pdf.GetPageSize()

	pdf.SetFillColor(constants.HeaderBg[0], constants.HeaderBg[1], constants.HeaderBg[2])
	pdf.Rect(0, 0, pageWidth, 25, "F")
	logoPath := "pkg/public/work2.png"
	orgCompanyPath := "pkg/public/nibsslogo.png"

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

	//fmt.Println(pageWidth, pageWidth/3, pageWidth*.75, pageWidth-50)
	pdf.Image(logoPath, pageWidth-50, 5, 35, 0, false, "", 0, "")
}

func ChartSection(pdf *gofpdf.Fpdf, metrics []internal.Metric) {
	pdf.AddPage()

	// Section Title
	pdf.SetFillColor(constants.TitleBg[0], constants.TitleBg[1], constants.TitleBg[2])
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
		pdf.Text(15, float64(startY), metric.AgentHostName)

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
	pdf.SetFillColor(constants.TableBg[0], constants.TableBg[1], constants.TableBg[2])
	pdf.Rect(x, y, width, height, "F")

	// Usage bar
	if percentage > 80 {
		pdf.SetFillColor(constants.AlertColor[0], constants.AlertColor[1], constants.AlertColor[2])
	} else {
		pdf.SetFillColor(constants.NormalColor[0], constants.NormalColor[1], constants.NormalColor[2])
	}
	pdf.Rect(x, y, width*(percentage/100), height, "F")
}

//
//func MetricsTable(pdf *gofpdf.Fpdf, metrics []internal.Metric, tableHeaders []string) {
//	pageWidth, _ := pdf.GetPageSize()
//	marginLeft, _, marginRight, _ := pdf.GetMargins()
//	tableWidth := pageWidth - marginLeft - marginRight
//
//	pdf.SetFont("Arial", "B", 10)
//	pdf.SetTextColor(0, 0, 0)
//	pdf.Text(pdf.GetX()+5, pdf.GetY(), "Infrastructure Report")
//
//	colWidth := 60.0
//	lineHeight := 10.0
//
//	// Table headers
//	for _, header := range tableHeaders {
//		pdf.CellFormat(colWidth, lineHeight, header, "1", 0, "C", false, 0, "")
//	}
//	//pdf.Ln(-1)
//
//	startX := pdf.GetX()
//	startY := pdf.GetY()
//
//	// Calculate wrapped text height
//	pdf.MultiCell(colWidth, lineHeight, longText, "1", "", false)
//	endY := pdf.GetY()
//
//	// Reset position and draw the second column
//	pdf.SetXY(startX+colWidth, startY)
//
//	// Draw table header
//	pdf.SetFillColor(constants.HeaderBg[0], constants.HeaderBg[1], constants.HeaderBg[2])
//	pdf.SetTextColor(255, 255, 255)
//	pdf.SetFont("Arial", "B", 10)
//
//	x := marginLeft
//	//for i, header := range tableHeaders {
//	//	pdf.Rect(x, pdf.GetY(), colWidths[i], 10, "DF")
//	//	pdf.Text(x+2, pdf.GetY()+5, header)
//	//	x += colWidths[i]
//	//}
//	//pdf.Ln(10)
//
//	// Add some content below the logo
//	//pdf.SetFont("Arial", "", 10)
//	//for _, m := range metrics {
//	//	pdf.Cell(colWidths[0], 10, m.ServerName)
//	//	pdf.Cell(colWidths[1], 10, m.IPAddress)
//	//	pdf.Cell(colWidths[2], 10, fmt.Sprintf("%.1f", m.CpuUsage))
//	//	pdf.Cell(colWidths[3], 10, fmt.Sprintf("%.1f", m.MemoryUsage))
//	//	pdf.Cell(colWidths[4], 10, fmt.Sprintf("%.1f", m.CurrentDiskUtilization))
//	//	//pdf.Cell(colWidths[5], 10, fmt.Sprintf("%.2f MB", m.CurrentDiskUtilization))
//	//	//pdf.Cell(colWidths[6], 10, fmt.Sprintf("%.2f MB", m.NetworkOut))
//	//	//pdf.Cell(colWidths[7], 10, fmt.Sprintf("%d", m.ActiveUsers))
//	//	//pdf.Cell(colWidths[8], 10, fmt.Sprintf("%d", m.AlertCount))
//	//	pdf.Ln(-1)
//	//}
//
//	// Table content
//	//pdf.SetTextColor(0, 0, 0)
//	pdf.SetFont("Arial", "", 7)
//	rowColor := false
//
//	for _, m := range metrics {
//		x = marginLeft
//		if rowColor {
//			pdf.SetFillColor(constants.TableBg[0], constants.TableBg[1], constants.TableBg[2])
//		} else {
//			pdf.SetFillColor(255, 255, 255)
//		}
//
//		if m.CpuUsage > 80 || m.MemoryUsage > 80 || m.CurrentDiskUtilization > 80 {
//			pdf.SetTextColor(constants.AlertColor[0], constants.AlertColor[1], constants.AlertColor[2])
//		} else {
//			pdf.SetTextColor(0, 32, 96) // Deep blue for normal values
//		}
//
//		// Draw row background
//		pdf.Rect(x, pdf.GetY(), tableWidth, 8, "F")
//
//		pdf.Rect(pdf.GetX(), pdf.GetY(), colWidth, cellHeight, "D")
//
//		// Helper function to format values
//		formatValue := func(value float64) string {
//			if value < 0 { // Check for invalid values
//				return "N/A"
//			}
//			return fmt.Sprintf("%.1f", value)
//		}
//
//		// Draw cell content
//		pdf.Text(x+2, pdf.GetY()+5, m.AgentHostName)
//		x += colWidths[0]
//		pdf.Text(x+2, pdf.GetY()+5, m.AgentHostAddress)
//		x += colWidths[0]
//		pdf.Text(x+2, pdf.GetY()+5, formatValue(m.CpuUsage))
//		x += colWidths[1]
//		pdf.Text(x+2, pdf.GetY()+5, formatValue(m.MemoryUsage))
//		x += colWidths[2]
//		pdf.Text(x+2, pdf.GetY()+5, formatValue(m.CurrentDiskUtilization))
//		//x += colWidths[3]
//		//pdf.Text(x+2, pdf.GetY()+5, fmt.Sprintf("%.2f MB", m.NetworkIn))
//		//x += colWidths[4]
//		//pdf.Text(x+2, pdf.GetY()+7, fmt.Sprintf("%.2f MB", m.NetworkOut))
//		//x += colWidths[5]
//		//pdf.Text(x+2, pdf.GetY()+7, fmt.Sprintf("%d", m.ActiveUsers))
//		//x += colWidths[6]
//		//pdf.Text(x+2, pdf.GetY()+7, fmt.Sprintf("%d", m.AlertCount))
//
//		pdf.Ln(10)
//		rowColor = !rowColor
//	}
//
//	// Fetch process details via Agent API.
//	for _, mx := range metrics {
//		processes, err := internal.ServerResourceDetails(mx.AgentAPIBaseURL, 10)
//
//		if err != nil {
//			log.Printf("Error fetching process details: %v", err)
//			pdf.SetFont("Arial", "", 12)
//			pdf.Cell(190, 10, "Failed to fetch process details")
//		} else {
//			ProcessTable(pdf, processes)
//		}
//	}
//}

func MetricsTable(pdf *gofpdf.Fpdf, metrics []internal.Metric, tableHeaders []string) {
	// Define column widths for each column.
	//colWidths := []float64{30, 50, 20, 40, 30, 30, 30}
	colWidths := []float64{30, 50, 30, 30, 50, 40, 20}

	// Set header font.
	pdf.SetFont("Arial", "B", 10)
	// Table header.
	for i, header := range tableHeaders {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Set normal font for table rows.
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(0, 32, 96)
	rowColor := false

	for _, m := range metrics {
		if rowColor {
			pdf.SetFillColor(constants.TableBg[0], constants.TableBg[1], constants.TableBg[2])
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		row := []string{
			m.AgentHostName,
			m.AgentHostAddress,
			fmt.Sprintf("%.2f%%", m.CpuUsage),
			fmt.Sprintf("%.2f%%", m.MemoryUsage),
			fmt.Sprintf("%.2f%%", m.CurrentDiskUtilization),
			m.TotalStorageCapacity,
		}

		for i, cell := range row {
			//pdf.CellFormat(colWidths[i], 7, cell, "1", 0, "C", false, 0, "")

			WrappedTextCell(pdf, colWidths[i], 7, cell)
		}

		rowColor = !rowColor
		pdf.Ln(-1)
	}

	pdf.Ln(15)

	// Fetch process details via Agent API.
	for _, mx := range metrics {
		processes, err := internal.ServerResourceDetails(mx.AgentAPI, 10)

		if err != nil {
			log.Printf("Error fetching process details: %v", err)
			pdf.SetFont("Arial", "", 12)
			pdf.Cell(190, 10, "Failed to fetch process details.")
		} else {
			ProcessTable(pdf, processes)
		}
	}

	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(0, 10, fmt.Sprintf("All Rights Reserved. Company Name. Page %d", pdf.PageNo()), "0", 0, "C", false, 0, "")
	})
}

func NetworkMetricsTable(pdf *gofpdf.Fpdf, metrics []internal.Metric, tableHeaders []string) {
	// Define column widths for each column.
	colWidths := []float64{30, 80, 30, 50, 40, 20, 20}

	// Set header font.
	pdf.SetFont("Arial", "B", 10)
	// Table header.
	for i, header := range tableHeaders {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Set normal font for table rows.
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 32, 96)
	rowColor := false

	for _, m := range metrics {
		if rowColor {
			pdf.SetFillColor(constants.TableBg[0], constants.TableBg[1], constants.TableBg[2])
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		row := []string{
			m.AgentHostAddress,
			m.AgentHostName,
			fmt.Sprintf("%.2f%%", m.CpuUsage),
			fmt.Sprintf("%.2f%%", m.MemoryUsage),
			fmt.Sprintf("%.2f%%", m.CurrentDiskUtilization),
			m.TotalStorageCapacity,
		}

		for i, cell := range row {
			pdf.CellFormat(colWidths[i], 7, cell, "1", 0, "C", false, 0, "")
		}

		rowColor = !rowColor
		pdf.Ln(-1)
	}
}

func WrappedTextCell(pdf *gofpdf.Fpdf, width float64, height float64, text string) {
	// Save current X and Y
	x := pdf.GetX()
	y := pdf.GetY()

	// Draw border manually if needed
	pdf.MultiCell(width, height, text, "1", "L", false)

	// Move cursor to the right of the cell (simulate single cell width)
	pdf.SetXY(x+width, y)
}

// ProcessTable adds a table to the PDF for a given server's process list.
func ProcessTable(pdf *gofpdf.Fpdf, processes []internal.ProcessResourceUsage) {
	// Define column widths for each column.
	colWidths := []float64{20, 80, 20, 40, 30, 20, 20}
	// Table header.
	headers := []string{"PID", "Process Name", "Status", "Create Time", "Username", "CPU %", "Memory %"}

	// Set header font.
	pdf.SetFont("Arial", "B", 10)
	for i, header := range headers {
		//pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", false, 0, "")
		WrappedTextCell(pdf, colWidths[i], 10, header)
	}
	pdf.Ln(-1)

	// Set normal font for table rows.
	pdf.SetFont("Arial", "", 10)
	for _, proc := range processes {
		// Format the Unix timestamp to a readable date.

		row := []string{
			fmt.Sprintf("%d", proc.PID),
			proc.Name,
			strings.ToUpper(proc.Status),
			time.Unix(int64(proc.CreateTime), 0).Format("2006-01-02 15:04:05"),
			proc.Username,
			fmt.Sprintf("%.2f%%", proc.CPUPercent),
			fmt.Sprintf("%.2f%%", proc.MemoryPercent),
		}

		for i, cell := range row {
			//pdf.CellFormat(colWidths[i], 7, cell, "1", 0, "C", false, 0, "")
			WrappedTextCell(pdf, colWidths[i], 7, cell)
		}
		pdf.Ln(-1)
	}
}

func GenerateCSV(metrics []internal.Metric, tableHeaders []string) string {
	// Construct the filename and path
	if err := os.MkdirAll(constants.ReportsDir, os.ModePerm); err != nil {
		log.Fatalf("Error creating reports directory: %s", err.Error())
	}

	currentTime := time.Now()
	formattedTime := currentTime.Format("January 2, 2006 15:04:05")

	// Create the CSV file
	fileName := "Hourly_IT_Report_" + formattedTime + ".csv"
	filePath := filepath.Join(constants.ReportsDir, fileName)

	csvFile, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	if writerErr := writer.Write(tableHeaders); writerErr != nil {
		log.Panic(writerErr)
	}

	// Write data rows
	// Write each process as a row in the CSV file
	for _, mx := range metrics {
		row := []string{
			mx.AgentHostName,
			mx.AgentHostAddress,
			fmt.Sprintf("%.2f%%", mx.CpuUsage),
			fmt.Sprintf("%.2f%%", mx.MemoryUsage),
			fmt.Sprintf("%.2f%%", mx.CurrentDiskUtilization),
			mx.TotalStorageCapacity,
		}

		if err_ := writer.Write(row); err_ != nil {
			log.Panic(err_)
		}
	}

	fmt.Printf("Writing CSV Report to %s", filePath)

	log.Printf("Writing CSV Report to %s", filePath)

	return filePath
}

// GeneratePDF metrics ServiceMonitorStatus
func GeneratePDF(metrics []internal.Metric, headers []string) string {
	// Get current time and format it
	currentTime := time.Now()
	formattedTime := currentTime.Format("January 2, 2006 15:04:05")
	//.Format("2006-01-02 15:04:05")

	// Initialize a new PDF document (A4 size, portrait orientation)
	pdf := gofpdf.New("L", "mm", "A4", "")

	// First page - Overview
	pdf.AddPage()
	Header(pdf, formattedTime)

	pdf.SetY(30)
	pdf.SetFont("Arial", "", 21)
	// Active / Inactive System / Scheduled for maintenance

	//var counts []SystemCount
	counts := []SystemCount{
		{"Active", 50000},
		{"Inactive", 2000},
		{"Scheduled", 302},
		{"Acknowledged", 31},
	}

	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(constants.TitleBg[0], constants.TitleBg[1], constants.TitleBg[2])
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
	pdf.SetY(60)
	MetricsTable(pdf, metrics, headers)

	// Network Devices Section
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(190, 10, "Top 10 Network Devices by Bandwidth Utilization")
	pdf.Ln(10)

	// Table headers
	pdf.SetFont("Arial", "B", 10)
	netHeaders := []string{"Device Name", "Interface", "Bandwidth (Mbps)", "Utilization %", "Last Updated"}
	NetworkMetricsTable(pdf, metrics, netHeaders)

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
	if err := os.MkdirAll(constants.ReportsDir, os.ModePerm); err != nil {
		log.Fatalf("Error creating reports directory: %s", err.Error())
	}

	//fileName := "Hourly_IT_Report.pdf"
	fileName := "Hourly_IT_Report_" + formattedTime + ".pdf"

	filePath := filepath.Join(constants.ReportsDir, fileName)
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
