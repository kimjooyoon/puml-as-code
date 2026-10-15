package tokenizer

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"yur4uwe/pac/pkg/parser/keyword"
)

const maxConsecutiveEOF = 25

type UnexpectedTokenError struct {
	Expected Token
	Found    Token
}

func (e UnexpectedTokenError) Error() string {
	return fmt.Sprintf("token stream: unexpected token %s(%q) at %d:%d, expected %s(%q)", e.Found.Type, e.Found.Literal, e.Found.Span.Start.Line, e.Found.Span.Start.Col, e.Expected.Type, e.Expected.Literal)
}

func unexpectedTokenError(expected Token, found Token) error {
	return UnexpectedTokenError{Expected: expected, Found: found}
}

type TokenStream struct {
	lexer               *Lexer
	collectedTrivia     []Token
	buffer              []Token
	sinks               []TokenSink
	rawModeTerminator   []rune
	PackageSeparator    string
	lastNonNewlineToken Token
	eofEmitCount        int
}

func NewTokenStream(input string) *TokenStream {
	return &TokenStream{
		lexer:            NewLexer(input),
		PackageSeparator: ".",
		sinks:            make([]TokenSink, 0, 2),
	}
}

func (ts *TokenStream) MatchPackageSeparatorAt(startIdx int) (int, bool) {
	if ts.PackageSeparator == "" {
		return 0, false
	}
	var acc []Token
	for {
		accCount := len(acc)
		tok := ts.PeekRawTokenAt(startIdx + accCount)
		if tok.Type == EOF || tok.Type == NEWLINE {
			break
		}
		if ts.IsPackageSeparator(acc) {
			return accCount, true
		}
		acc = append(acc, tok)
	}
	return 0, false
}

func (ts *TokenStream) IsPackageSeparator(toks []Token) bool {
	var sb strings.Builder
	for i, tok := range toks {
		if i > 0 && tok.Span.Start.Offset != toks[i-1].Span.End.Offset {
			return false
		}
		sb.WriteString(tok.Literal)
		if sb.String() == ts.PackageSeparator {
			return true
		}
		if len(sb.String()) >= len(ts.PackageSeparator) {
			return false
		}
	}
	return false
}

func (ts *TokenStream) TryConsumePackageSeparator() (string, bool) {
	count, ok := ts.MatchPackageSeparatorAt(0)
	if !ok {
		return "", false
	}
	for range count {
		ts.Emit()
	}
	return ts.PackageSeparator, true
}

// PeekRawTokenAt returns the idx-th token in the stream.
func (ts *TokenStream) PeekRawTokenAt(idx int) Token {
	for len(ts.buffer) <= idx {
		tok := ts.lexer.Emit()
		ts.buffer = append(ts.buffer, tok)
		if tok.Type != EOF {
			ts.eofEmitCount = 0
			continue
		}

		ts.eofEmitCount++
		if ts.eofEmitCount > maxConsecutiveEOF {
			panic(fmt.Sprintf(
				"tokenizer: possible infinite loop detected: emitted EOF %d times (at %s)",
				ts.eofEmitCount, ts.lexer.getPos(),
			))
		}
		break
	}
	if idx < len(ts.buffer) {
		return ts.buffer[idx]
	}
	return ts.buffer[len(ts.buffer)-1]
}

// PeekTokenAt returns the idx-th NON-COMMENT token in the stream.
func (ts *TokenStream) PeekTokenAt(idx int) Token {
	nonCommentCount := 0
	for rawIdx := 0; ; rawIdx++ {
		tok := ts.PeekRawTokenAt(rawIdx)
		if tok.Type != COMMENT {
			if nonCommentCount == idx {
				return tok
			}
			nonCommentCount++
		}
		if tok.Type == EOF {
			return tok
		}
	}
}

// Emit consumes the next NON-COMMENT token and returns it.
func (ts *TokenStream) Emit() Token {
	for ts.PeekRawTokenAt(0).Type == COMMENT {
		commentTok := ts.EmitRaw()
		ts.collectedTrivia = append(ts.collectedTrivia, commentTok)
	}
	tok := ts.EmitRaw()
	return tok
}

// EmitRaw consumes the next token and returns it.
func (ts *TokenStream) EmitRaw() Token {
	tok := ts.PeekRawTokenAt(0)
	if len(ts.buffer) > 0 {
		ts.buffer = ts.buffer[1:]
	}
	for _, sink := range ts.sinks {
		sink.Receive(tok)
	}
	if tok.Type != EOF && tok.Type != NEWLINE && tok.Type != COMMENT {
		ts.lastNonNewlineToken = tok
	}
	return tok
}

func (ts *TokenStream) DumpCollectedTrivia() []Token {
	defer func() {
		ts.collectedTrivia = nil
	}()
	return ts.collectedTrivia
}

