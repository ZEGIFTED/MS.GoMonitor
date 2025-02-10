package main

import (
	"fmt"
	"sort"
	"time"
)

var taskItems = []string{"hello", "world"}

//test_dict := make(map[string]string)

func MetricEngine(metrics ...[][]ServiceMonitorStatus) []ServiceMonitorStatus {
	var allMessageList []ServiceMonitorStatus

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

func checkTSDataAboveThreshold(metricName string, entity string, tsData []TimeSeriesData, threshold float64, arrSequenceLength int) []struct {
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
