package structdoc

import (
	"fmt"
	"path/filepath"
)

func Generate(
	targetFile string,
	packageName string,
	packageDir string,
	rootType string,
	tag string,
) (string, error) {
	absTargetFile, err := filepath.Abs(targetFile)
	if err != nil {
		return "", err
	}
	if packageDir == "" {
		packageDir = "."
	}
	if rootType == "" {
		return "", fmt.Errorf("root type is required")
	}
	if tag == "" {
		return "", fmt.Errorf("tag is required")
	}

	types, unmarshal, fieldTypes, err := collectPackageInfo(packageDir, absTargetFile, packageName)
	if err != nil {
		return "", err
	}

	// Root type drives the section hierarchy.
	root := types[rootType]
	if root == nil {
		return "", fmt.Errorf("root type %q not found", rootType)
	}

	g := &generator{
		tag:        tag,
		types:      types,
		unmarshal:  unmarshal,
		fieldTypes: fieldTypes,
	}
	g.renderFields(root, 2)
	g.output.WriteByte('\n')
	return g.output.String(), nil
}
