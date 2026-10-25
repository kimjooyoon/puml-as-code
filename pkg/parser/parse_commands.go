package parser

import (
	"fmt"
	"strconv"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) parseVisibilityCommand(tok tokenizer.Token) (*ast.VisibilityCommand, error) {
	cmd := &ast.VisibilityCommand{
		Kind: ast.VisibilityCMDUnknown,
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.dumpTrivia(),
		},
	}
	switch keyword.Classify(tok.Literal) {
	case keyword.Hide:
		cmd.Kind = ast.VisibilityCMDHide
	case keyword.Show:
		cmd.Kind = ast.VisibilityCMDShow
	case keyword.Remove:
		cmd.Kind = ast.VisibilityCMDRemove
	case keyword.Restore:
		cmd.Kind = ast.VisibilityCMDRestore
	}
	cmd.Target = p.stream.ReadUntilNewline()
	return cmd, nil
}

func (p *Parser) parseDiagDirection(tok tokenizer.Token) (*ast.DirectionCommand, error) {
	var to string
	switch tok.Literal {
	case "left":
		to = "right"
	case "top":
		to = "bottom"
	}

	expectedSeq := []tokenizer.Token{
		{
			Type:    tokenizer.IDENTIFIER,
			Literal: "to",
		},
		{
			Type:    tokenizer.IDENTIFIER,
			Literal: to,
		},
		{
			Type:    tokenizer.IDENTIFIER,
			Literal: "direction",
		},
	}

	cmd := &ast.DirectionCommand{
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.dumpTrivia(),
		},
	}

	for _, token := range expectedSeq {
		if _, ok := p.stream.TryConsume(token); !ok {
			return nil, NewParserError(token, "Unexpected diagram direction modifier")
		}
	}
	switch tok.Literal {
	case "left":
		cmd.Direction = ast.LeftToRightDirection
	case "top":
		cmd.Direction = ast.TopToBottomDirection
	}
	if !p.stream.AssertType(tokenizer.NEWLINE) {
		return nil, NewParserError(p.stream.PeekTokenAt(0), "Unexpected tokens after direction command")
	}
	return cmd, nil
}

func (p *Parser) extractScaleTokens() (lhs, sep, rhs, unit string, err error) {
	return "", "", "", "", nil
}

func isDecimalInt(s string) bool {
	return strings.ContainsAny(s, ".exob")
}

func isDecimalFloat(s string) bool {
	return strings.ContainsAny(s, "exob")
}

func (p *Parser) parseScaleTrial(startTok tokenizer.Token) (*ast.ScaleCommand, error) {
	cmd := &ast.ScaleCommand{
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.dumpTrivia(),
		},
	}
	if _, ok := p.stream.TryConsume(tokenizer.Token{Type: tokenizer.IDENTIFIER, Literal: "max"}); ok {
		cmd.IsMax = true
	}

	op1, sep, op2, unit, err := p.extractScaleTokens()
	if err != nil {
		return nil, err
	}

	// Step 2: Validate against State Machine rules
	isOp1Int := isDecimalInt(op1)
	isOp1Dec := isDecimalFloat(op1)

	if !isOp1Int && !isOp1Dec {
		return nil, NewParserError(startTok, "Expected number after scale")
	}

	// Max constraints
	if cmd.IsMax {
		if isOp1Dec {
			return nil, NewParserError(startTok, "Cannot use decimals with 'max'")
		}
		if sep == "/" {
			return nil, NewParserError(startTok, "Cannot use fractions with 'max'")
		}
		if sep == "" && unit == "" {
			return nil, NewParserError(startTok, "Cannot use numbers with 'max' without a unit or box")
		}
	}

	// Decimal constraints
	if isOp1Dec && sep != "" {
		return nil, NewParserError(startTok, "Cannot use decimals with separators ('*', 'x', '/')")
	}

	// Binary separator constraints (*, x, /)
	if sep != "" {
		if unit != "" {
			return nil, NewParserError(startTok, "Cannot specify unit with a box or fraction")
		}
		if !isDecimalInt(op2) {
			return nil, NewParserError(startTok, "Expected integer after separator")
		}
	}

	// Step 3: Populate AST
	if sep == "." { // e.g. "1.5" factor
		parts := strings.Split(op1, ".")
		cmd.Lhs, cmd.Sep, cmd.Rhs = parts[0], ".", parts[1]
	} else {
		cmd.Lhs, cmd.Sep, cmd.Rhs, cmd.Unit = op1, sep, op2, unit
	}

	return cmd, nil
}

