package parser

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"
)

func unamb(tok tokenizer.TokenType) tokenizer.Token {
	return tokenizer.Token{Type: tok}
}

func amb(tok tokenizer.TokenType, literal string) tokenizer.Token {
	return tokenizer.Token{Type: tok, Literal: literal}
}

type Parser struct {
	ast         *ast.Diagram
	stream      *tokenizer.TokenStream
	Dialect     dialect.Dialect
	TargetID    string
	IsBoundless bool
}

func NewParser(d dialect.Dialect) *Parser {
	return &Parser{
		Dialect: d,
	}
}

func (p *Parser) WithTargetID(targetID string) *Parser {
	p.TargetID = targetID
	return p
}

func (p *Parser) moveToDiagStart() error {
	if p.isStartMarker() {
		return nil
	}
	// find the target diagram start marker
	for tok := p.stream.Emit(); tok.Type != tokenizer.EOF; tok = p.stream.Emit() {
		if tok.Type != tokenizer.NEWLINE {
			continue
		}
		if p.isStartMarker() {
			return nil
		}
	}

	return tokenizer.ErrUnexpectedEOF
}

func (p *Parser) Parse(input string) (*ast.Diagram, error) {
	if p.Dialect == nil {
		return nil, errors.New("dialect not initialized")
	}

	p.ast = &ast.Diagram{}
	p.stream = tokenizer.NewTokenStream(input)

	var startBound *ast.DiagramBound
	if !p.IsBoundless {
		// Initial blockNum is -1 so that the first block is 0-indexed
		blockNum := -1
		for {
			err := p.moveToDiagStart()
			if err != nil {
				if blockNum == -1 {
					return nil, errors.New("no diagrams found")
				}
				return nil, fmt.Errorf("diagram block %s not found, file has %d blocks", p.TargetID, blockNum+1)
			}

			blockNum++

			startBound, err = p.readStartBounds()
			if err != nil {
				return nil, err
			} else if !startBound.IsStart {
				return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected diagram start marker")
			}

			if p.TargetID == "" {
				break
			}

			num, err := strconv.Atoi(p.TargetID)
			if err == nil {
				// We have numerical ID which means order of the block in the file
				if num != blockNum {
					continue
				} else {
					break
				}
			}

			// Otherwise, we have a named ID which have to match
			if p.TargetID == startBound.ID {
				break
			}
		}

		p.ast.Statements = append(p.ast.Statements, startBound)
		p.ast.Name = startBound.DiagramName()
	}

	for {
		// End condition check should be before consuming a token to avoid swallowing '@'
		tok := p.stream.Emit()
		if tok.Type == tokenizer.EOF {
			if p.IsBoundless {
				break
			}
			eofDiagnst := &ast.Diagnostic{
				ID:      len(p.ast.Diagnostics),
				Message: tokenizer.ErrUnexpectedEOF.Error(),
				Span:    p.stream.LastSemanticToken().Span,
				Err:     tokenizer.ErrUnexpectedEOF,
			}
			p.recordDiagnostic(eofDiagnst)
			break
		} else if tok.Type == tokenizer.NEWLINE {
			// We can leave it like this for now
			// If the newline is relevant it will be consumed
			// by the statement parser in some function
			continue
		}

		// Here tokens that are first in line are handled
		// Parsing must be constructed to result in a single statement per iteration
		// and to have NEWLINE at the head of the token stream that ends the statement

		// Imports via !include
		// Styles via <style>
		// Keyword handling switch
		// Handle comments
		// Handle Identifiers

		stmt := p.routeDiagramStatement(tok)
		p.ast.Statements = append(p.ast.Statements, stmt)
		if endBound, ok := stmt.(*ast.DiagramBound); ok && !endBound.IsStart {
			if endBound.Type != startBound.Type {
				msg := fmt.Errorf("types of starting and ending markers don't match")
				p.recordDiagnostic(&ast.Diagnostic{
					ID:      len(p.ast.Diagnostics),
					Message: msg.Error(),
					Span:    endBound.NodeSpan,
					Node:    endBound,
					Err:     msg,
				})
			}
			break
		}
	}

	if p.TargetID == "" {
		// Keep searching for the next diagram start marker
		// User might not know about discarded blocks
		err := p.moveToDiagStart()
		if err == nil {
			log.Printf("warning: discarded diagram block at %s and all following blocks\n", p.stream.PeekTokenAt(0).Span.Start)
		}
	}

	if p.ast.HasErrors() {
		return p.ast, p.ast.Diagnostics
	}
	return p.ast, nil
}

