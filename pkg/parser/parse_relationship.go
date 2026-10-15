package parser

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) mapTokenToDirection(tok tokenizer.Token) ast.DirectionKind {
	switch tok.Literal {
	case "left":
		return ast.DirectionLeft
	case "right":
		return ast.DirectionRight
	case "top":
		return ast.DirectionTop
	case "bottom":
		return ast.DirectionBottom
	default:
		return ast.DirectionUnknown
	}
}

func (p *Parser) parseRelationship(firstTargetTok tokenizer.Token) (*ast.Relationship, error) {
	// Entry token is supposedly the first identifier
	var err error
	rel := &ast.Relationship{
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}
	rel.LHS, err = p.parseTargetRef(firstTargetTok)
	if err != nil {
		return nil, err
	}

	if multTok, ok := p.stream.TryConsumeType(tokenizer.STRING); ok {
		rel.MultLHS, err = ast.ParseCardinality(multTok.Literal)
		if err != nil {
			return nil, WrapParserError(multTok, err)
		}
	}

	p.parseArrowTokens(rel)

	if multTok, ok := p.stream.TryConsumeType(tokenizer.STRING); ok {
		rel.MultRHS, err = ast.ParseCardinality(multTok.Literal)
		if err != nil {
			return nil, WrapParserError(multTok, err)
		}
	}

	if !p.stream.AssertAnyType(tokenizer.IDENTIFIER, tokenizer.STRING) {
		return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected identifier or string after relationship")
	}
	rel.RHS, err = p.parseTargetRef(p.stream.Emit())
	if err != nil {
		return nil, err
	}

	endingToken := p.stream.PeekTokenAt(0)
	switch endingToken.Type {
	case tokenizer.NEWLINE, tokenizer.EOF, tokenizer.RBRACE:
		// Do not consume newline, EOF, or RBRACE so outer loop/caller handles delimiter
		break
	case tokenizer.COLON:
		p.stream.Emit()
		rel.Label = p.stream.ReadUntilNewline()
	default:
		return nil, NewParserError(endingToken, "Expected newline or colon after relationship")
	}

	return rel, nil
}

