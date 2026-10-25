package parser

import (
	"errors"
	"fmt"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) tryReadModifier() (string, error) {
	start := []tokenizer.TokenType{tokenizer.LBRACE}
	end := []tokenizer.TokenType{tokenizer.RBRACE}
	toks, _, err := p.stream.ReadTokensBetween(start, end)
	if err != nil {
		return "", err
	}
	return p.stream.SliceInputEnclosingTokens(toks...), nil
}

func (p *Parser) tryParseStereotype() (*ast.Stereotype, error) {
	start := []tokenizer.TokenType{tokenizer.LANGLE, tokenizer.LANGLE}
	end := []tokenizer.TokenType{tokenizer.RANGLE, tokenizer.RANGLE}
	toks, span, err := p.stream.ReadTokensBetween(start, end)
	if err != nil {
		return nil, err
	}

	ident := ast.MergeTokensIntoIdent(toks...)

	stereo := &ast.Stereotype{
		Value:    ident,
		FullSpan: span,
	}

	// Check for Spot Stereotype
	if len(toks) < 3 || toks[0].Type != tokenizer.LPAREN {
		return stereo, nil
	}

	openingParenTok := toks[0]
	var closingParenTok tokenizer.Token

	// We discard the opening and closing parentheses
	// to get the closest syntax to the raw identifier
	spotToks := make([]tokenizer.Token, 0, len(toks))
	nonSpotToks := make([]tokenizer.Token, 0, len(toks))
	inSpot := true
	for i := 1; i < len(toks); i++ {
		if toks[i].Type == tokenizer.RPAREN {
			closingParenTok = toks[i]
			inSpot = false
			continue
		}
		if inSpot {
			spotToks = append(spotToks, toks[i])
		} else {
			nonSpotToks = append(nonSpotToks, toks[i])
		}
	}

	// Must have encountered a closing ')' and at least one spot token
	if inSpot || len(spotToks) == 0 {
		return stereo, nil
	}

	newValue := ast.MergeTokensIntoIdent(nonSpotToks...)
	spotSpan := tokenizer.SourceSpan{
		Start: openingParenTok.Span.Start,
		End:   closingParenTok.Span.End,
	}

	spotRuneTok := spotToks[0]
	spotRunes := []rune(spotRuneTok.Literal)
	if spotRuneTok.Type != tokenizer.IDENTIFIER || len(spotRunes) != 1 {
		return stereo, nil
	}

	spotRune := spotRunes[0]

	if len(spotToks) == 1 {
		stereo.Value = newValue
		stereo.SpotSpan = spotSpan
		stereo.SpotRune = spotRune
		return stereo, nil
	}

	if spotToks[1].Type != tokenizer.COMMA {
		return stereo, nil
	}

	color, err := parseColor(spotToks[2:])
	if err != nil {
		return stereo, WrapParserError(spotToks[2], err)
	}

	spotStereo := &ast.Stereotype{
		Value:     newValue,
		SpotRune:  rune(spotRune),
		SpotSpan:  spotSpan,
		SpotColor: color,
		FullSpan:  stereo.FullSpan,
	}

	return spotStereo, nil
}

func parseColor(toks []tokenizer.Token) (string, error) {
	if len(toks) == 0 {
		return "", errors.New("expected color, received no tokens")
	}

	var sb strings.Builder
	for _, tok := range toks {
		sb.WriteString(tok.Literal)
	}
	return sb.String(), nil
}

func (p *Parser) tryReadStereotype() (string, error) {
	stereo, err := p.tryParseStereotype()
	if err != nil {
		return "", err
	}
	return stereo.Value.Literal, nil
}

func (p *Parser) tryReadGeneric() (string, error) {
	start := []tokenizer.TokenType{tokenizer.LANGLE}
	end := []tokenizer.TokenType{tokenizer.RANGLE}
	toks, _, err := p.stream.ReadTokensBetween(start, end)
	if err != nil {
		return "", err
	}
	return p.stream.SliceInputEnclosingTokens(toks...), nil
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
			LeadingTrivia: p.dumpTrivia(),
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
		if p.stream.AssertTypeSeq(sepTokType, sepTokType) {
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