func (p *Parser) parseContainerStatement(tok tokenizer.Token) (ast.Statement, error) {
	if tok.Type == tokenizer.EXCLAMATION {
		return p.parseDirective(tok)
	}

	if stmt, err := p.disambiguateIdent(tok); stmt != nil || err != nil {
		return stmt, err
	}

	switch keyword.Classify(tok.Literal) {
	case keyword.Header,
		keyword.Footer,
		keyword.Legend:
		return p.parseLayoutStatement(tok, nil)
	case keyword.Direction:
		nextTok := p.stream.PeekTokenAt(0)
		nextKW := keyword.Classify(nextTok.Literal)
		if nextKW == keyword.Header || nextKW == keyword.Footer || nextKW == keyword.Legend || nextKW == keyword.Title {
			actualKwTok := p.stream.Emit()
			return p.parseLayoutStatement(actualKwTok, &tok)
		}
		return nil, NewParserError(tok, "Unexpected direction keyword in container")
	case keyword.Caption, keyword.Sprite:
		return p.parseUnhandled(tok)
	case keyword.Class,
		keyword.Interface,
		keyword.Struct,
		keyword.Abstract,
		keyword.Enum,
		keyword.Annotation,
		keyword.Record,
		keyword.Dataclass,
		keyword.Exception,
		keyword.Protocol,
		keyword.Entity:
		// Class-like Entities
		return p.parseEntity(tok)
	// Containers
	case keyword.Package,
		keyword.Together,
		keyword.Folder,
		keyword.Frame,
		keyword.Rectangle,
		keyword.Cloud,
		keyword.Database,
		keyword.Namespace,
		keyword.Node:
		return p.parseContainer(tok)
	case keyword.Circle, keyword.Diamond, keyword.Metaclass, keyword.Stereotype:
		return nil, NewParserError(tok, "unimplemented entity keyword handling")
	// Special Keywords
	case keyword.Note:
		return p.parseNote(tok)
	}

	return nil, nil
}

func (p *Parser) parseDiagramOnlyStatement(tok tokenizer.Token) (ast.Statement, error) {
	if !p.IsBoundless && p.isDiagramEndMarker(tok) {
		return p.readEndBounds(tok)
	}
	if tok.Type == tokenizer.LANGLE && p.stream.AssertSeq([]tokenizer.Token{amb(tokenizer.IDENTIFIER, "style"), unamb(tokenizer.RANGLE)}) {
		return p.parseStyleBlock(tok)
	}
	if tok.Type == tokenizer.EXCLAMATION {
		return p.parseDirective(tok)
	}

	if stmt, err := p.disambiguateIdent(tok); stmt != nil || err != nil {
		return stmt, err
	}

	switch keyword.Classify(tok.Literal) {
	case keyword.Skinparam:
		return p.parseSkinparamRoot(tok)
	case keyword.Title,
		keyword.Header,
		keyword.Footer,
		keyword.Legend:
		return p.parseLayoutStatement(tok, nil)
	case keyword.Hide, keyword.Show, keyword.Remove, keyword.Restore:
		return p.parseVisibilityCommand(tok)
	case keyword.Scale:
		return p.parseScale(tok)
	case keyword.Direction:
		nextTok := p.stream.PeekTokenAt(0)
		if nextTok.Literal == "to" {
			return p.parseDiagDirection(tok)
		} else if nextKW := keyword.Classify(nextTok.Literal); nextKW == keyword.Header ||
			nextKW == keyword.Footer ||
			nextKW == keyword.Legend ||
			nextKW == keyword.Title {
			actualKwTok := p.stream.Emit()
			return p.parseLayoutStatement(actualKwTok, &tok)
		} else {
			return nil, NewParserError(tok, "Unexpected token after direction")
		}
	case keyword.Set:
		return p.parseSetDirective(tok)
	}

	return nil, nil
}
