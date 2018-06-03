package debug

import (
	"encoding/json"
	"fmt"
	"kinetik-server/models"
)

func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}

func IntToPointer(a int) *int {
	return &a
}

func FloatToPointer(a float32) *float32 {
	return &a
}

func StateToPointer(a models.StateValue) *models.StateValue {
	return &a
}
