package parser

import (
	"errors"
	"fmt"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

type parserError struct {
	Err  error
	Tok  tokenizer.Token
	File string
}

var _ error = parserError{}

func (e parserError) Error() string {
	return fmt.Sprintf("parser: %v at %d:%d token: %s(%s)", e.Err, e.Tok.Span.Start.Line, e.Tok.Span.Start.Col, e.Tok.Type.String(), e.Tok.Literal)
}

func (e parserError) Unwrap() error {
	return e.Err
}

func NewParserError(tok tokenizer.Token, message string) error {
	return parserError{Err: errors.New(message), Tok: tok}
}

func NewParserErrorFromIdent(id ast.Ident, message string) error {
	return parserError{Err: errors.New(message), Tok: tokenizer.Token{Type: tokenizer.IDENTIFIER, Literal: id.Literal, Span: id.Span}}
}

func NewParserErrorf(tok tokenizer.Token, format string, args ...any) error {
	return parserError{Err: fmt.Errorf(format, args...), Tok: tok}
}

func WrapParserError(tok tokenizer.Token, err error) error {
	return parserError{Err: err, Tok: tok}
}
