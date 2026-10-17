package parser

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) parseDirective(tok1 tokenizer.Token) (ast.Statement, error) {
	directiveNameTok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
	if !ok {
		return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected directive name")
	}

	if directiveNameTok.Span.Start.Offset != tok1.Span.Start.Offset+1 {
		return nil, NewParserError(directiveNameTok, "Expected directive name right after !")
	}

	if strings.HasPrefix(directiveNameTok.Literal, "include") {
		return p.parseIncludeDirective(directiveNameTok)
	} else {
		return p.parseUnhandledDirective(tok1, directiveNameTok)
	}
}

func (p *Parser) parseIncludeDirective(tok tokenizer.Token) (*ast.IncludeDirective, error) {
	var kind ast.IncludeKind
	switch tok.Literal {
	case "include_many":
		kind = ast.IncludeMany
	case "include_once", "include":
		kind = ast.IncludeOnce
	default:
		return nil, NewParserError(tok, "Unknown include directive")
	}

	dir := &ast.IncludeDirective{
		Kind: kind,
	}

	filePathToks := p.stream.ConsumeUntilType(tokenizer.NEWLINE, tokenizer.EXCLAMATION)
	dir.Path = p.stream.TokensToString(filePathToks)
	if p.stream.AssertType(tokenizer.EXCLAMATION) {
		p.stream.Emit() // consume '!'
		// Id or order must be a single token
		lastDirTok := p.stream.Emit()
		dir.Tag = lastDirTok.Literal
	}

	return dir, nil
}