// EmitCommentToks consumes any immediate COMMENT tokens in the stream,
// appending them to collected trivia.
func (ts *TokenStream) EmitCommentToks() {
	for ts.PeekRawTokenAt(0).Type == COMMENT {
		ts.collectedTrivia = append(ts.collectedTrivia, ts.EmitRaw())
	}
}

func (ts *TokenStream) Attach(sink TokenSink) func() {
	ts.sinks = append(ts.sinks, sink)
	return func() {
		for i, s := range ts.sinks {
			if s == sink {
				ts.sinks = append(ts.sinks[:i], ts.sinks[i+1:]...)
				return
			}
		}
	}
}

// TokensToString is DEPRECATED as it can distort the original source
// Use SliceInputEnclosingTokens instead
func (ts *TokenStream) TokensToString(toks []Token) string {
	if len(toks) == 0 {
		return ""
	}

	var sb strings.Builder
	var prevTok *Token
	for i := range toks {
		tok := &toks[i]
		if tok.Type == COMMENT {
			continue
		}

		if prevTok != nil {
			startNext := tok.Span.Start.Offset
			endPrev := prevTok.Span.End.Offset
			if startNext > endPrev {
				sb.WriteString(string(ts.lexer.input[endPrev:startNext]))
			}
		}

		sb.WriteString(tok.Literal)
		prevTok = tok
	}

	return sb.String()
}

// ConsumeTextBlock consumes all tokens until a matching closer line is found.
// It handles single-token ("endheader") and multi-token ("end header") closers case-insensitively,
// preserves leading indentation on the body lines, strips indentation before the closer line,
// prevents false-positive matches mid-line, and tracks brace depth to avoid leaking past
// enclosing container scopes.
func (ts *TokenStream) ConsumeTextBlock(delimiterLiterals ...string) (string, error) {
	if len(delimiterLiterals) == 0 {
		return "", errors.New("empty closers for ConsumeTextBlock")
	}

	delim := strings.ToLower(strings.Join(delimiterLiterals, ""))

	var startBodyOffset uint
	if ts.AssertType(NEWLINE) {
		nlTok := ts.Emit()
		startBodyOffset = nlTok.Span.End.Offset
	} else if !ts.AssertType(EOF) {
		startBodyOffset = ts.PeekRawTokenAt(0).Span.Start.Offset
	}

	var closerStartOffset uint

	for {
		if ts.AssertType(EOF) {
			return "", ErrUnexpectedEOF
		}

		// Collect tokens on the current line
		curLineToks := ts.ConsumeUntilType(NEWLINE)
		if len(curLineToks) <= 0 {
			ts.TryConsumeType(NEWLINE)
			continue
		}

		if MatchTokenLine(curLineToks, delim) {
			closerStartOffset = curLineToks[0].Span.Start.Offset
			ts.TryConsumeType(NEWLINE)
			break
		} else if MatchTokenLine(curLineToks, "@enduml") {
			return "", ErrUnexpectedEOF
		}
	}

	rawBody := ts.SliceInput(startBodyOffset, closerStartOffset)
	if lastNL := strings.LastIndex(rawBody, "\n"); lastNL != -1 {
		rawBody = rawBody[:lastNL]
		rawBody = strings.TrimSuffix(rawBody, "\r")
	} else {
		rawBody = ""
	}

	return rawBody, nil
}

// Assert checks if the next token matches the given target (Type and Literal if literal is set).
// Does not emit the token.
func (ts *TokenStream) Assert(target Token) bool {
	next := ts.PeekTokenAt(0)
	if next.Type != target.Type {
		return false
	}
	if target.Literal != "" && next.Literal != target.Literal {
		return false
	}
	return true
}

func (ts *TokenStream) AssertType(targetType TokenType) bool {
	return ts.PeekTokenAt(0).Type == targetType
}

func (ts *TokenStream) AssertAny(targets ...Token) bool {
	return slices.ContainsFunc(targets, ts.Assert)
}

func (ts *TokenStream) AssertAnyType(targetTypes ...TokenType) bool {
	return slices.ContainsFunc(targetTypes, ts.AssertType)
}

func (ts *TokenStream) AssertKW(kw keyword.KeywordKind) bool {
	return keyword.Classify(ts.PeekTokenAt(0).Literal) == kw
}

func (ts *TokenStream) AssertSeq(seq []Token) bool {
	for i, t := range seq {
		next := ts.PeekTokenAt(i)
		if next.Type != t.Type {
			return false
		}
		if t.Literal != "" && next.Literal != t.Literal {
			return false
		}
	}
	return true
}

func (ts *TokenStream) AssertTypeSeq(seq []TokenType) bool {
	for i, t := range seq {
		next := ts.PeekTokenAt(i)
		if next.Type != t {
			return false
		}
	}
	return true
}

// MustConsumeType is ONLY for internal use as it panics
func (ts *TokenStream) MustConsumeType(targetType TokenType) Token {
	if !ts.AssertType(targetType) {
		panic(fmt.Sprintf("expected %s, got %s", targetType, ts.PeekTokenAt(0).Type))
	}
	return ts.Emit()
}

