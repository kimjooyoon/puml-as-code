package parser

import (
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) fallbackRoute(tok tokenizer.Token, stmt ast.Statement, err error, mark Mark) ast.Statement {
	var leadingTrivia []tokenizer.Token
	if stmt == nil {
		leadingTrivia = p.stream.DumpCollectedTrivia()
	} else {
		leadingTrivia = stmt.GetLeadingTrivia()
	}
	if err == nil {
		err = NewParserError(tok, "Unexpected token")
	}

	badToks := []tokenizer.Token{tok}
	badToks = append(badToks, p.syncToLineOrBlockEnd()...)
	span := p.Span(mark)

	dgnst := &ast.Diagnostic{
		ID:      len(p.ast.Diagnostics),
		Message: err.Error(),
		Span:    span,
		Err:     err,
	}

	p.stream.EmitCommentToks()
	badStmt := &ast.BadStatement{
		Diagnostic: dgnst,
		Tokens:     badToks,
		BaseNode: ast.BaseNode{
			NodeSpan:       span,
			LeadingTrivia:  leadingTrivia,
			TrailingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}

	dgnst.Node = badStmt

	p.ast.Diagnostics = append(p.ast.Diagnostics, dgnst)
	return badStmt
}

func (p *Parser) routeDiagramStatement(tok tokenizer.Token) ast.Statement {
	mark := p.Mark(tok)

	stmt, err := p.parseDiagramOnlyStatement(tok)
	if err != nil {
		return p.fallbackRoute(tok, stmt, err, mark)
	}
	if stmt != nil {
		p.stream.EmitCommentToks()
		stmt = ast.WithMetadata(stmt, p.Span(mark), p.stream.DumpCollectedTrivia())
		return stmt
	}

	stmt, err = p.parseContainerStatement(tok)
	if err != nil {
		return p.fallbackRoute(tok, stmt, err, mark)
	}
	if stmt != nil {
		p.stream.EmitCommentToks()
		stmt = ast.WithMetadata(stmt, p.Span(mark), p.stream.DumpCollectedTrivia())
		return stmt
	}

	return p.fallbackRoute(tok, nil, nil, mark)
}

func (p *Parser) routeContainerStatement(tok tokenizer.Token) ast.Statement {
	mark := p.Mark(tok)

	stmt, err := p.parseContainerStatement(tok)
	if err != nil {
		return p.fallbackRoute(tok, stmt, err, mark)
	}
	if stmt != nil {
		p.stream.EmitCommentToks()
		stmt = ast.WithMetadata(stmt, p.Span(mark), p.stream.DumpCollectedTrivia())
		return stmt
	}

	return p.fallbackRoute(tok, stmt, err, mark)
}
