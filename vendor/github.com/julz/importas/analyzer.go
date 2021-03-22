package importas

import (
	"fmt"
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

type Config struct {
	RequiredAlias map[string]string // path -> alias.
}

var Analyzer = &analysis.Analyzer{
	Name: "importas",
	Doc:  "Enforces consistent import aliases",
	Run:  run,

	Flags: flags(),

	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

var config = Config{
	RequiredAlias: make(map[string]string),
}

func run(pass *analysis.Pass) (interface{}, error) {
	return runWithConfig(config, pass)
}

func runWithConfig(config Config, pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder([]ast.Node{(*ast.ImportSpec)(nil)}, func(n ast.Node) {
		visitImportSpecNode(n.(*ast.ImportSpec), pass)
	})

	return nil, nil
}

func visitImportSpecNode(node *ast.ImportSpec, pass *analysis.Pass) {
	if node.Name == nil {
		return // not aliased at all, ignore. (Maybe add strict mode for this?).
	}

	alias := node.Name.String()
	if alias == "." {
		return // Dot aliases are generally used in tests, so ignore.
	}

	if strings.HasPrefix(alias, "_") {
		return // Used by go test and for auto-includes, not a conflict.
	}

	path, err := strconv.Unquote(node.Path.Value)
	if err != nil {
		pass.Reportf(node.Pos(), "import not quoted")
	}

	if required, exists := config.RequiredAlias[path]; exists && required != alias {
		pass.Report(analysis.Diagnostic{
			Pos:     node.Pos(),
			End:     node.End(),
			Message: fmt.Sprintf("import %q imported as %q but must be %q according to config", path, alias, required),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message:   "Use correct alias",
				TextEdits: findEdits(node, pass.TypesInfo.Uses, path, alias, required),
			}},
		})
	}
}

func findEdits(node ast.Node, uses map[*ast.Ident]types.Object, importPath, original, required string) []analysis.TextEdit {
	// Edit the actual import line.
	result := []analysis.TextEdit{{
		Pos:     node.Pos(),
		End:     node.End(),
		NewText: []byte(required + " " + strconv.Quote(importPath)),
	}}

	// Edit all the uses of the alias in the code.
	for use, pkg := range uses {
		pkgName, ok := pkg.(*types.PkgName)
		if !ok {
			// skip identifiers that aren't pointing at a PkgName.
			continue
		}

		if pkgName.Pos() != node.Pos() {
			// skip identifiers pointing to a different import statement.
			continue
		}

		if original == pkgName.Name() {
			result = append(result, analysis.TextEdit{
				Pos:     use.Pos(),
				End:     use.End(),
				NewText: []byte(required),
			})
		}
	}

	return result
}