func (ts *TokenStream) TryConsume(target Token) (Token, bool) {
	if ts.Assert(target) {
		return ts.Emit(), true
	}
	return Token{}, false
}

func (ts *TokenStream) TryConsumeType(targetType TokenType) (Token, bool) {
	if ts.AssertType(targetType) {
		return ts.Emit(), true
	}
	return Token{}, false
}

func (ts *TokenStream) TryConsumeKW(kw keyword.KeywordKind) (Token, bool) {
	if ts.AssertKW(kw) {
		return ts.Emit(), true
	}
	return Token{}, false
}

// ConsumeUntil consumes tokens until it encounters one of the specified target tokens or EOF.
// Returns the slice of consumed tokens.
// NOTE: The terminator token itself is NOT consumed and NOT included in the returned slice;
// it remains at the head of the stream for callers to assert (e.g. via ts.Assert / ts.AssertType).
func (ts *TokenStream) ConsumeUntil(targets ...Token) []Token {
	var consumed []Token
	for !ts.AssertAny(targets...) && !ts.AssertType(EOF) {
		consumed = append(consumed, ts.Emit())
	}
	return consumed
}

// ConsumeUntilType consumes tokens until it encounters one of the specified token types or EOF.
// Returns the slice of consumed tokens.
// NOTE: The terminator token itself is NOT consumed and NOT included in the returned slice;
// it remains at the head of the stream for callers to assert (e.g. via ts.Assert / ts.AssertType).
func (ts *TokenStream) ConsumeUntilType(targetTypes ...TokenType) []Token {
	var consumed []Token
	for !ts.AssertAnyType(targetTypes...) && !ts.AssertType(EOF) {
		consumed = append(consumed, ts.Emit())
	}
	return consumed
}

var (
	ErrStartMarkerNotFound = errors.New("start marker not found")
	ErrUnexpectedEOF       = errors.New("unexpected EOF")
	ErrScopeDelimiterHit   = errors.New("hit enclosing scope delimiter")
)

// ReadBetween handles the common pattern of [start markers]...[end markers]
func (ts *TokenStream) ReadBetween(start, end []Token) (string, error) {
	// 1. Check if start markers match
	if !ts.AssertSeq(start) {
		return "", ErrStartMarkerNotFound
	}

	// 2. Consume start markers
	for range start {
		ts.Emit()
	}

	// 3. Read content until end markers
	var collected []Token
	for {
		if ts.AssertType(EOF) || ts.AssertType(NEWLINE) {
			return "", ErrUnexpectedEOF
		}

		if ts.AssertSeq(end) {
			break
		}

		collected = append(collected, ts.Emit())
	}

	res := ts.SliceInputEnclosingTokens(collected...)

	// Consume end markers
	for range end {
		ts.Emit()
	}

	return res, nil
}

func (ts *TokenStream) collectUntilNewline(emitter func() Token) []Token {
	if ts.AssertType(EOF) || ts.AssertType(NEWLINE) {
		return nil
	}
	buf := []Token{}
	for {
		tok := emitter()
		buf = append(buf, tok)
		if tok.Type == NEWLINE || tok.Type == EOF {
			break
		}
	}
	return buf
}

func (ts *TokenStream) ReadUntilNewline() string {
	toks := ts.ConsumeUntilType(NEWLINE)
	if len(toks) == 0 {
		return ""
	}

	return ts.SliceInputEnclosingTokens(toks...)
}

// ReadBlock assumes that end tokens are first in the line
// So actual end markers are implicitly append([]Token{NEWLINE}, ...endMark)
func (ts *TokenStream) ReadBlock(endMark ...Token) (string, error) {
	tok := ts.EmitRaw()
	startPos := tok.Span.Start
	for tok.Type != EOF {
		if tok.Type == NEWLINE && ts.AssertSeq(endMark) {
			endPos := tok.Span.Start
			for range endMark {
				ts.EmitRaw() // consume end markers
			}
			return string(ts.lexer.input[startPos.Offset:endPos.Offset]), nil
		}
		tok = ts.EmitRaw()
	}
	return "", fmt.Errorf("block end not found")
}

func (ts *TokenStream) SliceInput(start, end uint) string {
	if int(start) > len(ts.lexer.input) {
		return ""
	}
	if int(end) > len(ts.lexer.input) {
		end = uint(len(ts.lexer.input))
	}
	if start > end {
		return ""
	}
	return string(ts.lexer.input[start:end])
}

func (ts *TokenStream) SliceInputEnclosingTokens(toks ...Token) string {
	span, ok := TokenSliceSpan(toks)
	if !ok {
		return ""
	}
	return ts.SliceInput(span.Start.Offset, span.End.Offset)
}

func (ts *TokenStream) LastSemanticToken() Token {
	return ts.lastNonNewlineToken
}
