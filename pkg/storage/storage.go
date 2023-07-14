package storage

import (
	"encoding/json"
	"fmt"
	"go-btc-downloader/pkg/node"
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

func Save(filename string, nodes []*node.Node) error {
	// save nodes as json
	fData := make([]string, len(nodes))
	for i, n := range nodes {
		fData[i] = n.EndpointSafe() // [addr]:port for ipv6
	}
	fDataJson, err := json.MarshalIndent(fData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %v", err)
	}
	err = os.WriteFile(filename, fDataJson, 0644)
	if err != nil {
		return fmt.Errorf("failed to write nodes: %v", err)
	}
	return nil
}
