package storage

import (
	"encoding/json"
	"os"
)

func Load(filename string) ([]string, error) {
	var ret []string
	// read from json
	fData, err := os.ReadFile(filename)
	if err != nil {
		return ret, err
	}
	err = json.Unmarshal(fData, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}
