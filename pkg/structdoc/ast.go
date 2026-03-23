package structdoc

import (
	"fmt"
	"go/ast"
	"go/token"
	gotypes "go/types"

	"golang.org/x/tools/go/packages"
)

func collectPackageInfo(dir string, targetFile string, pkgName string) (map[string]*ast.StructType, map[string]bool, map[ast.Expr]gotypes.Type, error) {
	pkgs, err := packages.Load(&packages.Config{
		// NeedCompiledGoFiles keeps file/package mapping aligned with build tags.
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  dir,
		// file=... loads the package(s) that actually contain this file.
	}, "file="+targetFile)
	if err != nil {
		return nil, nil, nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, nil, nil, fmt.Errorf("failed to load packages")
	}

	types := map[string]*ast.StructType{}
	unmarshal := map[string]bool{}
	fieldTypes := map[ast.Expr]gotypes.Type{}
	found := false
	for _, pkg := range pkgs {
		if pkgName != "" && pkg.Name != pkgName {
			continue
		}
		found = true
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if ok && isUnmarshalTOMLMethod(funcDecl) {
					// UnmarshalTOML types are rendered as leaf values.
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
					if pkg.TypesInfo != nil {
						for _, field := range st.Fields.List {
							if typ := pkg.TypesInfo.TypeOf(field.Type); typ != nil {
								fieldTypes[field.Type] = typ
							}
						}
					}
				}
			}
		}
	}
	if !found {
		return nil, nil, nil, fmt.Errorf("package %q not found", pkgName)
	}
	return types, unmarshal, fieldTypes, nil
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
