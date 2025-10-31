package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var CustomAnalyzer = &analysis.Analyzer{
	Name: "nopanic",
	Doc:  "check for panic func usage",
	Run:  run,
}

func run(p *analysis.Pass) (any, error) {
	for _, file := range p.Files {
		filename := p.Fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		var currentFunc *ast.FuncDecl

		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				currentFunc = x

			case *ast.CallExpr:
				if ident, ok := x.Fun.(*ast.Ident); ok {
					if ident.Name == "panic" {
						p.Reportf(
							x.Pos(),
							"usage of panic function is forbidden",
						)
					}
				}

				if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						pkgName := ident.Name
						funcName := sel.Sel.Name

						isMainFunc := currentFunc != nil &&
							currentFunc.Name.Name == "main"
						isMainPkg := p.Pkg.Name() == "main"

						if pkgName == "log" && funcName == "Fatal" {
							if !isMainFunc || !isMainPkg {
								p.Reportf(
									x.Pos(),
									"log.Fatal outside main",
								)
							}
						}

						if pkgName == "os" && funcName == "Exit" {
							if !isMainFunc || !isMainPkg {
								p.Reportf(
									x.Pos(),
									"os.Exit outside main",
								)
							}
						}
					}
				}
			}
			return true
		})
	}

	return nil, nil
}