func (p *Parser) parseArrowTokens(rel *ast.Relationship) error {
	var sawDirection, sawAttrs, isLolipop bool
	tok := p.stream.Emit()
	switch tok.Type {
	case tokenizer.LANGLE:
		if pipeTok, ok := p.stream.TryConsumeType(tokenizer.PIPE); ok {
			tok = pipeTok
		}
		fallthrough
	case tokenizer.RBRACE:
		rel.LArrow = rune(tok.Literal[0])
	// lolipop interface
	case tokenizer.LPAREN:
		p.stream.MustConsumeType(tokenizer.RPAREN)
		isLolipop = true
		rel.LArrow = rune(tok.Literal[0])
	// position INDEPENDENT start and end tokens
	case tokenizer.IDENTIFIER:
		// special case for 'x' and 'o' in relationship
		if tok.Literal != "x" && tok.Literal != "o" {
			return NewParserError(tok, "Unexpected identifier in relationship definition")
		}
		fallthrough
	case tokenizer.HASH, tokenizer.ASTERISK, tokenizer.PLUS, tokenizer.CARET:
		rel.LArrow = rune(tok.Literal[0])
	case tokenizer.DOT, tokenizer.DASH: // so that encountering them doesn't cause an error
	default:
		return NewParserError(tok, "Unexpected token at the start of relationship definition")
	}

	if tok.Type != tokenizer.DOT && tok.Type != tokenizer.DASH {
		tok = p.stream.Emit() // consume the asserted start token, if any
	}

	bodyTokType := tok.Type
	var oppositeBodyTokType tokenizer.TokenType
	switch bodyTokType {
	case tokenizer.DOT:
		rel.BodyCount = 1
		oppositeBodyTokType = tokenizer.DASH
		switch rel.LArrow {
		case '<':
			rel.TypeLHS = ast.RelationDependency
		case '|':
			rel.TypeLHS = ast.RelationRealization
		}
	case tokenizer.DASH:
		rel.BodyCount = 1
		oppositeBodyTokType = tokenizer.DOT
		switch rel.LArrow {
		case '<':
			rel.TypeLHS = ast.RelationAssociation
		case 'o':
			rel.TypeLHS = ast.RelationAggregation
		case '*':
			rel.TypeLHS = ast.RelationComposition
		case '|':
			rel.TypeLHS = ast.RelationInheritance
		}
	default:
		return NewParserError(tok, "Unexpected token as the relationship body")
	}
	var ok bool
	rel.Body = rune(tok.Literal[0])
	for tok.Type != tokenizer.EOF && tok.Type != tokenizer.NEWLINE {
		if tok, ok = p.stream.TryConsumeType(bodyTokType); ok {
			rel.BodyCount++
			continue
		} else if tok, ok = p.stream.TryConsumeType(oppositeBodyTokType); ok {
			// Simply convenient error message
			return NewParserError(tok, "Different body type runes in relationship")
		}
		if !p.stream.AssertType(tokenizer.LBRACKET) && !p.stream.AssertKW(keyword.Direction) {
			break
		} else if sawAttrs || sawDirection {
			return NewParserError(tok, "Cannot separate direction and attributes with a body token")
		}

		if isLolipop {
			return NewParserError(tok, "Lolipop interface cannot contain attributes or direction")
		}

		// We check these cases in order to be able to assert this squence:
		// -[attrs]->
		//    or
		// -dir->
		//    or
		// -[attrs]dir-> / -dir[attrs]->
		for range 2 {
			if !sawDirection {
				if tok, ok := p.stream.TryConsumeKW(keyword.Direction); ok {
					sawDirection = true
					var dir ast.DirectionKind
					switch tok.Literal {
					case "left", "l", "le":
						dir = ast.DirectionLeft
					case "right", "r", "ri":
						dir = ast.DirectionRight
					case "up", "u":
						dir = ast.DirectionTop
					case "down", "d", "do":
						dir = ast.DirectionBottom
					default:
						return NewParserError(tok, "Unexpected direction in relationship")
					}
					rel.Direction = dir
				}
			}
			if !sawAttrs {
				if _, ok := p.stream.TryConsumeType(tokenizer.LBRACKET); ok {
					sawAttrs = true
					// We have matched attribute container start
					// after the body, now - parse the attributes
					var attrSB strings.Builder
					for tok = p.stream.Emit(); tok.Type != tokenizer.RBRACKET; tok = p.stream.Emit() {
						switch tok.Type {
						case tokenizer.EOF, tokenizer.NEWLINE:
							return NewParserError(tok, "Unexpected break in relationship attribute container")
						case tokenizer.COMMA:
							if attrSB.Len() == 0 {
								return NewParserError(tok, "Unexpected comma in relationship attribute container")
							}
							rel.Attrs = append(rel.Attrs, attrSB.String())
							attrSB.Reset()
							continue // actually useless, added for readability
						default:
							attrSB.WriteString(tok.Literal)
						}
					}
					if attrSB.Len() > 0 {
						rel.Attrs = append(rel.Attrs, attrSB.String())
					}
				}
			}
		}
		// If none matched, we mustn't consume trailing arrow body rune
		if !sawAttrs && !sawDirection {
			continue
		}

		// consume trailing arrow body rune
		if tok, ok = p.stream.TryConsumeType(bodyTokType); !ok {
			return NewParserError(tok, "Unexpected token in body relationship definition")
		} else {
			rel.BodyCount++
		}
	}

	tok = p.stream.PeekTokenAt(0)
	switch tok.Type {
	case tokenizer.PIPE:
		// should only be encountered on --|> case as the end of the relationship
		p.stream.Emit()
		if _, ok := p.stream.TryConsumeType(tokenizer.RANGLE); !ok {
			return NewParserError(tok, "Expected '|>' after relationship")
		}
		switch rel.Body {
		case '-':
			rel.TypeRHS = ast.RelationInheritance
		case '.':
			rel.TypeRHS = ast.RelationRealization
		}
		rel.RArrow = rune(tok.Literal[0])
	case tokenizer.RANGLE:
		switch rel.Body {
		case '-':
			rel.TypeRHS = ast.RelationAssociation
		case '.':
			rel.TypeRHS = ast.RelationDependency
		}
		fallthrough
	case tokenizer.LBRACE:
		p.stream.Emit()
		// will return as this is the end of the relationship
		rel.RArrow = rune(tok.Literal[0])
	// lolipop interface
	case tokenizer.LPAREN:
		if sawDirection || sawAttrs {
			return NewParserError(tok, "Lolipop interface cannot contain direction or attributes")
		}
		if isLolipop {
			return NewParserError(tok, "Cannot have double headed lolipop relationship")
		}
		p.stream.Emit()
		p.stream.MustConsumeType(tokenizer.RPAREN)
		rel.RArrow = rune(tok.Literal[0])
	// position INDEPENDENT start and end tokens
	case tokenizer.ASTERISK:
		rel.TypeRHS = ast.RelationComposition
		p.stream.Emit()
		rel.RArrow = rune(tok.Literal[0])
	case tokenizer.IDENTIFIER:
		// special case for 'x' and 'o' in relationship
		switch tok.Literal {
		case "x":
			p.stream.Emit()
			rel.RArrow = rune(tok.Literal[0])
		case "o":
			if rel.Body == '-' {
				rel.TypeRHS = ast.RelationAggregation
			}
			p.stream.Emit()
			rel.RArrow = rune(tok.Literal[0])
		default:
			return NewParserError(tok, "Unexpected identifier in relationship definition")
		}
	case tokenizer.HASH, tokenizer.PLUS, tokenizer.CARET:
		p.stream.Emit()
		rel.RArrow = rune(tok.Literal[0])
	default:
		if rel.TypeLHS == ast.RelationUnknown && rel.Body == '-' {
			rel.TypeLHS = ast.RelationAssociation
			rel.TypeRHS = ast.RelationAssociation
		}
	}
	if rel.Body == 0 {
		return NewParserError(tok, "Missing body in the relationship")
	}
	return nil
}

