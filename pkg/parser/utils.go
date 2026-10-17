package parser

import (
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
	tokens := []tokenizer.Token{firstTok}

	for {
		length, ok := p.stream.MatchPackageSeparatorAt(0)
		if !ok {
			break
		}

		afterPkgSep := p.stream.PeekTokenAt(length)
		if afterPkgSep.Type != tokenizer.IDENTIFIER {
			// We have hit the case where the separator token is not a part of the entity reference
			break
		}

		for range length {
			p.stream.Emit()
		}

		tokens = append(tokens, p.stream.Emit())
	}

	if len(tokens) > 0 {
		ref.Name = tokens[len(tokens)-1]
		if len(tokens) > 1 {
			ref.PackagePath = tokens[:len(tokens)-1]
		}
	}

	// If encountering a () we in the case where package separator is '::' and encountered method reference
	if p.stream.AssertType(tokenizer.LPAREN) {
		lparenTok := p.stream.Emit() // consume (
		memberRef := &ast.MemberRef{
			Name: ref.Name,
		}
		paramToks := p.stream.ConsumeUntilType(tokenizer.RPAREN, tokenizer.NEWLINE)
		rparenTok, ok := p.stream.TryConsumeType(tokenizer.RPAREN)
		if !ok {
			return ref, NewParserError(p.stream.PeekTokenAt(0), "Expected ')' closing method signature in target ref")
		}
		sig := make([]tokenizer.Token, 0, len(paramToks)+2)
		sig = append(sig, lparenTok)
		sig = append(sig, paramToks...)
		sig = append(sig, rparenTok)
		memberRef.Signature = sig
		ref.Name = ref.PackagePath[len(ref.PackagePath)-1]
		ref.PackagePath = ref.PackagePath[:len(ref.PackagePath)-1]
		ref.Member = memberRef
	}

	if p.stream.AssertTypeSeq([]tokenizer.TokenType{tokenizer.COLON, tokenizer.COLON}) {
		p.stream.Emit() // consume first :
		p.stream.Emit() // consume second :

		tok := p.stream.Emit()
		switch tok.Type {
		case tokenizer.STRING:
			ref.Member = &ast.MemberRef{
				Name: tok,
			}
		case tokenizer.IDENTIFIER:
			memberRef := &ast.MemberRef{
				Name: tok,
			}
			if lparenTok, ok := p.stream.TryConsumeType(tokenizer.LPAREN); ok {
				paramToks := p.stream.ConsumeUntilType(tokenizer.RPAREN, tokenizer.NEWLINE)
				rparenTok, ok := p.stream.TryConsumeType(tokenizer.RPAREN)
				if !ok {
					return ref, NewParserError(p.stream.PeekTokenAt(0), "Expected ')' closing method signature in target ref")
				}
				sig := make([]tokenizer.Token, 0, len(paramToks)+2)
				sig = append(sig, lparenTok)
				sig = append(sig, paramToks...)
				sig = append(sig, rparenTok)
				memberRef.Signature = sig
			}
			ref.Member = memberRef
		default:
			return ref, NewParserError(tok, "Expected member identifier or string after '::'")
		}
	}

	return ref, nil
}
