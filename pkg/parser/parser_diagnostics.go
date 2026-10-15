package parser

import "yur4uwe/pac/pkg/parser/ast"

func (p *Parser) recordDiagnostic(d *ast.Diagnostic) {
	if p.ast == nil {
		p.ast = &ast.Diagram{}
	}
	p.ast.Diagnostics = append(p.ast.Diagnostics, d)
}
