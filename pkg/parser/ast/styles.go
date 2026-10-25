package ast

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"yur4uwe/pac/pkg/tokenizer"
)

// StyleDeclaration represents a single CSS property declaration within a style rule:
//
//	FontColor: #333333;
//	BackGroundColor: PaleGreen;
type StyleDeclaration struct {
	BaseNode
	Property string
	Value    string
}

// StatementNode implements [Statement].
func (d *StyleDeclaration) StatementNode() Statement { return d }

var _ Statement = (*StyleDeclaration)(nil)

// StyleRule represents a scoped CSS rule within a <style> block:
//
//	classDiagram {
//	    class { ... }
//	}
type StyleRule struct {
	BaseNode
	Selectors  []string      `json:",omitempty"`
	LBrace     tokenizer.Pos // Zero if bodyless declaration
	Statements []Statement   `json:",omitempty"` // StyleDeclaration, nested StyleRule, or UnhandledStatement
	RBrace     tokenizer.Pos // Zero if unclosed before EOF
}

func (s *StyleRule) BlockEnd() tokenizer.Pos   { return s.RBrace }
func (s *StyleRule) BlockStart() tokenizer.Pos { return s.LBrace }

func (s *StyleRule) HasBody() bool {
	return s.LBrace != tokenizer.Pos{}
}

func (s *StyleRule) IsClosed() bool {
	return !s.HasBody() || s.RBrace != tokenizer.Pos{}
}

// StatementNode implements [Statement].
func (s *StyleRule) StatementNode() Statement { return s }

var (
	_ Statement = (*StyleRule)(nil)
	_ Block     = (*StyleRule)(nil)
)

// StyleBlock represents the top-level <style> ... </style> block container.
type StyleBlock struct {
	BaseNode
	RAngle     tokenizer.Pos // Zero if bodyless declaration
	Statements []Statement   `json:",omitempty"` // StyleRule or UnhandledStatement
	LAngle     tokenizer.Pos // Zero if unclosed before EOF
}

func (b *StyleBlock) BlockEnd() tokenizer.Pos   { return b.RAngle }
func (b *StyleBlock) BlockStart() tokenizer.Pos { return b.LAngle }

func (b *StyleBlock) HasBody() bool {
	return b.LAngle != tokenizer.Pos{}
}

func (b *StyleBlock) IsClosed() bool {
	return !b.HasBody() || b.RAngle != tokenizer.Pos{}
}

// StatementNode implements [Statement].
func (b *StyleBlock) StatementNode() Statement { return b }

var (
	_ Statement = (*StyleBlock)(nil)
	_ Block     = (*StyleBlock)(nil)
)

// TokensToKey concatenates the literals of a token slice into a string.
func TokensToKey(tokens []tokenizer.Token) string {
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) == 1 {
		return tokens[0].Literal
	}
	var sb strings.Builder
	for _, tok := range tokens {
		sb.WriteString(tok.Literal)
	}
	return sb.String()
}

// SkinparamSetting represents a single-line skinparam setting:
//
//	skinparam backgroundColor #EEEBDC
//	skinparam shadowing<<no_shadow>> false
type SkinparamSetting struct {
	BaseNode
	Keyword    string      `json:",omitempty"` // "skinparam" or "skinparamlocked"
	Name       Ident       // Target name identifier
	Stereotype *Stereotype `json:",omitempty"` // Extracted stereotype if present
	Value      string      `json:",omitempty"` // Sliced verbatim from source (preserves quotes, backslashes)
}

// StatementNode implements [Statement].
func (s *SkinparamSetting) StatementNode() Statement { return s }

// EffectiveKey returns the full name of a single-line setting.
func (s SkinparamSetting) EffectiveKey() string { return s.Name.Literal }

var _ Statement = (*SkinparamSetting)(nil)

