package parser

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) parseNote(startTok tokenizer.Token) (*ast.Note, error) {
	// tok is a keyword 'note'
	note := &ast.Note{
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}
	tok := p.stream.Emit()
	var err error
	if tok.Type == tokenizer.STRING {
		err = p.parseInlineIdentNote(note, tok)
	} else {
		class := keyword.Classify(tok.Literal)
		switch class {
		case keyword.Direction:
			err = p.parseDirectedNote(note, tok)
		case keyword.Position:
			err = p.parseLinkNote(note, tok)
		case keyword.Alias:
			err = p.parseMultilineAliasNote(note)
		default:
			return nil, NewParserErrorf(tok, "expected direction, string, note position or alias after 'note', got %s", class.String())
		}
	}
	if err != nil {
		return nil, err
	}
	return note, nil
}

// parseDirectedNote can resolve into either a NoteTargeted or NoteLink.
func (p *Parser) parseDirectedNote(note *ast.Note, dirTok tokenizer.Token) error {
	note.Kind = ast.NoteTargeted
	note.Direction = p.mapTokenToDirection(dirTok)
	if relativeTok, ok := p.stream.TryConsumeKW(keyword.Position); ok {
		target, err := p.parseTargetRef(p.stream.Emit()) // consume target
		if err != nil {
			return err
		}
		if relativeTok.Literal == "on" {
			if strings.ToLower(target.EntityName()) != "link" {
				return NewParserError(relativeTok, "Unexpected identifier for a note link target")
			}
			note.Kind = ast.NoteLink
		} else {
			note.Kind = ast.NoteTargeted
		}
		note.Target = &target
	} else if tok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER); ok {
		return NewParserError(tok, "Unexpected identifier after direction")
	}
	if color := p.tryParseColor(); color != "" {
		note.Color = color
	}
	return p.parseNoteBody(note)
}

func (p *Parser) parseInlineIdentNote(note *ast.Note, stringTok tokenizer.Token) error {
	note.Text = strings.Trim(stringTok.Literal, "\"")
	note.Kind = ast.NoteAlias
	if aliasTok, ok := p.stream.TryConsumeKW(keyword.Alias); !ok {
		return NewParserError(aliasTok, "Expected alias keyword after note text")
	}
	tok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
	if !ok {
		return NewParserError(tok, "Expected identifier after alias keyword")
	}
	note.Identifier = ast.TargetRef{Name: tok}
	if color := p.tryParseColor(); color != "" {
		note.Color = color
	}
	if !p.stream.AssertAnyType(tokenizer.NEWLINE, tokenizer.EOF) {
		return NewParserError(p.stream.PeekTokenAt(0), "Unexpected tokens after inline alias note")
	}
	return nil
}

func (p *Parser) parseMultilineAliasNote(note *ast.Note) error {
	note.Kind = ast.NoteAlias
	tok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
	if !ok {
		return NewParserError(tok, "Expected identifier after alias keyword")
	}
	note.Identifier = ast.TargetRef{Name: tok}
	if color := p.tryParseColor(); color != "" {
		note.Color = color
	}
	if !p.stream.AssertType(tokenizer.NEWLINE) {
		return NewParserError(tok, "Expected newline after alias keyword")
	}
	return p.parseNoteBody(note)
}

func (p *Parser) parseLinkNote(note *ast.Note, onTok tokenizer.Token) error {
	if onTok.Literal != "on" {
		return NewParserError(onTok, "Unexpected identifier after 'note'")
	}
	linkTok, ok := p.stream.TryConsumeType(tokenizer.IDENTIFIER)
	if !ok || linkTok.Literal != "link" {
		return NewParserError(onTok, "Expected 'link' after 'note on'")
	}
	note.Target = &ast.TargetRef{Name: linkTok}
	note.Kind = ast.NoteLink
	if color := p.tryParseColor(); color != "" {
		note.Color = color
	}
	return p.parseNoteBody(note)
}

func (p *Parser) parseNoteBody(note *ast.Note) error {
	tok := p.stream.PeekTokenAt(0)
	switch tok.Type {
	case tokenizer.COLON:
		p.stream.Emit()
		note.Text = p.stream.ReadUntilNewline()
		return nil
	case tokenizer.NEWLINE:
		note.TrailingTrivia = p.stream.DumpCollectedTrivia()
		body, err := p.stream.ConsumeTextBlock("end", "note")
		if err != nil {
			return err
		}
		note.Text = body
		return nil
	default:
		p.stream.Emit()
		return NewParserError(tok, "Expected ':' or newline after note definition")
	}
}