func (p *Parser) parseScale(startTok tokenizer.Token) (*ast.ScaleCommand, error) {
	cmd := &ast.ScaleCommand{
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.dumpTrivia(),
		},
	}
	if _, ok := p.stream.TryConsume(tokenizer.Token{Type: tokenizer.IDENTIFIER, Literal: "max"}); ok {
		cmd.IsMax = true
	}

	divideSingleTok := func(lit, sep string) error {
		if !strings.Contains(lit, sep) {
			return fmt.Errorf("expected number after scale")
		}
		parts := strings.Split(lit, sep)
		if len(parts) != 2 {
			return fmt.Errorf("expected two parts after '%s'", sep)
		}
		_, errW := strconv.Atoi(parts[0])
		_, errH := strconv.Atoi(parts[1])
		if errW != nil || errH != nil {
			return fmt.Errorf("expected width and height to be integers")
		}
		cmd.Lhs = parts[0]
		cmd.Rhs = parts[1]
		cmd.Sep = sep
		return nil
	}
	var isInt, isFloat bool

	tok := p.stream.Emit()
	switch tok.Type {
	case tokenizer.NUMBER:
		if strings.ContainsAny(tok.Literal, "exob") {
			return nil, NewParserError(tok, "Cannot use non-decimal or scientific integer for scale")
		}
	case tokenizer.DOT:
		// .5 case
		numTok, ok := p.stream.TryConsumeType(tokenizer.NUMBER)
		if !ok {
			return nil, NewParserError(numTok, "Expected number after scale leading dot")
		}
		if strings.ContainsAny(numTok.Literal, ".exob") {
			return nil, NewParserError(numTok, "Cannot use non-decimal integer for shorthand float")
		}
		cmd.Lhs = "." + numTok.Literal
		if !p.stream.AssertAnyType(tokenizer.NEWLINE, tokenizer.EOF) {
			return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected scale to end after fractional scale")
		}
		if cmd.IsMax {
			return nil, NewParserError(tok, "Cannot use fractions with 'max'")
		}
		return cmd, nil
	case tokenizer.IDENTIFIER:
		// Try to handle 200x300 which is tokenized as an IDENTIFIER
		if strings.Contains(tok.Literal, "x") && !strings.HasSuffix(tok.Literal, "x") && !strings.HasPrefix(tok.Literal, "x") {
			if err := divideSingleTok(tok.Literal, "x"); err != nil {
				return nil, WrapParserError(tok, err)
			}
			return cmd, nil
		}

		if before, ok := strings.CutSuffix(tok.Literal, "x"); ok {
			widthStr := before
			if widthStr == "" || strings.ContainsAny(widthStr, ".exob") {
				return nil, NewParserError(tok, "Cannot use non-decimal integer for width of a box")
			}
			if _, err := strconv.Atoi(widthStr); err != nil {
				return nil, NewParserError(tok, "Expected width in a box to be an integer")
			}

			heightTok, ok := p.stream.TryConsumeType(tokenizer.NUMBER)
			if !ok {
				return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected height after 'x' in a box")
			}
			if strings.ContainsAny(heightTok.Literal, ".exob") {
				return nil, NewParserError(heightTok, "Cannot use non-decimal integer for height of a box")
			}
			if _, err := strconv.Atoi(heightTok.Literal); err != nil {
				return nil, NewParserError(heightTok, "Invalid height for the box")
			}

			cmd.Lhs = widthStr
			cmd.Sep = "x"
			cmd.Rhs = heightTok.Literal
			return cmd, nil
		}

		return nil, NewParserError(tok, "Expected number after scale")
	default:
		return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected number after scale")
	}

	valueLit := tok.Literal
	valueTok := tok
	if valueFloat, err := strconv.ParseFloat(valueLit, 64); err == nil {
		isFloat = true
		i := int(valueFloat)
		if float64(i) == valueFloat {
			isInt = true
		}
	}

	setBox := func(sep string) error {
		if !isInt {
			return NewParserError(tok, "Expected width in a box to be an integer")
		}
		if strings.ContainsAny(valueLit, ".exob") {
			return NewParserError(valueTok, "Cannot use non-decimal integer for width of a box")
		}
		cmd.Lhs = valueLit
		cmd.Sep = sep
		heightTok, ok := p.stream.TryConsumeType(tokenizer.NUMBER)
		if !ok {
			return NewParserErrorf(p.stream.PeekTokenAt(0), "Expected height after '%s' in a box", sep)
		}
		if strings.ContainsAny(heightTok.Literal, ".exob") {
			return NewParserError(heightTok, "Cannot use non-decimal integer for height of box")
		}
		if _, err := strconv.Atoi(heightTok.Literal); err != nil {
			return NewParserError(heightTok, "Invalid height for the box")
		}
		cmd.Rhs = heightTok.Literal
		return nil
	}

	tok = p.stream.Emit()
	switch tok.Type {
	case tokenizer.IDENTIFIER:
		switch tok.Literal {
		case "width":
			if strings.ContainsAny(valueLit, "exob") {
				return nil, NewParserError(valueTok, "Cannot use non-decimal or scientific integer for width of a box")
			}
			cmd.Lhs = valueLit
			cmd.Unit = tok.Literal
		case "height":
			if strings.ContainsAny(valueLit, "exob") {
				return nil, NewParserError(valueTok, "Cannot use non-decimal or scientific integer for height of a box")
			}
			cmd.Lhs = valueLit
			cmd.Unit = tok.Literal
		case "x":
			if err := setBox("x"); err != nil {
				return nil, err
			}
		default:
			after, ok := strings.CutPrefix(tok.Literal, "x")
			if !ok {
				return nil, NewParserErrorf(tok, "Unexpected identifier: %s", tok.Literal)
			}
			heightStr := after
			if heightStr == "" || strings.ContainsAny(heightStr, ".exob") {
				return nil, NewParserError(tok, "Cannot use non-decimal integer for height of a box")
			}
			if _, err := strconv.Atoi(heightStr); err != nil {
				return nil, NewParserError(tok, "Invalid height for the box")
			}
			if !isInt || strings.ContainsAny(valueLit, ".exob") {
				return nil, NewParserError(valueTok, "Expected width in a box to be an integer")
			}

			cmd.Lhs = valueLit
			cmd.Sep = "x"
			cmd.Rhs = heightStr
		}
	case tokenizer.ASTERISK:
		if err := setBox("*"); err != nil {
			return nil, err
		}
	case tokenizer.SLASH:
		if err := setBox("/"); err != nil {
			return nil, err
		}
	case tokenizer.NEWLINE, tokenizer.EOF:
		// EOF handling is a special case for a test case
		// valueLit is either a bare integer or a float
		if !isFloat {
			return nil, NewParserError(tok, "Expected number after scale")
		}

		if cmd.IsMax {
			return nil, NewParserError(tok, "Cannot use numbers with 'max' without a unit (width/height) or a box")
		}

		if isFloat && !isInt {
			if err := divideSingleTok(valueLit, "."); err != nil {
				return nil, WrapParserError(tok, err)
			}
		} else {
			cmd.Lhs = valueLit
		}
		return cmd, nil
	default:
		return nil, NewParserErrorf(tok, "Unexpected token: %s(%s)", tok.Type.String(), tok.Literal)
	}

	if !p.stream.AssertAnyType(tokenizer.NEWLINE, tokenizer.EOF) {
		return nil, NewParserError(p.stream.PeekTokenAt(0), "Unexpected tokens after scale command")
	}
	return cmd, nil
}