// SkinparamBlock represents a skinparam block (top-level or nested):
//
//	skinparam class {
//	    BackgroundColor PaleGreen
//	    header {
//	        FontSize 12
//	    }
//	}
type SkinparamBlock struct {
	BaseNode
	Keyword    string        `json:",omitempty"` // "skinparam" (empty for nested blocks)
	Name       Ident         // Block target name identifier, empty for nameless "skinparam { ... }"
	Stereotype *Stereotype   `json:",omitempty"` // Extracted stereotype if present
	LBrace     tokenizer.Pos // Zero if bodyless declaration
	Statements []Statement   `json:",omitempty"` // SkinparamSetting, nested SkinparamBlock, or UnhandledStatement
	RBrace     tokenizer.Pos // Zero if unclosed before EOF
}

func (b *SkinparamBlock) BlockEnd() tokenizer.Pos   { return b.RBrace }
func (b *SkinparamBlock) BlockStart() tokenizer.Pos { return b.LBrace }

func (b *SkinparamBlock) HasBody() bool {
	return b.LBrace != tokenizer.Pos{}
}

func (b *SkinparamBlock) IsClosed() bool {
	return !b.HasBody() || b.RBrace != tokenizer.Pos{}
}

// StatementNode implements [Statement].
func (b *SkinparamBlock) StatementNode() Statement { return b }

var (
	_ Statement = (*SkinparamBlock)(nil)
	_ Block     = (*SkinparamBlock)(nil)
)

// ResolvedSkinparam is one effective setting with its concatenated key,
// used by code generators and semantic passes.
type ResolvedSkinparam struct {
	Key   string   // Concatenated key (e.g. "class.header.FontSize")
	Value string   // Value of the setting
	Path  []string // Hierarchical path segments
}

// Settings flattens the block into effective settings in source order.
func (b SkinparamBlock) Settings() []ResolvedSkinparam {
	return flattenSkinparam(b.Name.Literal, b.Statements)
}

func flattenSkinparam(prefix string, stmts []Statement) []ResolvedSkinparam {
	var out []ResolvedSkinparam
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *SkinparamSetting:
			key := s.EffectiveKey()
			if prefix != "" {
				key = prefix + "." + key
			}
			out = append(out, ResolvedSkinparam{
				Key:   key,
				Value: s.Value,
				Path:  strings.Split(key, "."),
			})
		case *SkinparamBlock:
			blockName := s.Name.Literal
			nextPrefix := blockName
			if prefix != "" && blockName != "" {
				nextPrefix = prefix + "." + blockName
			} else if prefix != "" {
				nextPrefix = prefix
			}
			out = append(out, flattenSkinparam(nextPrefix, s.Statements)...)
		}
	}
	return out
}

type SkinparamValueType int

const (
	SkinparamString SkinparamValueType = iota
	SkinparamColor
	SkinparamInt
	SkinparamBool
	SkinparamEnum
)

func (vt SkinparamValueType) Validate(value string) error {
	switch vt {
	case SkinparamColor:
		if !isValidColor(value) {
			return fmt.Errorf("invalid color: %s", value)
		}
	case SkinparamInt:
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
	case SkinparamBool:
		v := strings.ToLower(value)
		if v != "true" && v != "false" && v != "on" && v != "off" && v != "yes" && v != "no" {
			return fmt.Errorf("invalid boolean: %s", value)
		}
	}
	return nil
}

func isValidColor(color string) bool {
	color = strings.TrimSpace(strings.ToLower(color))
	if color == "" || color == "transparent" {
		return true
	}

	if strings.HasPrefix(color, "#") {
		hex := color[1:]

		switch len(hex) {
		case 3, 6, 8:
			break
		default:
			return false
		}

		for _, r := range hex {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				return false
			}
		}
		return true
	}

	if strings.HasPrefix(color, "rgb") {
		start := strings.Index(color, "(")
		end := strings.LastIndex(color, ")")
		if start == -1 || end == -1 || end < start {
			return false
		}

		strs := strings.Split(color[start+1:end], ",")
		isRGBA := strings.HasPrefix(color, "rgba")

		if (isRGBA && len(strs) != 4) || (!isRGBA && len(strs) != 3) {
			return false
		}

		for i, s := range strs {
			val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return false
			}
			if isRGBA && i == 3 {
				if val < 0 || val > 1 {
					return false
				}
				continue
			}
			if val < 0 || val > 255 {
				return false
			}
		}
		return true
	}

	return isValidNamedColor(color)
}

