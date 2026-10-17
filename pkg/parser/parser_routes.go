package parser

import (
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) fallbackRoute(
	tok tokenizer.Token,
	stmt ast.Statement,
	err error,
	mark Mark,
	collector *tokenizer.TokenCollector,
	detach func(),
	syncToBlockEnd bool,
) ast.Statement {
	var leadingTrivia []tokenizer.Token
	if err == nil && stmt != nil {
		leadingTrivia = stmt.GetLeadingTrivia()
	} else {
		leadingTrivia = p.stream.DumpCollectedTrivia()
	}
	if err == nil {
		err = NewParserError(tok, "Unexpected token")
	}

	if syncToBlockEnd {
		p.syncToLineOrBlockEnd()
	} else {
		p.syncToLineEnd()
	}

	if collector.Len() == 0 && !p.stream.AssertType(tokenizer.EOF) && !p.isEndMarker() {
		p.stream.Emit()
	}

	detach()
	badToks := collector.Tokens()
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

	p.recordDiagnostic(dgnst)
	return badStmt
}

func (p *Parser) routeDiagramStatement(tok tokenizer.Token) ast.Statement {
	collector := &tokenizer.TokenCollector{}
	collector.Receive(tok)
	detach := p.stream.Attach(collector)
	defer detach()

	mark := p.Mark(tok)

	leadingTrivia := p.stream.DumpCollectedTrivia()

	stmt, err := p.parseDiagramOnlyStatement(tok)
	if err != nil {
		return p.fallbackRoute(tok, stmt, err, mark, collector, detach, false)
	}
	if stmt != nil {
		detach()
		p.stream.EmitCommentToks()
		stmt.SetMetadata(p.Span(mark), leadingTrivia, p.stream.DumpCollectedTrivia())
		return stmt
	}

	stmt, err = p.parseContainerStatement(tok)
	if err != nil {
		return p.fallbackRoute(tok, stmt, err, mark, collector, detach, false)
	}
	if stmt != nil {
		detach()
		p.stream.EmitCommentToks()
		stmt.SetMetadata(p.Span(mark), leadingTrivia, p.stream.DumpCollectedTrivia())
		return stmt
	}

	return p.fallbackRoute(tok, nil, nil, mark, collector, detach, false)
}

func (p *Parser) routeContainerStatement(tok tokenizer.Token) ast.Statement {
	collector := &tokenizer.TokenCollector{}
	collector.Receive(tok)
	detach := p.stream.Attach(collector)
	defer detach()

	mark := p.Mark(tok)

	leadingTrivia := p.stream.DumpCollectedTrivia()

	stmt, err := p.parseContainerStatement(tok)
	if err != nil {
		return p.fallbackRoute(tok, stmt, err, mark, collector, detach, true)
	}
	if stmt != nil {
		detach()
		p.stream.EmitCommentToks()
		stmt.SetMetadata(p.Span(mark), leadingTrivia, p.stream.DumpCollectedTrivia())
		return stmt
	}

	return p.fallbackRoute(tok, stmt, err, mark, collector, detach, true)
}
