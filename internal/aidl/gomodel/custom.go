package gomodel

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadCustomParcelableMappings(path string) (map[string]CustomParcelableConfig, error) {
	file, err := LoadTypeMappingFile(path)
	if err != nil {
		return nil, err
	}

	out := make(map[string]CustomParcelableConfig, len(file.Parcelables))
	for _, entry := range file.Parcelables {
		switch {
		case entry.AIDLName == "":
			return nil, fmt.Errorf("type mappings %s: custom parcelable aidl_name is required", path)
		case entry.GoPackage == "":
			return nil, fmt.Errorf("type mappings %s: %s go_package is required", path, entry.AIDLName)
		case entry.GoType == "":
			return nil, fmt.Errorf("type mappings %s: %s go_type is required", path, entry.AIDLName)
		case entry.WriteFunc == "":
			return nil, fmt.Errorf("type mappings %s: %s write_func is required", path, entry.AIDLName)
		case entry.ReadFunc == "":
			return nil, fmt.Errorf("type mappings %s: %s read_func is required", path, entry.AIDLName)
		}
		if _, exists := out[entry.AIDLName]; exists {
			return nil, fmt.Errorf("type mappings %s: duplicate custom parcelable %s", path, entry.AIDLName)
		}
		out[entry.AIDLName] = entry
	}
	return out, nil
}

func LoadStableInterfaceMappings(path string) (map[string]StableInterfaceConfig, error) {
	file, err := LoadTypeMappingFile(path)
	if err != nil {
		return nil, err
	}

	out := make(map[string]StableInterfaceConfig, len(file.Interfaces))
	for _, entry := range file.Interfaces {
		switch {
		case entry.AIDLName == "":
			return nil, fmt.Errorf("type mappings %s: stable interface aidl_name is required", path)
		case entry.Version <= 0:
			return nil, fmt.Errorf("type mappings %s: %s version must be > 0", path, entry.AIDLName)
		case entry.Hash == "":
			return nil, fmt.Errorf("type mappings %s: %s hash is required", path, entry.AIDLName)
		}
		if _, exists := out[entry.AIDLName]; exists {
			return nil, fmt.Errorf("type mappings %s: duplicate stable interface %s", path, entry.AIDLName)
		}
		out[entry.AIDLName] = entry
	}
	return out, nil
}

func LoadTypeMappingFile(path string) (*TypeMappingFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read type mappings %s: %w", path, err)
	}

	var file TypeMappingFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("decode type mappings %s: %w", path, err)
	}
	if file.Version != 1 {
		return nil, fmt.Errorf("type mappings %s: unsupported version %d", path, file.Version)
	}
	return &file, nil
}

func customImportAlias(goPackage string) string {
	goPackage = strings.TrimSuffix(goPackage, "/")
	parts := strings.Split(goPackage, "/")
	switch len(parts) {
	case 0:
		return "customparcelable"
	case 1:
		return sanitizePackageName(parts[0])
	default:
		return sanitizePackageName(parts[len(parts)-2] + "_" + parts[len(parts)-1])
	}
}
