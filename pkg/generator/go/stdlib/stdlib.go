package stdlib

//go:generate go run yur4uwe/pac/cmd/go-stdlib-gen

import (
	"path"
	"strings"

	"yur4uwe/pac/pkg/resolver"
)

var (
	stdPackages map[string]struct{}
	shortToPath map[string]string
)

var canonicalOverrides = map[string]string{
	"rand":     "crypto/rand",
	"template": "text/template",
	"pprof":    "runtime/pprof",
	"trace":    "runtime/trace",
}

// BuiltinTypes is the set of Go predefined identifiers that require no import and should not be emitted.
var BuiltinTypes = map[string]struct{}{
	"any":        {},
	"bool":       {},
	"byte":       {},
	"comparable": {},
	"complex64":  {},
	"complex128": {},
	"error":      {},
	"float32":    {},
	"float64":    {},
	"int":        {},
	"int8":       {},
	"int16":      {},
	"int32":      {},
	"int64":      {},
	"rune":       {},
	"string":     {},
	"uint":       {},
	"uint8":      {},
	"uint16":     {},
	"uint32":     {},
	"uint64":     {},
	"uintptr":    {},
}

func init() {
	stdPackages = make(map[string]struct{}, len(stdlibPackageList))
	shortToPath = make(map[string]string, len(stdlibPackageList))

	for _, pkgPath := range stdlibPackageList {
		stdPackages[pkgPath] = struct{}{}
		pkgName := path.Base(pkgPath)

		if override, ok := canonicalOverrides[pkgName]; ok {
			shortToPath[pkgName] = override
		} else if _, exists := shortToPath[pkgName]; !exists {
			shortToPath[pkgName] = pkgPath
		}
	}
}

func IsBuiltinType(name string) bool {
	_, ok := BuiltinTypes[name]
	return ok
}

// LookupImportPath returns full standard library import path for a package identifier or path
func LookupImportPath(pkgNameOrPath []string) (string, bool) {
	if len(pkgNameOrPath) == 0 {
		return "", false
	}

	pkgStr := strings.Join(pkgNameOrPath, "/")

	// Exact match on full import path
	if _, ok := stdPackages[pkgStr]; ok {
		return pkgStr, true
	}
	// Match on short package name (e.g. "http" -> "net/http", "json" -> "encoding/json")
	if path, ok := shortToPath[pkgStr]; ok {
		return path, true
	}
	return "", false
}

// IsStdlibPackage checks if the name or path belongs to the Go standard library
func IsStdlibPackage(pkgNameOrPath []string) bool {
	if len(pkgNameOrPath) == 0 {
		return false
	}

	pkgStr := strings.Join(pkgNameOrPath, "/")

	if _, ok := stdPackages[pkgStr]; ok {
		return true
	}
	if _, ok := shortToPath[pkgStr]; ok {
		return true
	}
	return false
}

// IsStdlibEntity checks if an entity symbol is a Go builtin or standard library type
func IsStdlibEntity(ent *resolver.EntitySymbol) bool {
	if ent == nil {
		return false
	}

	name := ""
	if ent.AST != nil {
		name = ent.AST.Identifier.EntityName()
	}
	if name == "" {
		name = SimpleName(ent.FQN)
	}

	// check for builtin types without package (e.g. error, any, string)
	if len(ent.PackagePath) == 0 && IsBuiltinType(name) {
		return true
	}

	// 2. Package path matches standard library (e.g. ["time"], ["net", "http"])
	if len(ent.PackagePath) > 0 {
		if IsStdlibPackage(ent.PackagePath) {
			return true
		}
	}

	// 3. FQN starts with standard library package name (e.g. "time.Time", "http.Request")
	if strings.Contains(ent.FQN, ".") {
		parts := strings.Split(ent.FQN, ".")
		// Check first segment (e.g. "time" from "time.Time")
		if IsStdlibPackage(parts[:1]) {
			return true
		}
		// Check joined segments except last (e.g. "net/http" from "net.http.Request")
		if IsStdlibPackage(parts[:len(parts)-1]) {
			return true
		}
	}

	return false
}

func SimpleName(fqn string) string {
	if idx := strings.LastIndex(fqn, "."); idx != -1 {
		return fqn[idx+1:]
	}
	return fqn
}
