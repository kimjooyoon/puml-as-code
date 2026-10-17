# AGENTS.md

## 1. Project Overview & Architecture

### Purpose

Command-line compiler/transpiler written in Go that parses PlantUML Class Diagrams and emits formatted, idiomatic source code (starting with Go, structured for multi-target language extensibility).

### Core Pipeline Flow

  1. **Lexical Analysis:** Hand-written lexer in [`pkg/tokenizer`](file:///home/yur4uwe/Projects/puml-as-code/pkg/tokenizer) converts raw PUML into token streams.
  2. **Parsing & AST Construction:** Hand-written recursive descent parser in [`pkg/parser`](file:///home/yur4uwe/Projects/puml-as-code/pkg/parser) builds language-agnostic AST nodes defined in [`pkg/parser/ast`](file:///home/yur4uwe/Projects/puml-as-code/pkg/parser/ast).
  3. **Include Resolution:** Post-parse preprocessing in [`pkg/resolver`](file:///home/yur4uwe/Projects/puml-as-code/pkg/resolver) evaluates `!include` directives and resolves file dependencies/diagram tags.
  4. **Code Generation:** Template-based generator in [`pkg/generator`](file:///home/yur4uwe/Projects/puml-as-code/pkg/generator) (e.g. Go backend in [`pkg/generator/go`](file:///home/yur4uwe/Projects/puml-as-code/pkg/generator/go)) performs semantic validation, populates view models, renders templates, and runs target language formatters (`go/format`).
* **Key Invariants & Guardrails:**
  * **Parser I/O Isolation:** The parser has no direct filesystem access; all external file inclusion is deferred to the [`resolver`](file:///home/yur4uwe/Projects/puml-as-code/pkg/resolver) pass.
  * **AST Node Semantics:** Preprocessor directives are classified as `*Directive` (e.g. `IncludeDirective`), whereas diagram render/visibility instructions are classified as `*Command` (e.g. `VisibilityCommand`, `DirectionCommand`).

---

## 2. Codebase & Package Structure

```
puml-as-code/
+-- cmd/
|   +-- main.go                       # Root CLI entry point
|   +-- docs-examples-gen/            # Utility tool to generate documentation examples
+-- internal/
|   +-- helpers/                      # Shared internal helper functions
+-- pkg/
|   +-- tokenizer/                    # Lexical scanner & token definitions
|   +-- parser/                       # Recursive-descent parser
|   |   +-- ast/                      # AST node definitions & enums
|   |   +-- dialect/                  # Target-language syntax parsing for members
|   |   +-- keyword/                  # Keyword tables & classification
|   +-- resolver/                     # Include directives & symbol resolver pass
|   +-- generator/                    # Generator factory & interface
|       +-- go/                       # Go code generator & templates
+-- input/
|   +-- integration_testdata/         # Test diagrams & golden fixtures
+-- output/                           # Target directory for generated output
+-- docs/                             # Language specs & dialect documentation
```

### Inspecting Package APIs
When detailed information is needed about package responsibilities, types, or exported functions, inspect the package doc comments and API dynamically using `go doc`:
```bash
go doc ./pkg/<package-path>
# Examples:
go doc ./pkg/parser
go doc ./pkg/parser/ast
go doc ./pkg/generator/go
```

## 3. Navigating the Codebase

### Current Backlog

Full list of tasks is available in [`plan.md`](file:///home/yur4uwe/Projects/puml-as-code/plan.md).

## 4. Rules

### 1. Implicit Containers

Implicit containers are entities or containers that are not explicitly declared in the source code via a declaration syntax, but whose existance implied by other explicit declarations. E.g.:

```puml
class pkg.Foo
```

In this snippet package `pkg` is implicitly declared as a container. In AST such containers only have their name as it is the only inferrable thing. They may become explicit containers under resolution pass which unifies all declarations, but after that point the concepts of AST aren't applicable.

### 2. Stereotype

There can be multiple stereotypes in a single entity/container declaration. Everything inside <<...>> is a single stereotype. Commas are valid but they break the `<style>...</style>` styling because it thinks of `.Stereo1, Stereo2 {...}` as 2 different selectors for the same style.
