package components

import (
	"encoding/json"
	"fmt"
)

// JSON marshals an object to a JSON string, returning "{}" on error
func JSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return "{}"
	}
	return string(b)
}
