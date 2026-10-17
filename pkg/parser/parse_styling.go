package parser

import (
	"errors"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) parseSkinparamRoot(tok tokenizer.Token) (ast.Statement, error) {
	return p.parseSkinparamStatement(tok, true)
}

func (p *Parser) parseSkinparamStatement(tok tokenizer.Token, isRoot bool) (ast.Statement, error) {
	mark := p.Mark(tok)
	leadingTrivia := p.stream.DumpCollectedTrivia()

	var keyword string
	var name string

	if isRoot {
		keyword = tok.Literal

		// Check for anonymous block: `skinparam { ... }`
		if !p.stream.AssertType(tokenizer.LBRACE) {
			nameTok := p.stream.Emit()
			if nameTok.Type == tokenizer.NEWLINE || nameTok.Type == tokenizer.EOF {
				return nil, NewParserError(nameTok, "Expected target or parameter after skinparam")
			}
			name = nameTok.Literal
		}
	} else {
		name = tok.Literal
	}

	stereo, err := p.tryReadStereotype()
	if errors.Is(err, tokenizer.ErrUnexpectedEOF) {
		// EOF will be the next token
		return nil, WrapParserError(p.stream.PeekTokenAt(0), err)
	}

	if _, ok := p.stream.TryConsumeType(tokenizer.LBRACE); ok {
		p.stream.EmitCommentToks()
		trailingTrivia := p.stream.DumpCollectedTrivia()
		block := &ast.SkinparamBlock{
			BaseNode: ast.BaseNode{
				TrailingTrivia: trailingTrivia,
			},
			Keyword:    keyword,
			Name:       name,
			Stereotype: stereo,
		}
		err := p.parseSkinparamBlockMembers(block)
		if err != nil {
			return nil, err
		}
		p.stream.EmitCommentToks()
		block.SetMetadata(p.Span(mark), leadingTrivia, p.stream.DumpCollectedTrivia())
		return block, nil
	}

	// read contiguous tokens as a key
	prevTokEnd := p.stream.LastSemanticToken().Span.End.Offset
	for {
		nextTok := p.stream.PeekTokenAt(0)
		if nextTok.Span.Start.Offset != prevTokEnd ||
			nextTok.Type == tokenizer.NEWLINE ||
			nextTok.Type == tokenizer.EOF ||
			nextTok.Type == tokenizer.SEMICOLON {
			break
		}

		switch nextTok.Type {
		case tokenizer.DOT, tokenizer.UNDERSCORE, tokenizer.IDENTIFIER:
			// It's a key.
			nameContinues := p.stream.Emit()
			name += nameContinues.Literal
			prevTokEnd = nextTok.Span.End.Offset
		default:
			return nil, NewParserError(nextTok, "Unexpected token in skinparam block")
		}
	}

	toks := p.stream.ConsumeUntilType(tokenizer.NEWLINE, tokenizer.SEMICOLON)
	value := p.stream.SliceInputEnclosingTokens(toks...)
	p.stream.TryConsumeType(tokenizer.SEMICOLON)
	return &ast.SkinparamSetting{
		Keyword:    keyword,
		Name:       name,
		Value:      value,
		Stereotype: stereo,
		BaseNode: ast.BaseNode{
			LeadingTrivia: leadingTrivia,
		},
	}, nil
}

func (p *Parser) parseSkinparamBlockMembers(block *ast.SkinparamBlock) error {
	for tok := p.stream.Emit(); tok.Type != tokenizer.RBRACE; tok = p.stream.Emit() {
		switch tok.Type {
		case tokenizer.NEWLINE, tokenizer.SEMICOLON:
			continue
		case tokenizer.EOF:
			return NewParserError(tok, "Unexpected EOF in skinparam block")
		}

		memberMark := p.Mark(tok)
		leadingTrivia := p.stream.DumpCollectedTrivia()

		var member ast.Statement
		var err error
		switch tok.Type {
		case tokenizer.EXCLAMATION:
			member, err = p.parseDirective(tok)
		default:
			member, err = p.parseSkinparamStatement(tok, false)
		}
		if err != nil {
			return err
		}
		p.stream.EmitCommentToks()
		member.SetMetadata(p.Span(memberMark), leadingTrivia, p.stream.DumpCollectedTrivia())
		block.Statements = append(block.Statements, member)
	}

	return nil
}

func (p *Parser) isStyleTagEnd() bool {
	return p.stream.AssertSeq([]tokenizer.Token{
		{Type: tokenizer.LANGLE},
		{Type: tokenizer.SLASH},
		{Type: tokenizer.IDENTIFIER, Literal: "style"},
		{Type: tokenizer.RANGLE},
	})
}

