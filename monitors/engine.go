package monitors

import (
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/utils"
	"sort"
	"time"
)

type AgentServiceChecker struct{}
type WebModulesServiceChecker struct{}
type SNMPServiceChecker struct{}

// MetricEngine Aggregates all metric sources by AppId and metric
func MetricEngine(metrics ...[][]utils.ServiceMonitorStatus) []utils.ServiceMonitorStatus {

	var allMessageList []utils.ServiceMonitorStatus

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	for _, messageArray := range metrics {
		for _, messages := range messageArray {
			allMessageList = append(allMessageList, messages...)
		}
	}

	// Sort based on FLAG
	sort.Slice(allMessageList, func(i, j int) bool {
		return allMessageList[i].LiveCheckFlag < allMessageList[j].LiveCheckFlag
	})

	// Print the sorted list
	fmt.Println(allMessageList)

	return allMessageList
}

type TimeSeriesData struct {
	Timestamp int64
	Value     float64
}

func CheckTSDataAboveThreshold(metricName string, entity string, tsData []TimeSeriesData, threshold float64, arrSequenceLength int) []struct {
	Timestamp string
	Values    []float64
} {
	var timestampArr []int64
	var valueArr []float64
	var thresholds []struct {
		Timestamp string
		Values    []float64
	}
	var sequenceData [][]float64

	// Populate timestampArr and valueArr
	for _, val := range tsData {
		timestampArr = append(timestampArr, val.Timestamp)
		valueArr = append(valueArr, val.Value)
	}

	// Group values into sequences
	for i := 0; i < len(valueArr); i += arrSequenceLength {
		if i+arrSequenceLength <= len(valueArr) {
			sequence := valueArr[i : i+arrSequenceLength]
			sequenceData = append(sequenceData, sequence)
		}
	}

	// Check for sequences above threshold
	for i, arr := range sequenceData {
		if allAboveThreshold(arr, threshold) && len(arr) >= arrSequenceLength {
			timestamp := timestampArr[i*arrSequenceLength]
			formattedTime := time.UnixMilli(timestamp).Format("2006-01-02 15:04:05")
			thresholds = append(thresholds, struct {
				Timestamp string
				Values    []float64
			}{Timestamp: formattedTime, Values: arr})
		}
	}

	return thresholds
}

func allAboveThreshold(arr []float64, threshold float64) bool {
	for _, num := range arr {
		if num <= threshold {
			return false
		}
	}
	return true
}

//func Run() {
//	//
//	sendTo := []string{"calebb.jnr@gmail.com"}
//	//
//	//	messages := MetricEngine()
//	//
//	//	// Construct the email subject
//	//	//subject := fmt.Sprintf("Alert: %d Threshold Messages for %s", len(groupMessages), group)
//	//	subject := fmt.Sprintf("Alert Threshold Messages")
//	//
//	actionURL := ""
//	//
//	err_ := messaging.SendEmail(sendTo, "Test Subject", "Hello World from Go")
//	if err_ != nil {
//		return
//	}
//
//	emailBody := messaging.FormatEmailMessageToSend("Hello World from Go", actionURL)
//	// Send the email
//	if err := messaging.SendEmail(sendTo, subject, emailBody); err != nil {
//		log.Printf("Failed to send email to %s: %v", group, err)
//	} else {
//		log.Printf("Alert sent to %s", group)
//	}
//
//	extraInfo := map[string]string{}
//
//	slackClient := messaging.SlackBotClient()
//	slackMessage := messaging.FormatSlackMessageToSend("Test Notification", "Hello World from Go", "critical", actionURL, extraInfo)
//
//	_, err := slackClient.SendSlackMessage("admin_x", slackMessage)
//	if err != nil {
//		return
//	}
//}
