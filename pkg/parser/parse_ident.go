package parser

import (
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"
)

func isAdjacent(prev, curr tokenizer.Token) bool {
	return prev.Span.End.Offset == curr.Span.Start.Offset
}

func isRelStartArrow(tok tokenizer.Token) bool {
	switch tok.Type {
	case tokenizer.IDENTIFIER:
		return tok.Literal == "o" || tok.Literal == "x"
	case tokenizer.DOT, tokenizer.DASH, tokenizer.LANGLE, tokenizer.RBRACE,
		tokenizer.LPAREN, tokenizer.ASTERISK, tokenizer.CARET, tokenizer.HASH,
		tokenizer.PLUS:
		return true
	default:
		return false
	}
}

func (p *Parser) disambiguateIdent(identTok tokenizer.Token) (ast.Statement, error) {
	prevTok := identTok
	offst := 0
	isQualified := false

	for {
		nextTok := p.stream.PeekTokenAt(offst)
		if !isAdjacent(prevTok, nextTok) {
			break
		}

		// peek at composite names
		count, ok := p.stream.MatchPackageSeparatorAt(offst)
		if !ok {
			break
		}

		sepEndTok := p.stream.PeekTokenAt(offst + count - 1)
		followingIdent := p.stream.PeekTokenAt(offst + count)
		// if isRelStartArrow(sepEndTok) we can have -left-> arrow (or -left[...]->)
		// we must anticipate this case
		if followingIdent.Type != tokenizer.IDENTIFIER || !isAdjacent(sepEndTok, followingIdent) {
			break
		}

		isQualified = true
		offst += count + 1
		prevTok = followingIdent
	}

	nextTok := p.stream.PeekTokenAt(offst)

	if isRelStartArrow(nextTok) {
		return p.parseRelationship(identTok)
	}

	if isQualified {
		switch nextTok.Type {
		case tokenizer.STRING:
			// it is a relationship multiplicity
			return p.parseRelationship(identTok)
		case tokenizer.COLON:
			if p.stream.PeekTokenAt(offst+1).Type == tokenizer.COLON {
				return p.parseRelationship(identTok)
			}
			return p.parseInlineMember(identTok)
		}

		return nil, NewParserError(nextTok, "expected relationship arrow or ':' after qualified target")
	}

	switch nextTok.Type {
	case tokenizer.LBRACE:
		if keyword.Classify(identTok.Literal) == keyword.Skinparam {
			return p.parseSkinparamRoot(identTok)
		}
		// delegate to keyword resolution
		return nil, nil
	case tokenizer.STRING:
		followingTok := p.stream.PeekTokenAt(offst + 1)
		if followingTok.Type == tokenizer.IDENTIFIER {
			if followingTok.Literal == "o" || followingTok.Literal == "x" {
				return p.parseRelationship(identTok)
			}
			// We should have a container or an entity
			// basically delegate to keyword resolution
			return nil, nil
		} else if followingTok.Type == tokenizer.LANGLE {
			// handle class "name" <t> vs User "1" --> "*" Order
			followingTok = p.stream.PeekTokenAt(offst + 2)
			switch followingTok.Type {
			case tokenizer.PIPE, tokenizer.DOT, tokenizer.DASH:
				return p.parseRelationship(identTok)
			case tokenizer.LANGLE:
				// stereotype
				return nil, nil
			case tokenizer.IDENTIFIER:
				// generic
				return nil, nil
			default:
				return nil, NewParserError(followingTok, "must be either relationship continuation or generic identifier")
			}
		} else if followingTok.Type == tokenizer.HASH {
			afterHash := p.stream.PeekTokenAt(offst + 2)
			if afterHash.Type == tokenizer.DASH || afterHash.Type == tokenizer.DOT {
				return p.parseRelationship(identTok)
			}
			return nil, nil // entity with color e.g. class "Foo" #red
		} else if isRelStartArrow(followingTok) {
			return p.parseRelationship(identTok)
		} else {
			return nil, nil
		}
	case tokenizer.COLON:
		if p.stream.PeekTokenAt(offst+1).Type == tokenizer.COLON {
			return p.parseRelationship(identTok)
		}
		return p.parseInlineMember(identTok)
	default:
		// return error here
		return nil, nil
	}
}
