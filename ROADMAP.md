# Tooling Roadmap: Formatter, Highlighting & LSP

High-level implementation roadmap for extending `pac` into formatting, syntax highlighting, and editor tooling.

---

## Track 1: PlantUML Formatter (`pac fmt`)

Goal: Canonical, idempotent pretty-printer for PUML diagrams.

- [x] **1. Lossless AST & Trivia Hardening**
  - Add `LaxDialect` (`pkg/parser/dialect/lax.go`): default dialect for formatting that captures arbitrary/sketch member syntax into `LaxField` / `LaxMethod` with raw tokens instead of rejecting non-Go syntax.
  - Add fallback node (`ast.UnhandledStatement`) to pass unhandled diagram lines without failing.
  - *Trivia normalization rule:* Mid-line comments (e.g. `A /' note '/ --> B`) are normalized to line-trailing comments. Idempotence is preserved: `Format(Format(S)) == Format(S)`.
- [x] **2. AST Pretty-Printer (`pkg/formatter`)**
  - Canonical indentation (2 spaces) for blocks (`package`, `class`, `interface`).
  - Standardize relationship spacing (`Foo "1" *-- "0..*" Bar : label`).
  - Render leading/trailing comments from [`ast.Trivia`](file:///home/yur4uwe/Projects/puml-as-code/pkg/parser/ast/structs.go#L20-L27).
- [x] **3. Quality & Invariants**
  - **Idempotence Test:** `Format(Format(src)) == Format(src)`.
  - **AST Stability Test:** `Parse(src) == Parse(Format(src))`.
- [x] **4. CLI Integration (`cmd/pac`)**
  - Flags: `pac fmt <file.puml>` (stdout), `-w` (in-place write), `-check` (CI diff check).

## Track 3: The Unified Bridge (Language Server Protocol)

Goal: Connect the Go compiler/formatter directly to editors via `pac lsp`.

- [ ] **1. LSP Core (`cmd/pac-lsp` or `pac lsp`)**
  - JSON-RPC 2.0 loop over stdin/stdout.
- [ ] **2. Diagnostics (`textDocument/publishDiagnostics`)**
  - Emit parse errors from [`parser.Parser`](file:///home/yur4uwe/Projects/puml-as-code/pkg/parser/parser.go#L15-L21) as editor squiggles.
- [ ] **3. In-Editor Formatting (`textDocument/formatting`)**
  - Delegate formatting requests directly to `pkg/formatter`.
- [ ] **4. Semantic Highlighting (`textDocument/semanticTokens`)**
  - Use symbol tables from [`resolver.ResolveSymbols`](file:///home/yur4uwe/Projects/puml-as-code/pkg/resolver/symbols.go) to provide context-aware highlighting (e.g. resolved vs unresolved types).
