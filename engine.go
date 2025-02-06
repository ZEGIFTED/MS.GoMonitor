package main

import (
	"fmt"
	"sort"
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

func Run() ([]string, int) {
	for index, taskItem := range taskItems {

		fmt.Printf("%d: %s \n", index+1, taskItem)
	}

	v := len(taskItems)

	var x = append(taskItems, "test")

	return x, v
}