func (p *Parser) parseStyleBlock(startTok tokenizer.Token) (ast.Statement, error) {
	p.stream.Emit() // consume 'style'
	p.stream.Emit() // consume '>'

	styleBlock := &ast.StyleBlock{
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}

	for !p.isStyleTagEnd() {
		tok := p.stream.Emit()
		switch tok.Type {
		case tokenizer.NEWLINE, tokenizer.SEMICOLON:
			continue
		case tokenizer.EOF:
			return nil, NewParserError(tok, "Unexpected EOF in style block")
		}

		ruleMark := p.Mark(tok)
		leadingTrivia := p.stream.DumpCollectedTrivia()

		var stmnt ast.Statement
		var err error
		switch tok.Type {
		case tokenizer.EXCLAMATION:
			stmnt, err = p.parseDirective(tok)
		default:
			// Must be a StyleRule (e.g. "root { ... }" or "classDiagram { ... }")
			stmnt, err = p.parseStyleRule(tok)
		}
		if err != nil {
			return nil, err
		}
		p.stream.EmitCommentToks()
		stmnt.SetMetadata(p.Span(ruleMark), leadingTrivia, p.stream.DumpCollectedTrivia())
		styleBlock.Statements = append(styleBlock.Statements, stmnt)
	}

	// Consume '</style>'
	for range 4 {
		p.stream.Emit()
	}

	return styleBlock, nil
}

func (p *Parser) getSelectors(startTok tokenizer.Token) ([]string, error) {
	// LBRACE is canonocal syntax but leave newline for error recovery
	toks := p.stream.ConsumeUntilType(tokenizer.LBRACE, tokenizer.NEWLINE)
	toks = append([]tokenizer.Token{startTok}, toks...)
	headerText := p.stream.SliceInputEnclosingTokens(toks...)

	var selectors []string
	for part := range strings.SplitSeq(headerText, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, NewParserError(toks[0], "Empty selector in style rule")
		}
		selectors = append(selectors, trimmed)
	}
	return selectors, nil
}

func (p *Parser) parseStyleRule(startTok tokenizer.Token) (*ast.StyleRule, error) {
	currentRule := &ast.StyleRule{
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}

	var err error
	currentRule.Selectors, err = p.getSelectors(startTok)
	if err != nil {
		return nil, err
	}

	if _, ok := p.stream.TryConsumeType(tokenizer.LBRACE); !ok {
		return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected opening brace after style rule")
	}

	for !p.isStyleTagEnd() && !p.stream.AssertType(tokenizer.RBRACE) && !p.stream.AssertType(tokenizer.EOF) {

		tok := p.stream.Emit()
		switch tok.Type {
		case tokenizer.NEWLINE, tokenizer.SEMICOLON:
			continue
		case tokenizer.EXCLAMATION:
			dir, err := p.parseDirective(tok)
			if err != nil {
				return nil, err
			}
			currentRule.Statements = append(currentRule.Statements, dir)
			continue
		}

		if p.hasLBraceOnLine() {
			nestedRule, err := p.parseStyleRule(tok)
			if err != nil {
				return nil, err
			}
			currentRule.Statements = append(currentRule.Statements, nestedRule)
			continue
		}

		// By elimination: it is a style declaration
		if tok.Type != tokenizer.IDENTIFIER {
			return nil, NewParserError(tok, "Expected style property name or rule selector")
		}

		declMark := p.Mark(tok)
		p.stream.TryConsumeType(tokenizer.COLON)
		leadingTrivia := p.stream.DumpCollectedTrivia()
		toks := p.stream.ConsumeUntilType(tokenizer.SEMICOLON, tokenizer.NEWLINE)
		if len(toks) == 0 {
			return nil, NewParserError(p.stream.PeekRawTokenAt(0), "Expected value after style declaration")
		}

		val := p.stream.SliceInputEnclosingTokens(toks...)
		p.stream.TryConsumeType(tokenizer.SEMICOLON)
		p.stream.EmitCommentToks()
		declaration := &ast.StyleDeclaration{
			Property: tok.Literal,
			Value:    val,
		}
		declaration.SetMetadata(p.Span(declMark), leadingTrivia, p.stream.DumpCollectedTrivia())
		currentRule.Statements = append(currentRule.Statements, declaration)
	}

	if _, ok := p.stream.TryConsumeType(tokenizer.RBRACE); !ok {
		return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected closing brace '}' after style rule")
	}

	return currentRule, nil
}

func (p *Parser) hasLBraceOnLine() bool {
	for i := 0; ; i++ {
		tok := p.stream.PeekTokenAt(i)
		if tok.Type == tokenizer.LBRACE {
			return true
		}
		if tok.Type == tokenizer.NEWLINE || tok.Type == tokenizer.EOF || tok.Type == tokenizer.RBRACE || tok.Type == tokenizer.SEMICOLON {
			return false
		}
	}
}
