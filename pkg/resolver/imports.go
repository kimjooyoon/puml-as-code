// Package resolver implements different resolvers, like the import resolver or
// symbol resolver.
package resolver

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"yur4uwe/pac/pkg/parser"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"
)

const (
	MaxDepth = 15
)

var (
	ErrIncludeCycle               = errors.New("include cycle detected")
	ErrMaxDepthExceeded           = errors.New("include depth exceeded")
	ErrStdlibNotImplemented       = errors.New("stdlib includes (<...>) are not implemented")
	ErrRemoteIncludeUnimplemented = errors.New("remote URL includes (http/https) are not implemented")
)

func resolveIncludes(diagram *ast.Diagram, basePath string, fs FileReader, lang string, includeStack []string, includedOnce map[string]struct{}) error {
	if len(includeStack) > MaxDepth {
		return fmt.Errorf("%w: %d", ErrMaxDepthExceeded, len(includeStack))
	}

	newStmts := make([]ast.Statement, 0, len(diagram.Statements))
	targetDir := filepath.Dir(basePath)
	for _, stmt := range diagram.Statements {
		var v *ast.IncludeDirective
		switch d := stmt.(type) {
		case *ast.Container:
			// make a contrived diagram to satisfy 'resolveIncludes'
			auxDiagram := &ast.Diagram{
				Statements: d.Statements,
			}
			err := resolveIncludes(auxDiagram, basePath, fs, lang, includeStack, includedOnce)
			if err != nil {
				return fmt.Errorf("failed to resolve container imports: %w", err)
			}
			d.Statements = auxDiagram.Statements
			newStmts = append(newStmts, d)
			continue
		case *ast.IncludeDirective:
			v = d
		default:
			newStmts = append(newStmts, stmt)
			continue
		}

		if strings.HasPrefix(v.Path, "<") && strings.HasSuffix(v.Path, ">") {
			return fmt.Errorf("%w: %s", ErrStdlibNotImplemented, v.Path)
		}
		if strings.HasPrefix(v.Path, "http://") || strings.HasPrefix(v.Path, "https://") {
			return fmt.Errorf("%w: %s", ErrRemoteIncludeUnimplemented, v.Path)
		}

		// Resolve included diagrams and splice them into the current diagram
		// 1. Resolve the path
		// 2. Read the file
		// 3. Parse the file
		// 4. Splice the statements into the current diagram
		targetPath := filepath.Join(targetDir, v.Path)
		targetPath = filepath.Clean(targetPath)
		diagramIdent := targetPath
		if v.Tag != "" {
			diagramIdent += "!" + v.Tag
		}

		if slices.Contains(includeStack, diagramIdent) {
			return fmt.Errorf("%w: %s", ErrIncludeCycle, diagramIdent)
		}

		if v.Kind == ast.IncludeOnce {
			if _, ok := includedOnce[diagramIdent]; ok {
				continue
			}
			includedOnce[diagramIdent] = struct{}{}
		}

		childStack := append(slices.Clone(includeStack), diagramIdent)

		fileBytes, err := fs.ReadFile(targetPath)
		if err != nil {
			return err
		}

		isBoundless := v.Tag == ""
		p := parser.NewParser(dialect.Factory(lang)).
			WithTargetID(v.Tag)
		p.IsBoundless = isBoundless
		diag, err := p.Parse(string(fileBytes))
		if err != nil && !errors.Is(err, tokenizer.ErrUnexpectedEOF) {
			return err
		}

		err = resolveIncludes(diag, targetPath, fs, lang, childStack, includedOnce)
		if err != nil {
			return err
		}

		// Skip empty diagrams
		// using <= 2 may be a bit of an overkill
		// BUT its better to be safe than sorry
		if !isBoundless && len(diag.Statements) <= 2 {
			continue
		}

		// Strip diagram bound markers
		if _, ok := diag.Statements[0].StatementNode().(*ast.DiagramBound); ok {
			diag.Statements = diag.Statements[1:]
		}
		if _, ok := diag.Statements[len(diag.Statements)-1].StatementNode().(*ast.DiagramBound); ok {
			diag.Statements = diag.Statements[:len(diag.Statements)-1]
		}

		newStmts = append(newStmts, diag.Statements...)
	}
	diagram.Statements = newStmts
	return nil
}

func ResolveImports(diagram *ast.Diagram, basePath string, tag string, fs FileReader, lang string) error {
	rootIdent := filepath.Clean(basePath)
	if tag != "" {
		rootIdent += "!" + tag
	}
	includeStack := []string{rootIdent}
	includedOnce := map[string]struct{}{
		rootIdent: {},
	}
	return resolveIncludes(diagram, basePath, fs, lang, includeStack, includedOnce)
}
