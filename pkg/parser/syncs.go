package parser

import "yur4uwe/pac/pkg/tokenizer"

func (p *Parser) syncToLineEnd() []tokenizer.Token {
	var consumed []tokenizer.Token
	for !p.stream.AssertType(tokenizer.NEWLINE) &&
		!p.stream.AssertType(tokenizer.EOF) &&
		!p.isEndMarker() {
		consumed = append(consumed, p.stream.Emit())
	}
	return consumed
}

func (p *Parser) syncToLineOrBlockEnd() []tokenizer.Token {
	var consumed []tokenizer.Token
	for !p.stream.AssertType(tokenizer.NEWLINE) &&
		!p.stream.AssertType(tokenizer.EOF) &&
		!p.isEndMarker() &&
		!p.stream.AssertType(tokenizer.RBRACE) {
		consumed = append(consumed, p.stream.Emit())
	}
	return consumed
}
