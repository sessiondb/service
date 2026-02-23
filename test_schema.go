package main

import (
	"fmt"
)

func main() {
	privMap := make(map[string]map[string]bool)

	// Simulate user 'mouli'@'%' inherited from 'cloudsqlsuperuser'@'%'
	// cloudsqlsuperuser has *.*
	key := "*.*"
	privMap[key] = make(map[string]bool)
	privMap[key]["SELECT"] = true
	privMap[key]["INSERT"] = true

	dbName := "promiseEngine"
	tableName := "buyXGetY"

	globalWildcardPrivs := privMap["*.*"]
	dbWildcardPrivs := privMap[fmt.Sprintf("%s.*", dbName)]
	tableKey := fmt.Sprintf("%s.%s", dbName, tableName)

	combinedPrivs := make(map[string]bool)
	if privs, ok := privMap[tableKey]; ok {
		for p := range privs {
			combinedPrivs[p] = true
		}
		fmt.Println("Table Match")
	}
	for p := range dbWildcardPrivs {
		combinedPrivs[p] = true
	}
	for p := range globalWildcardPrivs {
		combinedPrivs[p] = true
	}

	fmt.Printf("Combined for %s: %v\n", tableKey, combinedPrivs)
}