func (p *Parser) parseSetDirective(startTok tokenizer.Token) (ast.Statement, error) {
	leadingTrivia := p.dumpTrivia()
	keyTok := p.stream.Emit() // consume "separator"

	toks := p.stream.ConsumeUntilType(tokenizer.NEWLINE)
	var sb strings.Builder
	for _, t := range toks {
		sb.WriteString(t.Literal)
	}
	directiveVal := sb.String()
	if keyTok.Literal == "separator" {
		if directiveVal == "none" {
			p.stream.PackageSeparator = ""
		} else {
			p.stream.PackageSeparator = directiveVal
		}
	}

	return &ast.SetCommand{
		Key:   keyTok.Literal,
		Value: directiveVal,
		BaseNode: ast.BaseNode{
			LeadingTrivia: leadingTrivia,
		},
	}, nil
}

func parseLayoutAlignmentToken(tok tokenizer.Token, blockKind ast.TextBlockKind) (horiz string, vert string, isAlign bool) {
	switch strings.ToLower(tok.Literal) {
	case "left", "right", "center":
		return strings.ToLower(tok.Literal), "", true
	case "top", "bottom":
		if blockKind != ast.BlockLegend {
			return "", "", false
		}
		return "", strings.ToLower(tok.Literal), true
	default:
		return "", "", false
	}
}

