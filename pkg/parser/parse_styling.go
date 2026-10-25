package parser

import (
	"errors"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) parseSkinparamRoot(tok tokenizer.Token) (ast.Statement, error) {
	leadingTrivia := p.dumpTrivia()
	m := p.Mark(tok)
	skinparam, err := p.parseSkinparamStatement(tok, true)
	if err != nil {
		return nil, err
	}
	p.stream.EmitCommentToks()
	skinparam.SetMetadata(p.Span(m), leadingTrivia, p.dumpTrivia())
	return skinparam, err
}

func (p *Parser) parseSkinparamTarget(firstTok tokenizer.Token) ([]tokenizer.Token, *ast.Stereotype, error) {
	var tokens []tokenizer.Token
	var stereo *ast.Stereotype

	switch firstTok.Type {
	case tokenizer.DOT, tokenizer.UNDERSCORE, tokenizer.DASH, tokenizer.IDENTIFIER,
		tokenizer.NUMBER:
		tokens = append(tokens, firstTok)
	default:
		return nil, nil, NewParserError(firstTok, "expected valid skinparam target name")
	}
	prevTokEnd := firstTok.EndOffset()

	// Collect contiguous key tokens (e.g. "class.BackgroundColor")
	for {
		nextTok := p.stream.PeekTokenAt(0)
		if nextTok.Span.Start.Offset != prevTokEnd {
			break
		}
		switch nextTok.Type {
		case tokenizer.NEWLINE, tokenizer.SEMICOLON, tokenizer.EOF, tokenizer.LBRACE:
			return tokens, stereo, nil
		case tokenizer.DOT, tokenizer.UNDERSCORE, tokenizer.DASH, tokenizer.IDENTIFIER, tokenizer.NUMBER:
			tokens = append(tokens, p.stream.Emit())
			prevTokEnd = nextTok.EndOffset()
		case tokenizer.LANGLE:
			if stereo != nil {
				return tokens, stereo, NewParserError(nextTok, "multiple stereotypes")
			}
			var err error
			stereo, err = p.tryParseStereotype()
			if errors.Is(err, tokenizer.ErrStartMarkerNotFound) {
				return tokens, stereo, NewParserError(nextTok, "unexpected token in stereotype target declaration")
			}
			if err != nil {
				return tokens, stereo, err
			}
			prevTokEnd = stereo.FullSpan.End.Offset
		default:
			return tokens, stereo, NewParserError(nextTok, "unexpected token in stereotype target declaration")
		}
	}

	if stereo != nil && p.stream.AssertTypeSeq(tokenizer.LANGLE, tokenizer.LANGLE) {
		return tokens, stereo, NewParserError(p.stream.PeekTokenAt(0), "multiple stereotypes")
	}
	if stereo == nil {
		var err error
		stereo, err = p.tryParseStereotype()
		if err != nil && !errors.Is(err, tokenizer.ErrStartMarkerNotFound) {
			return tokens, nil, err
		}
	}

	return tokens, stereo, nil
}

func (p *Parser) parseSkinparamStatement(tok tokenizer.Token, isRoot bool) (ast.Statement, error) {
	var keyword string
	var targetTokens []tokenizer.Token
	var stereo *ast.Stereotype
	var err error
	var nameStartTok tokenizer.Token

	if isRoot {
		keyword = tok.Literal

		// Check for anonymous block: `skinparam { ... }`
		if _, ok := p.stream.TryConsumeType(tokenizer.LBRACE); ok {
			p.stream.EmitCommentToks()
			trailingTrivia := p.dumpTrivia()
			block := &ast.SkinparamBlock{
				BaseNode: ast.BaseNode{
					TrailingTrivia: trailingTrivia,
				},
				Keyword: keyword,
			}
			err := p.parseSkinparamBlockMembers(block)
			return block, err
		}

		nameStartTok = p.stream.Emit()
		if nameStartTok.Type == tokenizer.NEWLINE || nameStartTok.Type == tokenizer.EOF {
			return nil, NewParserError(nameStartTok, "Expected target or parameter after skinparam")
		}
	} else {
		nameStartTok = tok
	}

	targetTokens, stereo, err = p.parseSkinparamTarget(nameStartTok)
	if err != nil {
		return nil, err
	}

	if _, ok := p.stream.TryConsumeType(tokenizer.LBRACE); ok {
		p.stream.EmitCommentToks()
		trailingTrivia := p.dumpTrivia()
		block := &ast.SkinparamBlock{
			BaseNode: ast.BaseNode{
				TrailingTrivia: trailingTrivia,
			},
			Keyword:    keyword,
			Name:       ast.IdentFromTokens(targetTokens...),
			Stereotype: stereo,
		}
		err := p.parseSkinparamBlockMembers(block)
		return block, err
	}

	toks := p.stream.ConsumeUntilType(tokenizer.NEWLINE, tokenizer.SEMICOLON)
	value := p.stream.SliceInputEnclosingTokens(toks...)
	p.stream.TryConsumeType(tokenizer.SEMICOLON)
	return &ast.SkinparamSetting{
		Keyword:    keyword,
		Name:       ast.IdentFromTokens(targetTokens...),
		Value:      value,
		Stereotype: stereo,
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
		leadingTrivia := p.dumpTrivia()

		var member ast.Statement
		var err error
		switch tok.Type {
		case tokenizer.EXCLAMATION:
			member, err = p.parseDirective(tok)
		default:
			member, err = p.parseSkinparamStatement(tok, false)
		}
		if err != nil {
			p.recordDiagnostic(&ast.Diagnostic{
				ID:      len(p.ast.Diagnostics),
				Message: err.Error(),
				Span:    p.Span(memberMark),
				Err:     err,
			})
			continue
		}
		p.stream.EmitCommentToks()
		member.SetMetadata(p.Span(memberMark), leadingTrivia, p.dumpTrivia())
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
			LeadingTrivia: p.dumpTrivia(),
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
		leadingTrivia := p.dumpTrivia()

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
		stmnt.SetMetadata(p.Span(ruleMark), leadingTrivia, p.dumpTrivia())
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
			LeadingTrivia: p.dumpTrivia(),
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
		leadingTrivia := p.dumpTrivia()
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
		declaration.SetMetadata(p.Span(declMark), leadingTrivia, p.dumpTrivia())
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