func isValidNamedColor(color string) bool {
	// Standard PlantUML colors are quite extensive.
	// For now we accept basic ones or common patterns.
	basicColors := []string{"red", "green", "blue", "white", "black", "grey", "gray", "yellow", "cyan", "magenta", "orange", "pink", "brown"}
	return slices.Contains(basicColors, strings.ToLower(color))
}

type SkinparamKey struct {
	MainTarget string
	SubTarget  string
	Param      string
	Stereotype string
}

func (k SkinparamKey) String() string {
	return fmt.Sprintf("%s.%s.%s.%s", k.MainTarget, k.SubTarget, k.Param, k.Stereotype)
}

func (k SkinparamKey) Lower() SkinparamKey {
	return SkinparamKey{
		MainTarget: strings.ToLower(k.MainTarget),
		SubTarget:  strings.ToLower(k.SubTarget),
		Param:      strings.ToLower(k.Param),
		Stereotype: strings.ToLower(k.Stereotype),
	}
}

type Skinparam map[string]string

var (
	// Base Property Groups
	fontProps = []string{"fontname", "fontsize", "fontstyle", "fontcolor"}
	bgBorder  = []string{"backgroundcolor", "bordercolor"}

	// Composite Groups
	stdNode       = slices.Concat(fontProps, bgBorder)
	stdStereoNode = slices.Concat(stdNode, []string{"stereotype"})
	boxNode       = slices.Concat(stdNode, []string{"borderthickness"})
	stdBoxNode    = slices.Concat(boxNode, []string{"stereotype"})

	paramRegistry = map[string]SkinparamValueType{
		"backgroundcolor":    SkinparamColor,
		"bordercolor":        SkinparamColor,
		"borderthickness":    SkinparamInt,
		"fontcolor":          SkinparamColor,
		"fontname":           SkinparamString,
		"fontsize":           SkinparamInt,
		"fontstyle":          SkinparamString,
		"color":              SkinparamColor,
		"roundcorner":        SkinparamInt,
		"shadowing":          SkinparamBool,
		"iconsize":           SkinparamInt,
		"padding":            SkinparamInt,
		"style":              SkinparamString,
		"textalignment":      SkinparamString,
		"sep":                SkinparamInt,
		"position":           SkinparamString,
		"size":               SkinparamInt,
		"radius":             SkinparamInt,
		"thickness":          SkinparamInt,
		"margin":             SkinparamInt,
		"wraptitlewidth":     SkinparamInt,
		"width":              SkinparamInt,
		"underline":          SkinparamBool,
		"barcolor":           SkinparamColor,
		"externalcolor":      SkinparamColor,
		"alignment":          SkinparamString,
		"titlealignment":     SkinparamString,
		"messagealignment":   SkinparamString,
		"monospacedfontname": SkinparamString,
	}

	subTargetRegistry = map[string][]string{
		"stereotype":       fontProps,
		"attribute":        slices.Concat(fontProps, []string{"iconsize"}),
		"header":           {"backgroundcolor"},
		"footer":           fontProps,
		"title":            fontProps,
		"diamond":          slices.Concat(fontProps, bgBorder),
		"end":              {"color"},
		"start":            {"color"},
		"arrow":            {"thickness"},
		"lollipop":         {"color"},
		"iemandatory":      {"color"},
		"private":          slices.Concat(bgBorder, []string{"color"}),
		"public":           slices.Concat(bgBorder, []string{"color"}),
		"protected":        slices.Concat(bgBorder, []string{"color"}),
		"package":          slices.Concat(bgBorder, []string{"color"}),
		"box":              stdNode,
		"delay":            fontProps,
		"divider":          boxNode,
		"group":            boxNode,
		"groupheader":      fontProps,
		"lifeline":         slices.Concat(bgBorder, []string{"borderthickness"}),
		"message":          {"alignment", "textalignment"},
		"newpageseparator": {"color"},
		"reference":        slices.Concat(boxNode, []string{"alignment"}),
		"referenceheader":  {"backgroundcolor"},
		"participant":      {"borderthickness"},
	}

	mainTargetRegistry = map[string][]string{
		"activity":         slices.Concat(stdNode, []string{"barcolor", "borderthickness", "diamond", "end", "start"}),
		"actor":            stdStereoNode,
		"agent":            stdBoxNode,
		"arrow":            slices.Concat(fontProps, []string{"color", "lollipop", "messagealignment", "thickness"}),
		"artifact":         stdStereoNode,
		"biddable":         bgBorder,
		"boundary":         stdStereoNode,
		"box":              {"padding"},
		"caption":          fontProps,
		"card":             stdBoxNode,
		"circledcharacter": slices.Concat(fontProps, []string{"radius"}),
		"class":            slices.Concat(stdBoxNode, []string{"attribute", "header"}),
		"cloud":            stdStereoNode,
		"collections":      bgBorder,
		"component":        slices.Concat(stdBoxNode, []string{"style"}),
		"control":          stdStereoNode,
		"database":         stdStereoNode,
		"default":          slices.Concat(fontProps, []string{"monospacedfontname", "textalignment"}),
		"designeddomain":   slices.Concat(fontProps, []string{"borderthickness", "stereotype"}),
		"diagram":          {"bordercolor", "borderthickness"},
		"domain":           stdBoxNode,
		"entity":           stdStereoNode,
		"file":             stdStereoNode,
		"folder":           stdStereoNode,
		"footer":           fontProps,
		"frame":            stdStereoNode,
		"header":           fontProps,
		"hyperlink":        {"color", "underline"},
		"icon":             {"iemandatory", "package", "private", "protected", "public"},
		"interface":        stdStereoNode,
		"legend":           boxNode,
		"lexical":          bgBorder,
		"machine":          stdBoxNode,
		"node":             slices.Concat(stdStereoNode, []string{"sep"}),
		"note":             slices.Concat(bgBorder, []string{"borderthickness", "textalignment", "shadowing"}),
		"object":           slices.Concat(stdBoxNode, []string{"attribute"}),
		"package":          slices.Concat(stdBoxNode, []string{"style", "titlealignment"}),
		"page":             slices.Concat(fontProps, []string{"bordercolor", "margin", "externalcolor"}),
		"participant":      slices.Concat(stdStereoNode, []string{"padding"}),
		"partition":        boxNode,
		"queue":            stdStereoNode,
		"rank":             {"sep"},
		"rectangle":        stdBoxNode,
		"requirement":      stdBoxNode,
		"sequence":         {"actor", "arrow", "box", "divider", "group", "groupheader", "lifeline", "participant"},
		"stack":            stdStereoNode,
		"state":            slices.Concat(stdNode, []string{"attribute", "end", "start"}),
		"stereotype":       {"a", "c", "e", "i", "n", "position"},
		"storage":          stdStereoNode,
		"swimlane":         slices.Concat(boxNode, []string{"stereotype", "title", "width", "wraptitlewidth"}),
		"tab":              {"size"},
		"title":            slices.Concat(boxNode, []string{"borderroundcorner"}),
		"usecase":          stdBoxNode,
		"wrap":             {"width"},
	}

	globalRegistry = map[string]SkinparamValueType{
		"colorarrowseparationspace": SkinparamInt,
		"dpi":                       SkinparamInt,
		"genericdisplay":            SkinparamString,
		"guillement":                SkinparamBool,
		"handwritten":               SkinparamBool,
		"linetype":                  SkinparamString,
		"maxasciimessagelength":     SkinparamInt,
		"maxmessagesize":            SkinparamInt,
		"minclasswidth":             SkinparamInt,
		"monochrome":                SkinparamBool,
		"pathhovercolor":            SkinparamColor,
		"responsemessagebelowarrow": SkinparamBool,
		"sameclasswidth":            SkinparamBool,
		"svglinktarget":             SkinparamString,
	}
)

