package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

const builderTemplate = `package {{.Package}}

type {{.BuilderName}} struct {
	d {{.StructName}}
}

func New{{.BuilderName}}() *{{.BuilderName}} {
	return &{{.BuilderName}}{}
}

{{range .Fields}}
// {{.Comment}}
func (b *{{$.BuilderName}}) Set{{.Name}}(v {{.Type}}) *{{$.BuilderName}} {
	b.d.{{.OriginalName}} = v
	return b
}
{{end}}

func (b *{{.BuilderName}}) Build() *{{.StructName}} {
	return &b.d
}

func (b *{{.BuilderName}}) ReadBuild() *{{.StructName}} {
	return &b.d
}
`

type Field struct {
	Name         string
	OriginalName string
	Type         string
	Comment      string
}

type BuilderData struct {
	Package     string
	StructName  string
	BuilderName string
	Fields      []Field
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-builder-gen <filename>")
		os.Exit(1)
	}

	filename := os.Args[1]
	generateBuilder(filename)
}

func generateBuilder(filename string) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		return
	}

	packageName := node.Name.Name

	ast.Inspect(node, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				structName := typeSpec.Name.Name
				builderName := structName + "Builder"
				var fields []Field

				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					fieldName := field.Names[0].Name
					fieldType := exprToString(field.Type)
					comment := ""
					if field.Comment != nil {
						comment = field.Comment.Text()
					}

					fields = append(fields, Field{
						Name:         strings.Title(fieldName),
						OriginalName: fieldName,
						Type:         fieldType,
						Comment:      strings.TrimSpace(comment),
					})
				}

				data := BuilderData{
					Package:     packageName,
					StructName:  structName,
					BuilderName: builderName,
					Fields:      fields,
				}

				outputFilename := strings.ToLower(structName) + "_builder.go"
				generateFile(outputFilename, data)
			}
		}
		return true
	})
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	default:
		return "unknown"
	}
}

func generateFile(filename string, data BuilderData) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	tmpl, err := template.New("builder").Parse(builderTemplate)
	if err != nil {
		fmt.Println("Error parsing template:", err)
		return
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Println("Error executing template:", err)
	}
}
