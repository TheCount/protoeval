package protoeval

import (
	"sync"

	"github.com/google/cel-go/cel"
)

// celAstMap maps CEL code to its AST.
type celAstMap struct {
	// mx protects this structure.
	mx sync.RWMutex

	// asts maps CEL code to its AST.
	asts map[string]*cel.Ast
}

// GetAST retrieves the AST for the specified code from this map.
// If no such AST exists, it is created.
func (cam *celAstMap) GetAST(code string) (*cel.Ast, error) {
	cam.mx.RLock()
	ast, ok := cam.asts[code]
	cam.mx.RUnlock()
	if ok {
		return ast, nil
	}
	initCel()
	newast, iss := commonCelEnv.Compile(code)
	if err := iss.Err(); err != nil {
		return nil, err
	}
	cam.mx.Lock()
	defer cam.mx.Unlock()
	ast, ok = cam.asts[code]
	if ok {
		// someone else was faster, discard newast
		return ast, nil
	}
	cam.asts[code] = newast
	return newast, nil
}

// commonAsts is the common cache of CEL ASTs.
var commonAsts = celAstMap{
  asts: make(map[string]*cel.Ast),
}
