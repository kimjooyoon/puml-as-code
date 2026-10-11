package parser

import (
	"fmt"
	"slices"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) tryReadModifier() (string, error) {
	start := []tokenizer.Token{{Type: tokenizer.LBRACE}}
	end := []tokenizer.Token{{Type: tokenizer.RBRACE}}
	return p.stream.ReadBetween(start, end)
}

func (p *Parser) tryReadStereotype() (string, error) {
	start := []tokenizer.Token{{Type: tokenizer.LANGLE}, {Type: tokenizer.LANGLE}}
	end := []tokenizer.Token{{Type: tokenizer.RANGLE}, {Type: tokenizer.RANGLE}}
	return p.stream.ReadBetween(start, end)
}

func (p *Parser) tryReadGeneric() (string, error) {
	start := []tokenizer.Token{{Type: tokenizer.LANGLE}}
	end := []tokenizer.Token{{Type: tokenizer.RANGLE}}
	return p.stream.ReadBetween(start, end)
}

func (p *Parser) tryReadClassSeparator() (*ast.ClassSeparator, error) {
	var sepChar rune
	sepTok := p.stream.PeekTokenAt(0)
	sepTokType := sepTok.Type
	switch sepTokType {
	case tokenizer.DASH:
		sepChar = '-'
	case tokenizer.DOT:
		sepChar = '.'
	case tokenizer.EQUALS:
		sepChar = '='
	case tokenizer.UNDERSCORE:
		sepChar = '_'
	default:
		return nil, fmt.Errorf("unexpected class separator")
	}

	sep := &ast.ClassSeparator{
		Type: sepChar,
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}

	lastTok := sepTok
	// non-consuming lookahead
	for i := 0; ; i++ {
		lastTok = p.stream.PeekTokenAt(i)
		if lastTok.Type != sepTokType {
			break
		}
		sep.TypeCount++
	}

	if sep.TypeCount < 2 {
		return nil, tokenizer.ErrStartMarkerNotFound
	}

	for i := 0; i < sep.TypeCount; i++ {
		p.stream.Emit() // consume the separator token
	}

	if lastTok.Type == tokenizer.NEWLINE {
		return sep, nil
	}

	var labelToks []tokenizer.Token
	for {
		tok := p.stream.PeekTokenAt(0)
		if tok.Type == tokenizer.NEWLINE || tok.Type == tokenizer.EOF {
			break
		}
		if p.stream.AssertTypeSeq([]tokenizer.TokenType{sepTokType, sepTokType}) {
			break
		}
		labelToks = append(labelToks, p.stream.Emit())
	}

	switch p.stream.PeekTokenAt(0).Type {
	case tokenizer.NEWLINE:
		return nil, fmt.Errorf("unclosed class separator")
	case tokenizer.EOF:
		return nil, tokenizer.ErrUnexpectedEOF
	}

	for {
		if _, ok := p.stream.TryConsumeType(sepTokType); !ok {
			break
		}
		sep.TypeCount++
	}

	if !p.stream.AssertAnyType(tokenizer.NEWLINE, tokenizer.EOF) {
		return nil, fmt.Errorf("unexpected tokens after class separator")
	}
	sep.Label = p.stream.SliceInputEnclosingTokens(labelToks...)
	sep.Type = sepChar
	return sep, nil
}

func (p *Parser) tryReadTags() []string {
	var tags []string
	for p.stream.AssertType(tokenizer.DOLLAR) {
		p.stream.Emit() // consume $
		tagTok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
		if !ok {
			break
		}
		tags = append(tags, tagTok.Literal)
	}
	return tags
}

type Mark struct {
	start tokenizer.Token
}

func (p *Parser) Mark(tok tokenizer.Token) Mark {
	return Mark{
		start: tok,
	}
}

func (p *Parser) Span(m Mark) tokenizer.SourceSpan {
	return tokenizer.SpanEnclosing(m.start, p.stream.LastSemanticToken())
}

func (p *Parser) SpanTo(m Mark, endTok tokenizer.Token) tokenizer.SourceSpan {
	return tokenizer.SpanEnclosing(m.start, endTok)
}
