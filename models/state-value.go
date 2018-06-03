package models

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type StateValue int

const (
	Ok StateValue = iota
	Warn
	Critical
	Down
	Unknown
)

func fromString(s string) StateValue {
	names := [4]string{
		"OK",
		"WARN",
		"CRITICAL",
		"DOWN",
	}

	for i, v := range names {
		if v == s {
			return StateValue(i)
		}
	}

	return Unknown
}

func (s StateValue) String() string {
	names := [5]string{
		"OK",
		"WARN",
		"CRITICAL",
		"DOWN",
		"UNKNOWN",
	}
	fmt.Printf("%d, %s\n", int(s), names[int(s)])
	return names[int(s)]
}

func (s *StateValue) MarshalJSON() ([]byte, error) {
	fmt.Println("Calling MarshalJSON")
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(s.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func (s *StateValue) UnmarshalJSON(b []byte) error {
	// unmarshal as string
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}
	res := fromString(str)
	*s = res

	return nil
}
