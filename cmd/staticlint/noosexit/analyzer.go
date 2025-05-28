// Package noosexit реализует анализатор, который запрещает прямые вызовы os.Exit
// в функции main пакета main.
//
// Анализатор помогает писать более тестируемый код, заставляя разработчиков
// использовать паттерн с функцией run() и обработкой ошибок через log.Fatal
// или другие механизмы в функции main.
//
// Пример проблемного кода:
//
//	package main
//
//	import "os"
//
//	func main() {
//	    if someCondition {
//	        os.Exit(1) // ❌ Будет найдено анализатором
//	    }
//	}
//
// Пример правильного кода:
//
//	package main
//
//	import (
//	    "log"
//	    "os"
//	)
//
//	func main() {
//	    if err := run(); err != nil {
//	        log.Fatal(err) // ✅ Правильный подход
//	    }
//	}
//
//	func run() error {
//	    if someCondition {
//	        os.Exit(1) // ✅ Разрешено вне функции main
//	    }
//	    return nil
//	}
package noosexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer представляет анализатор для обнаружения прямых вызовов os.Exit
// в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "запрещает прямые вызовы os.Exit в функции main пакета main",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// run выполняет анализ кода и ищет запрещенные вызовы os.Exit.
func run(pass *analysis.Pass) (interface{}, error) {
	// Проверяем только свои исходные пакеты, игнорируя временные от компилятора
	if !strings.HasPrefix(pass.Pkg.Path(), "github.com/AlenaMolokova/http") {
		return nil, nil
	}

	// Проверяем только пакеты main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Отслеживаем текущую функцию
	var currentFunc *ast.FuncDecl

	// Фильтруем функции и вызовы
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		switch n := node.(type) {
		case *ast.FuncDecl:
			currentFunc = n
		case *ast.CallExpr:
			// Проверяем вызов только если мы находимся в функции main
			if currentFunc != nil && currentFunc.Name.Name == "main" && currentFunc.Recv == nil {
				if isOsExitCall(n, pass.TypesInfo) {
					pass.Reportf(n.Pos(), "прямой вызов os.Exit в функции main запрещен")
				}
			}
		}
	})

	return nil, nil
}

// isOsExitCall проверяет, является ли вызов функции вызовом os.Exit.
func isOsExitCall(call *ast.CallExpr, info *types.Info) bool {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		// Проверяем вызов вида pkg.Exit
		if ident, ok := fun.X.(*ast.Ident); ok {
			if fun.Sel.Name == "Exit" {
				// Проверяем, что пакет действительно "os"
				if obj := info.Uses[ident]; obj != nil {
					if pkg, ok := obj.(*types.PkgName); ok {
						return pkg.Imported().Path() == "os"
					}
				}
			}
		}
	}
	return false
}
