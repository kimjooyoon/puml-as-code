package parser

import (
	"errors"
	"log"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"
)

func (p *Parser) parseContainer(tok tokenizer.Token) (*ast.Container, error) {
	containerClass := keyword.Classify(tok.Literal)
	container := &ast.Container{
		Kind: p.mapKeywordToContainerKind(containerClass),
		BaseNode: ast.BaseNode{
			LeadingTrivia: p.stream.DumpCollectedTrivia(),
		},
	}

	// Handle container name, alias, stereotype, color
	// Name rules:
	// - Name and alias must be a string or Identifier
	// - Name and alias CAN be of the same type
	// - if Name and alias are of the same type, alias is on the left
	// and Name is on the right of 'as' keyword

	if containerClass != keyword.Together {
		var err error
		container.Alias, container.Identifier, err = p.parseContainerIdentAndAlias()
		if err != nil {
			return nil, err
		}
		preTags := p.tryReadTags()
		container.Stereotype, err = p.tryReadStereotype()
		if err != nil && !errors.Is(err, tokenizer.ErrStartMarkerNotFound) {
			return nil, err
		}
		postTags := p.tryReadTags()
		if len(preTags) > 0 {
			if len(postTags) > 0 {
				log.Printf("warning: tags %v after stereotype on container %q are ignored in favor of pre-stereotype tags %v\n", postTags, container.Identifier.EntityName(), preTags)
			}
			container.Tags = preTags
		} else if len(postTags) > 0 {
			container.Tags = postTags
		}

		container.Color = p.tryParseColor()
	}

	if _, ok := p.stream.TryConsumeType(tokenizer.LBRACE); !ok {
		if !p.stream.AssertType(tokenizer.NEWLINE) {
			return nil, NewParserError(p.stream.PeekTokenAt(0), "Expected container body to end")
		}
		return container, nil
	}

	p.stream.EmitCommentToks()
	container.TrailingTrivia = p.stream.DumpCollectedTrivia()

	for tok := p.stream.Emit(); tok.Type != tokenizer.RBRACE; tok = p.stream.Emit() {
		if tok.Type == tokenizer.EOF {
			return nil, tokenizer.ErrUnexpectedEOF
		} else if tok.Type == tokenizer.NEWLINE {
			// Skip NEWLINE tokens inside parseContainer() so empty lines within
			// container blocks { ... } do not trigger an erroneous "Expected a statement in a
			// container body" error.
			continue
		}

		stmt := p.routeContainerStatement(tok)
		container.Statements = append(container.Statements, stmt)
	}

	return container, nil
}

// parseContainerIdentAndAlias parses the container name and alias.
//
// Returns:
// - Alias (Token)
// - Identifier (TargetRef)
// - error
func (p *Parser) parseContainerIdentAndAlias() (tokenizer.Token, ast.TargetRef, error) {
	firstTok := p.stream.Emit()
	var (
		lhsTok      tokenizer.Token
		lhsTarget   ast.TargetRef
		lhsIsTarget bool
		err         error
	)

	switch firstTok.Type {
	case tokenizer.STRING:
		lhsTok = firstTok
	case tokenizer.IDENTIFIER:
		lhsTarget, err = p.parseTargetRef(firstTok)
		if err != nil {
			return tokenizer.Token{}, ast.TargetRef{}, err
		}
		lhsIsTarget = true
	default:
		return tokenizer.Token{}, ast.TargetRef{}, NewParserError(firstTok, "Expected identifier or string for container name")
	}

	if _, ok := p.stream.TryConsumeKW(keyword.Alias); !ok {
		if lhsIsTarget {
			return tokenizer.Token{}, lhsTarget, nil
		}
		return tokenizer.Token{}, ast.TargetRef{Name: lhsTok}, nil
	}

	secondTok := p.stream.Emit()
	var (
		rhsTok      tokenizer.Token
		rhsTarget   ast.TargetRef
		rhsIsTarget bool
	)

	switch secondTok.Type {
	case tokenizer.STRING:
		rhsTok = secondTok
	case tokenizer.IDENTIFIER:
		rhsTarget, err = p.parseTargetRef(secondTok)
		if err != nil {
			return tokenizer.Token{}, ast.TargetRef{}, err
		}
		rhsIsTarget = true
	default:
		return tokenizer.Token{}, ast.TargetRef{}, NewParserError(secondTok, "Expected identifier or string for container alias")
	}

	if lhsIsTarget && !rhsIsTarget {
		return rhsTok, lhsTarget, nil
	}
	if !lhsIsTarget && rhsIsTarget {
		return lhsTok, rhsTarget, nil
	}
	if lhsIsTarget && rhsIsTarget {
		// When both are identifiers, alias is on the left, identifier is on the right
		return lhsTarget.Name, rhsTarget, nil
	}
	// When both are strings
	return lhsTok, ast.TargetRef{Name: rhsTok}, nil
}

