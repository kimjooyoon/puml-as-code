package parser

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) tryParseColor() string {
	if !p.stream.AssertType(tokenizer.HASH) {
		return ""
	}
	tokens := p.stream.ConsumeUntilType(tokenizer.NEWLINE, tokenizer.COLON, tokenizer.LBRACE)
	return p.stream.SliceInputEnclosingTokens(tokens...)
}

func (p *Parser) parseTargetRef(firstTok tokenizer.Token) (ast.TargetRef, error) {
	var ref ast.TargetRef
	segments := []string{firstTok.Literal}

	for {
		length, ok := p.stream.MatchPackageSeparatorAt(0)
		if !ok {
			break
		}

		afterPkgSep := p.stream.PeekTokenAt(length)
		if afterPkgSep.Type != tokenizer.IDENTIFIER && afterPkgSep.Type != tokenizer.STRING {
			// We have hit the case where the separator token is not a part of the entity reference
			break
		}

		for range length {
			p.stream.Emit()
		}

		segments = append(segments, p.stream.Emit().Literal)
	}

	if len(segments) > 0 {
		ref.Entity = segments[len(segments)-1]
		if len(segments) > 1 {
			ref.PackagePath = segments[:len(segments)-1]
		}
	}

	if p.stream.AssertTypeSeq([]tokenizer.TokenType{tokenizer.COLON, tokenizer.COLON}) {
		p.stream.Emit() // consume first :
		p.stream.Emit() // consume second :

		tok := p.stream.Emit()
		switch tok.Type {
		case tokenizer.STRING:
			ref.Member = tok.Literal
		case tokenizer.IDENTIFIER:
			var sb strings.Builder
			sb.WriteString(tok.Literal)
			if p.stream.AssertType(tokenizer.LPAREN) {
				p.stream.Emit() // consume (
				sb.WriteString("(")
				paramToks := p.stream.ConsumeUntilType(tokenizer.RPAREN, tokenizer.NEWLINE, tokenizer.EOF)
				sb.WriteString(p.stream.TokensToString(paramToks))
				if _, ok := p.stream.TryConsumeType(tokenizer.RPAREN); !ok {
					return ref, NewParserError(p.stream.PeekTokenAt(0), "Expected ')' closing method signature in target ref")
				}
				sb.WriteString(")")
			}
			ref.Member = sb.String()
		default:
			return ref, NewParserError(tok, "Expected member identifier or string after '::'")
		}
	}

	return ref, nil
}
