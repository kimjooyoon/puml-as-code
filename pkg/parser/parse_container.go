package parser

import (
	"errors"
	"log"
	"slices"
	"strings"

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
				log.Printf("warning: tags %v after stereotype on container %q are ignored in favor of pre-stereotype tags %v\n", postTags, container.Identifier, preTags)
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
		return p.wrapImplicitPackageContainers(container), nil
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

	return p.wrapImplicitPackageContainers(container), nil
}

func (p *Parser) wrapImplicitPackageContainers(c *ast.Container) *ast.Container {
	if p.stream.PackageSeparator == "" || !strings.Contains(c.Identifier, p.stream.PackageSeparator) {
		return c
	}
	segments := strings.Split(c.Identifier, p.stream.PackageSeparator)
	if len(segments) <= 1 {
		return c
	}

	current := &ast.Container{
		Kind:       c.Kind,
		Identifier: segments[len(segments)-1],
		Alias:      c.Alias,
		Stereotype: c.Stereotype,
		Color:      c.Color,
		Statements: c.Statements,
	}
	for _, pkg := range slices.Backward(segments[:len(segments)-1]) {
		current = &ast.Container{
			Kind:       c.Kind,
			Identifier: pkg,
			Statements: []ast.Statement{current},
		}
	}
	return current
}

func (p *Parser) parseContinerIdent(tok tokenizer.Token) (tokenizer.Token, error) {
	if tok.Type == tokenizer.STRING {
		return tok, nil
	}
	if tok.Type != tokenizer.IDENTIFIER {
		return tokenizer.Token{}, NewParserError(tok, "Expected identifier for container name")
	}

	var sb strings.Builder
	sb.WriteString(tok.Literal)
	lastTok := tok
	for tok = p.stream.PeekTokenAt(0); tok.Type != tokenizer.EOF; tok = p.stream.PeekRawTokenAt(0) {
		// early exit if we see an alias keyword and not hit the
		// the double identifier case
		if keyword.Classify(tok.Literal) == keyword.Alias {
			return amb(tokenizer.IDENTIFIER, sb.String()), nil
		}
		if sep, ok := p.stream.TryConsumePackageSeparator(); ok {
			sb.WriteString(sep)
			lastTok = tokenizer.Token{Type: tokenizer.DOT}
			continue
		}
		if lastTok.Type == tokenizer.IDENTIFIER && tok.Type == tokenizer.IDENTIFIER {
			return tokenizer.Token{}, NewParserError(tok, "Expected container name to be a single identifier")
		}
		switch tok.Type {
		case tokenizer.LBRACE, tokenizer.LANGLE, tokenizer.HASH, tokenizer.NEWLINE, tokenizer.DOLLAR:
			return amb(tokenizer.IDENTIFIER, sb.String()), nil
		default:
			sb.WriteString(tok.Literal)
			p.stream.Emit()
			lastTok = tok
		}
	}

	return tokenizer.Token{}, tokenizer.ErrUnexpectedEOF
}

// parseContainerIdentAndAlias parses the container name and alias
//
// Returns
// - Alias
// - Identifier
// - error
func (p *Parser) parseContainerIdentAndAlias() (string, string, error) {
	lhs, err := p.parseContinerIdent(p.stream.Emit())
	if err != nil {
		return "", "", err
	}
	if _, ok := p.stream.TryConsumeKW(keyword.Alias); !ok {
		if lhs.Type == tokenizer.STRING {
			return "", p.stream.SliceInputEnclosingTokens(lhs), nil
		}
		return "", lhs.Literal, nil
	}
	rhs, err := p.parseContinerIdent(p.stream.Emit())
	if err != nil {
		return "", "", err
	}
	lhsStr := lhs.Literal
	if lhs.Type == tokenizer.STRING {
		lhsStr = p.stream.SliceInputEnclosingTokens(lhs)
	}
	rhsStr := rhs.Literal
	if rhs.Type == tokenizer.STRING {
		rhsStr = p.stream.SliceInputEnclosingTokens(rhs)
	}
	if lhs.Type == rhs.Type {
		return lhsStr, rhsStr, nil
	}
	switch lhs.Type {
	case tokenizer.STRING:
		return lhsStr, rhsStr, nil
	case tokenizer.IDENTIFIER:
		return rhsStr, lhsStr, nil
	}
	return "", "", NewParserError(lhs, "Invalid container alias and identifier combination")
}

