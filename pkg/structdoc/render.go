package structdoc

import (
	"go/ast"
	gotypes "go/types"
	"reflect"
	"strconv"
	"strings"
)

type generator struct {
	tag        string
	types      map[string]*ast.StructType
	unmarshal  map[string]bool
	fieldTypes map[ast.Expr]gotypes.Type
	output     strings.Builder
}

func (g *generator) renderFields(typ *ast.StructType, headingLevel int) {
	for _, field := range typ.Fields.List {
		if len(field.Names) == 0 {
			embeddedName := embeddedTypeName(field.Type)
			if embeddedName == "" {
				continue
			}
			embedded, ok := g.types[embeddedName]
			if !ok || g.unmarshal[embeddedName] {
				continue
			}
			g.renderFields(embedded, headingLevel)
			continue
		}

		name := g.fieldName(field)
		g.output.WriteString("\n\n")
		g.output.WriteString(strings.Repeat("#", headingLevel))
		g.output.WriteByte(' ')
		g.output.WriteString(name)

		typeName := fieldTypeName(field)
		if nested, ok := g.types[typeName]; ok && !g.unmarshal[typeName] {
			if def, ok := g.fieldTag(field, "default"); ok {
				g.output.WriteString("\n- Default: `")
				g.output.WriteString(def)
				g.output.WriteString("`")
			}
			if doc := fieldDoc(field); doc != "" {
				g.output.WriteString("\n")
				g.output.WriteString(doc)
			}
			g.renderFields(nested, headingLevel+1)
			continue
		}

		if typ, ok := g.fieldTypeLabel(field.Type); ok {
			g.output.WriteString("\n- Type: `")
			g.output.WriteString(typ)
			g.output.WriteString("`")
		}
		if def, ok := g.fieldTag(field, "default"); ok {
			g.output.WriteString("\n- Default: `")
			g.output.WriteString(g.fieldDefaultLabel(field.Type, def))
			g.output.WriteString("`")
		}
		if doc := fieldDoc(field); doc != "" {
			g.output.WriteString("\n- Description: ")
			g.output.WriteString(doc)
		}
	}
}

func (g *generator) fieldDefaultLabel(expr ast.Expr, def string) string {
	if typ, ok := g.fieldTypes[expr]; ok {
		if basic, ok := typ.Underlying().(*gotypes.Basic); ok && basic.Kind() == gotypes.String {
			return strconv.Quote(def)
		}
		return def
	}
	if label, ok := g.fieldTypeLabel(expr); ok && label == "string" {
		return strconv.Quote(def)
	}
	return def
}

func embeddedTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		ident, ok := t.X.(*ast.Ident)
		if !ok {
			return ""
		}
		return ident.Name
	default:
		return ""
	}
}

func (g *generator) fieldName(field *ast.Field) string {
	tag, _ := strconv.Unquote(field.Tag.Value)
	return reflect.StructTag(tag).Get(g.tag)
}

func (g *generator) fieldTag(field *ast.Field, key string) (string, bool) {
	if field.Tag == nil {
		return "", false
	}
	tag, _ := strconv.Unquote(field.Tag.Value)
	value, ok := reflect.StructTag(tag).Lookup(key)
	if !ok {
		return "", false
	}
	return value, true
}

func fieldTypeName(field *ast.Field) string {
	ident, ok := field.Type.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

func (g *generator) fieldTypeLabel(expr ast.Expr) (string, bool) {
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
		elem, ok := g.fieldTypeLabel(t.Elt)
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
		elem, ok := g.fieldTypeLabel(t.X)
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
	return strings.ReplaceAll(strings.TrimSpace(field.Doc.Text()), "\n", "  \n")
}
