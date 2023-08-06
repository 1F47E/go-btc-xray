package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/1F47E/go-btc-xray/pkg/client/node"
	"github.com/1F47E/go-btc-xray/pkg/config"
)

var cfg = config.New()

func CreateDir(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return nil
}

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

func Save(nodes []*node.Node) error {
	path := filepath.Join(cfg.DataDir, cfg.NodesFilename)
	// save nodes as json
	fData := make([]string, len(nodes))
	for i, n := range nodes {
		fData[i] = n.EndpointSafe() // [addr]:port for ipv6
	}
	fDataJson, err := json.MarshalIndent(fData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %v", err)
	}
	err = os.WriteFile(path, fDataJson, 0644)
	if err != nil {
		return fmt.Errorf("failed to write nodes: %v", err)
	}
	return nil
}
