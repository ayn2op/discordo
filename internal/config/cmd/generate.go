package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type walker struct {
	pendingLine bool
	targetLine  int
	file        *token.File
	root        string
	types       map[string]*ast.StructType
	unmarshal   map[string]bool
	output      strings.Builder
}

func newWalker(targetLine int, file *token.File) *walker {
	w := &walker{
		targetLine: targetLine,
		file:       file,
		types:      map[string]*ast.StructType{},
		unmarshal:  map[string]bool{},
	}
	w.output.WriteString("# Configuration")
	return w
}

var _ ast.Visitor = (*walker)(nil)

func (w *walker) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.Comment:
		if !node.Pos().IsValid() {
			return w
		}

		// The next type after //go:generate is treated as the root config struct.
		line := w.file.Line(node.Pos())
		if line == w.targetLine {
			w.pendingLine = true
		}
	case *ast.TypeSpec:
		if _, ok := node.Type.(*ast.StructType); !ok {
			return w
		}
		if w.pendingLine {
			w.pendingLine = false
			w.root = node.Name.Name
		}
	}
	return w
}

func (w *walker) render() {
	// Root fields start at ## (level 2), nested fields increase heading depth.
	w.renderFields(w.types[w.root], 2)
	w.output.WriteByte('\n')
}

func (w *walker) renderFields(typ *ast.StructType, headingLevel int) {
	for _, field := range typ.Fields.List {
		if len(field.Names) == 0 {
			// Ignore embedded fields; config fields are all named.
			continue
		}

		name := fieldName(field)
		w.output.WriteString("\n\n")
		w.output.WriteString(strings.Repeat("#", headingLevel))
		w.output.WriteByte(' ')
		w.output.WriteString(name)

		typeName := fieldTypeName(field)
		// Types with UnmarshalTOML are treated as leaf values, not nested sections.
		if nested, ok := w.types[typeName]; ok && !w.unmarshal[typeName] {
			// Struct-typed field: render heading/description, then nested fields.
			if doc := fieldDoc(field); doc != "" {
				w.output.WriteString("\n")
				w.output.WriteString(doc)
			}
			// Recurse into package-local structs unless type is UnmarshalTOML leaf.
			w.renderFields(nested, headingLevel+1)
			continue
		}

		if typ, ok := w.fieldTypeLabel(field.Type); ok {
			w.output.WriteString("\n- Type: `")
			w.output.WriteString(typ)
			w.output.WriteString("`")
		}

		if doc := fieldDoc(field); doc != "" {
			w.output.WriteString("\n- Description: ")
			w.output.WriteString(doc)
		}
	}
}

func fieldName(field *ast.Field) string {
	tag, _ := strconv.Unquote(field.Tag.Value)
	return reflect.StructTag(tag).Get("toml")
}

func fieldTypeName(field *ast.Field) string {
	ident, ok := field.Type.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

func (w *walker) fieldTypeLabel(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, true
	case *ast.SelectorExpr:
		x, ok := t.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		return x.Name + "." + t.Sel.Name, true
	case *ast.ArrayType:
		elem, ok := w.fieldTypeLabel(t.Elt)
		if !ok {
			return "", false
		}
		if t.Len == nil {
			return "[]" + elem, true
		}
		lit, ok := t.Len.(*ast.BasicLit)
		if !ok {
			return "", false
		}
		return "[" + lit.Value + "]" + elem, true
	case *ast.StarExpr:
		elem, ok := w.fieldTypeLabel(t.X)
		if !ok {
			return "", false
		}
		return "*" + elem, true
	default:
		return "", false
	}
}

func fieldDoc(field *ast.Field) string {
	if field.Doc == nil {
		return ""
	}
	// Keep user-authored line breaks as hard line breaks in Markdown.
	return strings.ReplaceAll(strings.TrimSpace(field.Doc.Text()), "\n", "  \n")
}

func main() {
	targetFile := os.Getenv("GOFILE")
	targetPackage := os.Getenv("GOPACKAGE")

	targetLineRaw := os.Getenv("GOLINE")
	targetLine, err := strconv.Atoi(targetLineRaw)
	if err != nil {
		panic(err)
	}

	fileSet := token.NewFileSet()
	astFile, err := parser.ParseFile(fileSet, targetFile, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	tokenFile := fileSet.File(astFile.Pos())
	wlkr := newWalker(targetLine, tokenFile)
	ast.Walk(wlkr, astFile)
	types, unmarshal, err := collectPackageInfo(fileSet, ".", targetPackage)
	if err != nil {
		panic(err)
	}
	wlkr.types = types
	wlkr.unmarshal = unmarshal
	wlkr.render()

	root, err := findModuleRoot()
	if err != nil {
		panic(err)
	}
	docDir := filepath.Join(root, "doc")
	if err := os.MkdirAll(docDir, 0o700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "config.md"), []byte(wlkr.output.String()), 0o600); err != nil {
		panic(err)
	}
}

// Collect package-local struct declarations and UnmarshalTOML receiver types.
func collectPackageInfo(fileSet *token.FileSet, dir string, pkgName string) (map[string]*ast.StructType, map[string]bool, error) {
	pkgs, err := parser.ParseDir(fileSet, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	pkg := pkgs[pkgName]
	if pkg == nil {
		return nil, nil, os.ErrNotExist
	}
	types := map[string]*ast.StructType{}
	unmarshal := map[string]bool{}
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if ok && isUnmarshalTOMLMethod(funcDecl) {
				// Pointer and value receivers both mark the named type as leaf/listable.
				if recv := receiverTypeName(funcDecl.Recv.List[0].Type); recv != "" {
					unmarshal[recv] = true
				}
			}

			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				types[typeSpec.Name.Name] = st
			}
		}
	}

	return types, unmarshal, nil
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func isUnmarshalTOMLMethod(fn *ast.FuncDecl) bool {
	if fn == nil || fn.Name == nil || fn.Name.Name != "UnmarshalTOML" {
		return false
	}
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	if fn.Type == nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	resultType, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && resultType.Name == "error"
}

// https://github.com/golang/go/blob/cfb67d08712c64308ccaa13870e119d517743271/src/cmd/go/internal/modload/init.go#L1704
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			return dir, nil
		}
		// Walk upward until filesystem root.
		next := filepath.Dir(dir)
		if next == dir {
			return "", os.ErrNotExist
		}
		dir = next
	}
}
