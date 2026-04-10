package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func UpdateWorkflowGlobalVars(path string, vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("workflow %s is not a mapping document", path)
	}

	root := doc.Content[0]
	globalVars := ensureMappingValue(root, "global_vars")
	for key, value := range vars {
		setMappingString(globalVars, key, value)
	}

	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close encoder for %s: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, out.Bytes(), info.Mode()); err != nil {
		return fmt.Errorf("write %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func ensureMappingValue(parent *yaml.Node, key string) *yaml.Node {
	for idx := 0; idx+1 < len(parent.Content); idx += 2 {
		if parent.Content[idx].Value == key {
			if parent.Content[idx+1].Kind != yaml.MappingNode {
				parent.Content[idx+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
			return parent.Content[idx+1]
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content, keyNode, valueNode)
	return valueNode
}

func setMappingString(mapping *yaml.Node, key, value string) {
	for idx := 0; idx+1 < len(mapping.Content); idx += 2 {
		if mapping.Content[idx].Value == key {
			mapping.Content[idx+1].Kind = yaml.ScalarNode
			mapping.Content[idx+1].Tag = "!!str"
			mapping.Content[idx+1].Value = value
			return
		}
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}
