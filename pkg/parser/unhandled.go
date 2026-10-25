package parser

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) parseUnhandled(tok tokenizer.Token) (ast.Statement, error) {
	leadingTrivia := p.dumpTrivia()

	kw := FindUnhandledKeyword(tok.Literal)
	if kw == nil {
		return nil, NewParserError(tok, "Unexpected unhandled keyword")
	}

	if kw.Closer != "" {
		return p.consumeUnhandledBlock(tok, *kw, leadingTrivia)
	}
	return p.consumeUnhandledLine(tok, leadingTrivia)
}

func (p *Parser) parseUnhandledDirective(exclTok tokenizer.Token, dirNameTok tokenizer.Token) (ast.Statement, error) {
	leadingTrivia := p.dumpTrivia()

	kw := FindUnhandledKeyword("!" + dirNameTok.Literal)
	if kw != nil && kw.Closer != "" {
		return p.consumeUnhandledBlock(exclTok, *kw, leadingTrivia)
	}
	return p.consumeUnhandledLine(exclTok, leadingTrivia)
}

func (p *Parser) consumeUnhandledLine(startTok tokenizer.Token, leadingTrivia []ast.Comment) (ast.Statement, error) {
	lineToks := p.stream.ConsumeUntilType(tokenizer.NEWLINE)

	endTok := startTok
	if len(lineToks) > 0 {
		endTok = lineToks[len(lineToks)-1]
	}

	text := p.stream.SliceInput(startTok.Span.Start.Offset, endTok.EndOffset())

	return &ast.UnhandledStatement{
		Raw: text,
		BaseNode: ast.BaseNode{
			LeadingTrivia: leadingTrivia,
		},
	}, nil
}

func (p *Parser) consumeUnhandledBlock(startTok tokenizer.Token, kw UnhandledKeyword, leadingTrivia []ast.Comment) (ast.Statement, error) {
	// Consume remainder of opener line
	p.stream.ConsumeUntilType(tokenizer.NEWLINE)
	p.stream.TryConsumeType(tokenizer.NEWLINE)

	depth := 1
	braceDepth := 0
	var endTok tokenizer.Token

	for {
		if p.stream.AssertType(tokenizer.EOF) {
			return nil, NewParserErrorf(startTok, "unterminated block statement for %s", kw.Keyword)
		}
		if p.stream.AssertType(tokenizer.AT) && strings.HasPrefix(strings.ToLower(p.stream.PeekTokenAt(1).Literal), "end") {
			return nil, NewParserErrorf(startTok, "unterminated block statement for %s", kw.Keyword)
		}

		lineToks := p.stream.ConsumeUntilType(tokenizer.NEWLINE)
		if len(lineToks) == 0 {
			p.stream.TryConsumeType(tokenizer.NEWLINE)
			continue
		}

		// Track container brace depth to catch unclosed blocks hitting scope delimiter '}'
		for _, t := range lineToks {
			switch t.Type {
			case tokenizer.LBRACE:
				braceDepth++
			case tokenizer.RBRACE:
				braceDepth--
				if braceDepth < 0 {
					return nil, NewParserErrorf(startTok, "unterminated block statement for %s (hit enclosing scope delimiter)", kw.Keyword)
				}
			}
		}

		if isMatchingCloser(lineToks, kw.Closer) {
			depth--
			if depth == 0 {
				endTok = lineToks[len(lineToks)-1]
				break
			}
		} else if kw.Nestable && isMatchingOpener(lineToks, kw.Keyword) {
			depth++
		}

		p.stream.TryConsumeType(tokenizer.NEWLINE)
	}

	text := p.stream.SliceInput(startTok.Span.Start.Offset, endTok.EndOffset())

	return &ast.UnhandledStatement{
		Raw: text,
		BaseNode: ast.BaseNode{
			LeadingTrivia: leadingTrivia,
		},
	}, nil
}

func isMatchingCloser(lineToks []tokenizer.Token, closer string) bool {
	return tokenizer.MatchTokenLine(lineToks, closer)
}

func isMatchingOpener(lineToks []tokenizer.Token, opener string) bool {
	_, ok := tokenizer.MatchTokenPrefix(lineToks, opener)
	return ok
}
