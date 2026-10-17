package parser

import (
	"fmt"
	"slices"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) isStartMarker() bool {
	if !p.stream.AssertType(tokenizer.AT) {
		return false
	}
	tok := p.stream.PeekTokenAt(1)
	if tok.Type != tokenizer.IDENTIFIER {
		return false
	}
	return strings.HasPrefix(tok.Literal, "start")
}

func (p *Parser) isEndMarker() bool {
	if !p.stream.AssertType(tokenizer.AT) {
		return false
	}
	tok := p.stream.PeekTokenAt(1)
	if tok.Type != tokenizer.IDENTIFIER {
		return false
	}
	return strings.HasPrefix(tok.Literal, "end")
}

// isDiagramEndMarker checks if the next token is a diagram end marker.
func (p *Parser) isDiagramEndMarker(tok tokenizer.Token) bool {
	if tok.Type != tokenizer.AT {
		return false
	}
	tok = p.stream.PeekTokenAt(0)
	return strings.HasPrefix(tok.Literal, "end")
}

func (p *Parser) isDiagramBound() bool {
	return p.isStartMarker() || p.isEndMarker()
}

var possibleBounds = []string{
	"uml",
	"gantt",
	"mindmap",
	"def",
}

// readStartBounds reads the diagram bounds of a diagram.
// You must assert that the following tokens are of a diagram bounds start/end marker.
func (p *Parser) readStartBounds() (*ast.DiagramBound, error) {
	atTok, consumed := p.stream.TryConsumeType(tokenizer.AT)
	if !consumed {
		return nil, fmt.Errorf("expected @ at diagram bounds start, got %s", atTok.Type)
	}

	m := p.Mark(atTok)

	tok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
	if !ok {
		return nil, fmt.Errorf("expected identifier at diagram bounds, got %s", tok.Type)
	}

	if !strings.HasPrefix(tok.Literal, "start") &&
		!strings.HasPrefix(tok.Literal, "end") {
		return nil, fmt.Errorf("invalid bounding marker for diagram, expected something that starts with 'start' or 'end'")
	}

	diag := &ast.DiagramBound{
		IsStart: true,
	}

	if typ, found := strings.CutPrefix(tok.Literal, "start"); found {
		if !slices.Contains(possibleBounds, typ) {
			return nil, fmt.Errorf("invalid starting marker for diagram, expected one of %s, got %s", possibleBounds, typ)
		}
		diag.Type = typ
	}

	if !p.stream.AssertType(tokenizer.LPAREN) && !p.stream.AssertType(tokenizer.LBRACE) {
		diag.TrailingName = p.stream.ReadUntilNewline()
		diag.SetMetadata(p.Span(m), nil, nil)
		return diag, nil
	}

	readKvp := func(blockOpener, blockTerminator tokenizer.TokenType) (ast.BoundOption, error) {
		keyTok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
		if !ok {
			return ast.BoundOption{}, fmt.Errorf("expected a key")
		}
		if _, ok := p.stream.TryConsumeType(tokenizer.EQUALS); !ok {
			return ast.BoundOption{}, fmt.Errorf("expected = after key")
		}
		opt := ast.BoundOption{
			Key: keyTok.Literal,
		}

		toks := []tokenizer.Token{}
		depth := 0
	collectorLoop:
		for {
			tok := p.stream.PeekRawTokenAt(0)
			switch tok.Type {
			case blockOpener:
				depth++
			case blockTerminator:
				if depth == 0 {
					break collectorLoop
				}
				depth--
			case tokenizer.COMMA:
				if depth == 0 {
					break collectorLoop
				}
			case tokenizer.EOF, tokenizer.NEWLINE:
				// prevent infinite loops on unclosed input
				break collectorLoop
			}
			toks = append(toks, p.stream.Emit())
		}
		opt.Value = p.stream.SliceInputEnclosingTokens(toks...)
		return opt, nil
	}

	if _, consumed := p.stream.TryConsumeType(tokenizer.LPAREN); consumed {
		seenParams := make(map[string]bool)
		for !p.stream.AssertType(tokenizer.RPAREN) && !p.stream.AssertType(tokenizer.EOF) && !p.stream.AssertType(tokenizer.NEWLINE) {
			param, err := readKvp(tokenizer.LPAREN, tokenizer.RPAREN)
			if err != nil {
				return nil, err
			}
			if seenParams[param.Key] {
				return nil, fmt.Errorf("duplicate key in diagram bounds: %s", param.Key)
			}
			seenParams[param.Key] = true
			if param.Key == "id" {
				diag.ID = param.Value
			}
			diag.Params = append(diag.Params, param)

			if _, ok := p.stream.TryConsumeType(tokenizer.COMMA); !ok {
				break
			}
		}
		if _, ok := p.stream.TryConsumeType(tokenizer.RPAREN); !ok {
			return nil, fmt.Errorf("expected closing parenthesis in diagram bounds")
		}
	}

	if !p.stream.AssertType(tokenizer.LBRACE) {
		diag.TrailingName = p.stream.ReadUntilNewline()
		diag.SetMetadata(p.Span(m), nil, nil)
		return diag, nil
	}

	if _, consumed := p.stream.TryConsumeType(tokenizer.LBRACE); consumed {
		tokens := p.stream.ConsumeUntilType(tokenizer.EOF, tokenizer.NEWLINE, tokenizer.COMMA, tokenizer.RBRACE)
		if p.stream.AssertType(tokenizer.EOF) || p.stream.AssertType(tokenizer.NEWLINE) {
			return nil, fmt.Errorf("unexpected EOF or newline")
		}
		tools := &ast.BoundToolOptions{
			File: p.stream.TokensToString(tokens),
		}

		if p.stream.PeekTokenAt(2).Type != tokenizer.EQUALS {
			p.stream.TryConsumeType(tokenizer.COMMA)
			captionTokens := p.stream.ConsumeUntilType(tokenizer.EOF, tokenizer.NEWLINE, tokenizer.COMMA, tokenizer.RBRACE)
			if p.stream.AssertType(tokenizer.EOF) || p.stream.AssertType(tokenizer.NEWLINE) {
				return nil, fmt.Errorf("unexpected EOF or newline")
			}
			tools.Caption = p.stream.TokensToString(captionTokens)
		}

		seenOpts := make(map[string]bool)
		for p.stream.AssertType(tokenizer.COMMA) {
			p.stream.TryConsumeType(tokenizer.COMMA)
			opt, err := readKvp(tokenizer.LBRACE, tokenizer.RBRACE)
			if err != nil {
				return nil, err
			}
			if seenOpts[opt.Key] {
				return nil, fmt.Errorf("duplicate key in diagram bounds: %s", opt.Key)
			}
			seenOpts[opt.Key] = true
			tools.Options = append(tools.Options, opt)
		}

		if _, ok := p.stream.TryConsumeType(tokenizer.RBRACE); !ok {
			return nil, fmt.Errorf("expected } at end of diagram bounds options")
		}
		diag.Tools = tools
	}

	if p.stream.AssertType(tokenizer.LPAREN) {
		return nil, fmt.Errorf("possibly incorrect order of tools and id")
	}

	if !p.stream.AssertType(tokenizer.NEWLINE) {
		return nil, fmt.Errorf("expected newline after diagram bounds")
	}

	diag.SetMetadata(p.Span(m), nil, nil)
	return diag, nil
}

func (p *Parser) readEndBounds(atTok tokenizer.Token) (*ast.DiagramBound, error) {
	if atTok.Type != tokenizer.AT {
		return nil, fmt.Errorf("expected @ at diagram bounds start, got %s", atTok.Type)
	}

	bnd := &ast.DiagramBound{
		IsStart: false,
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}

	tok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
	if !ok {
		return nil, NewParserErrorf(tok, "expected identifier at diagram bounds, got %s", tok.Type)
	}

	bnd.Type, ok = strings.CutPrefix(tok.Literal, "end")
	if !ok {
		return nil, NewParserErrorf(tok, "invalid ending marker for diagram, expected something that starts with 'end', got %s", tok.Literal)
	}

	return bnd, nil
}