func (s Skinparam) Set(key SkinparamKey, value string) error {
	lk := key.Lower()
	if lk.Param == "" {
		return fmt.Errorf("param cannot be empty")
	}

	var allowed []string
	var ok bool
	if lk.MainTarget != "" {
		allowed, ok = mainTargetRegistry[lk.MainTarget]
		if !ok {
			return fmt.Errorf("unknown main target: %s", key.MainTarget)
		}
	}

	if lk.MainTarget != "" && lk.SubTarget != "" {
		if !slices.Contains(allowed, lk.SubTarget) {
			return fmt.Errorf("sub-target %s not allowed for main target %s", key.SubTarget, key.MainTarget)
		}

		allowed, ok = subTargetRegistry[lk.SubTarget]
		if !ok {
			panic(fmt.Sprintf("unknown sub-target: %s", lk.SubTarget))
		}
	}

	if len(allowed) != 0 && !slices.Contains(allowed, lk.Param) {
		return fmt.Errorf("param %s not allowed for target \"%s.%s\"", lk.Param, lk.MainTarget, lk.SubTarget)
	}

	valueType, ok := paramRegistry[lk.Param]
	if !ok {
		valueType, ok = globalRegistry[lk.Param]
		if !ok {
			return fmt.Errorf("unknown param: %s", key.Param)
		}
	}

	if err := valueType.Validate(value); err != nil {
		return err
	}

	s[lk.String()] = value
	return nil
}