func (p *Parser) parseLayoutStatement(kwTok tokenizer.Token, prefixAlignment *tokenizer.Token) (ast.Statement, error) {
	blockKind := mapKwTokToTextBlockKind(keyword.Classify(kwTok.Literal))
	leadingTrivia := p.dumpTrivia()

	block := &ast.TextBlock{
		Kind: blockKind,
		BaseNode: ast.BaseNode{
			LeadingTrivia: leadingTrivia,
		},
	}

	// Process prefix alignment if provided (e.g. "left header", "center footer")
	if prefixAlignment != nil {
		if blockKind == ast.BlockTitle {
			return nil, NewParserError(*prefixAlignment, "Title alignment not supported")
		}
		h, v, isAlign := parseLayoutAlignmentToken(*prefixAlignment, blockKind)
		if !isAlign {
			if strings.EqualFold(prefixAlignment.Literal, "top") || strings.EqualFold(prefixAlignment.Literal, "bottom") {
				return nil, NewParserError(*prefixAlignment, "Vertical alignment only supported for legend")
			}
			return nil, NewParserError(*prefixAlignment, "Invalid alignment")
		}
		block.HorizontalAlignment = h
		block.VerticalAlignment = v
	}

	// Consume any trailing alignment modifiers on the opener line (e.g. "legend top left", "header center")
	for !p.stream.AssertAnyType(tokenizer.NEWLINE, tokenizer.EOF) {
		peekTok := p.stream.PeekTokenAt(0)
		h, v, isAlign := parseLayoutAlignmentToken(peekTok, blockKind)
		if !isAlign {
			break
		}
		if blockKind == ast.BlockTitle {
			break // For title, any tokens on the same line are title text
		}
		if h != "" {
			if block.HorizontalAlignment != "" {
				return nil, NewParserError(peekTok, "Horizontal alignment already set")
			}
			block.HorizontalAlignment = h
			p.stream.Emit()
		} else if v != "" {
			if block.VerticalAlignment != "" {
				return nil, NewParserError(peekTok, "Vertical alignment already set")
			}
			if blockKind != ast.BlockLegend {
				return nil, NewParserError(peekTok, "Vertical alignment only supported for legend")
			}
			block.VerticalAlignment = v
			p.stream.Emit()
		}
	}

	lineContentToks := p.stream.ConsumeUntilType(tokenizer.NEWLINE)

	var body string
	if len(lineContentToks) == 0 {
		// --- MULTI-LINE BLOCK FORM ---
		var err error
		body, err = p.stream.ConsumeTextBlock("end", kwTok.Literal)
		if err != nil {
			return nil, NewParserErrorf(kwTok, "unterminated block statement for %s", kwTok.Literal)
		}
	} else {
		// --- SINGLE-LINE INLINE FORM ---
		body = p.stream.SliceInputEnclosingTokens(lineContentToks...)
	}

	block.Text = body
	if blockKind == ast.BlockTitle {
		p.ast.Title = block.Text
	}
	return block, nil
}