func (s Skinparam) Get(key SkinparamKey) string {
	lk := key.Lower()

	// 1. Most specific: Main + Sub + Param + Stereo
	if val, ok := s[lk.String()]; ok {
		return val
	}
	// 2. Main + Sub + Param
	if val, ok := s[fmt.Sprintf("%s.%s.%s.", lk.MainTarget, lk.SubTarget, lk.Param)]; ok {
		return val
	}
	// 3. Main + Param + Stereo
	if val, ok := s[fmt.Sprintf("%s..%s.%s", lk.MainTarget, lk.Param, lk.Stereotype)]; ok {
		return val
	}
	// 4. Main + Param
	if val, ok := s[fmt.Sprintf("%s..%s.", lk.MainTarget, lk.Param)]; ok {
		return val
	}
	// 5. Sub + Param
	if val, ok := s[fmt.Sprintf(".%s.%s.", lk.SubTarget, lk.Param)]; ok {
		return val
	}
	// 6. Global Param
	if val, ok := s[fmt.Sprintf("..%s.", lk.Param)]; ok {
		return val
	}

	return ""
}

func (s Skinparam) SetAndDecodeWithContext(context SkinparamKey, combinedName, value string) error {
	name := strings.ToLower(combinedName)

	// If context doesn't have a MainTarget, try to find it in name
	if context.MainTarget == "" {
		for t := range mainTargetRegistry {
			if !strings.HasPrefix(name, t) {
				continue
			}
			if len(t) <= len(context.MainTarget) {
				continue
			}
			context.MainTarget = t
		}
		if context.MainTarget != "" {
			name = name[len(context.MainTarget):]
		}
	}

	// If context doesn't have a SubTarget, try to find it in name
	if context.SubTarget == "" {
		for st := range subTargetRegistry {
			if strings.HasPrefix(name, st) {
				context.SubTarget = st
				name = name[len(st):]
				break
			}
		}
	}

	context.Param = name
	if context.Param == "" && context.SubTarget == "" && context.MainTarget == "" {
		return fmt.Errorf("could not decode skinparam name: %s", combinedName)
	}

	return s.Set(context, value)
}
