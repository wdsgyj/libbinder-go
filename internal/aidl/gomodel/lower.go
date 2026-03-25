package gomodel

import (
	"fmt"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
)

type LowerOptions struct {
	SourcePath        string
	CustomParcelables map[string]CustomParcelableConfig
	StableInterfaces  map[string]StableInterfaceConfig
	DependencyFiles   []*ast.File
}

type namedKind string

const (
	namedInterface  namedKind = "interface"
	namedParcelable namedKind = "parcelable"
	namedEnum       namedKind = "enum"
	namedUnion      namedKind = "union"
)

type namedSymbol struct {
	kind        namedKind
	aidlName    string
	goName      string
	scope       *scope
	decl        any
	declaredIn  *scope
	packageName string
}

type scope struct {
	parent  *scope
	prefix  string
	symbols map[string]*namedSymbol
}

func Lower(file *ast.File, opts LowerOptions) (*File, []Diagnostic) {
	if file == nil {
		return nil, []Diagnostic{{Message: "nil file"}}
	}

	goPkg := defaultGoPackageName(file.PackageName, opts.SourcePath)
	out := &File{
		SourcePath:  opts.SourcePath,
		AIDLPackage: file.PackageName,
		GoPackage:   goPkg,
		Imports:     append([]ast.Import(nil), file.Imports...),
	}

	state := lowerState{
		file:              file,
		out:               out,
		byQName:           map[string]*namedSymbol{},
		customParcelables: opts.CustomParcelables,
		stableInterfaces:  opts.StableInterfaces,
	}
	state.root = &scope{symbols: map[string]*namedSymbol{}}

	state.collectTopLevel(file)
	for _, dep := range opts.DependencyFiles {
		state.collectTopLevel(dep)
	}
	for _, decl := range file.Decls {
		state.lowerDecl(state.root, nil, decl)
	}

	return out, state.diags
}

type lowerState struct {
	file              *ast.File
	out               *File
	root              *scope
	diags             []Diagnostic
	byQName           map[string]*namedSymbol
	customParcelables map[string]CustomParcelableConfig
	stableInterfaces  map[string]StableInterfaceConfig
}

func (s *lowerState) collectTopLevel(file *ast.File) {
	if file == nil {
		return
	}
	for _, decl := range file.Decls {
		s.collectDecl(file, s.root, nil, decl)
	}
}

func (s *lowerState) collectDecl(file *ast.File, current *scope, owner *namedSymbol, decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.InterfaceDecl:
		sym := s.registerDecl(file, current, owner, d.Name, namedInterface, d)
		nestedScope := &scope{
			parent:  current,
			prefix:  sym.aidlName,
			symbols: map[string]*namedSymbol{},
		}
		sym.scope = nestedScope
		for _, member := range d.Members {
			switch nested := member.(type) {
			case *ast.ParcelableDecl:
				s.collectNestedDecl(file, nestedScope, sym, nested)
			case *ast.EnumDecl:
				s.collectNestedDecl(file, nestedScope, sym, nested)
			case *ast.UnionDecl:
				s.collectNestedDecl(file, nestedScope, sym, nested)
			}
		}
	case *ast.ParcelableDecl:
		s.registerDecl(file, current, owner, d.Name, namedParcelable, d)
	case *ast.EnumDecl:
		s.registerDecl(file, current, owner, d.Name, namedEnum, d)
	case *ast.UnionDecl:
		s.registerDecl(file, current, owner, d.Name, namedUnion, d)
	}
}

func (s *lowerState) collectNestedDecl(file *ast.File, current *scope, owner *namedSymbol, decl ast.InterfaceMember) {
	switch d := decl.(type) {
	case *ast.ParcelableDecl:
		s.registerDecl(file, current, owner, d.Name, namedParcelable, d)
	case *ast.EnumDecl:
		s.registerDecl(file, current, owner, d.Name, namedEnum, d)
	case *ast.UnionDecl:
		s.registerDecl(file, current, owner, d.Name, namedUnion, d)
	}
}

func (s *lowerState) registerDecl(file *ast.File, current *scope, owner *namedSymbol, name string, kind namedKind, decl any) *namedSymbol {
	qualified := name
	if owner != nil {
		qualified = owner.aidlName + "." + name
	}
	if file != nil && file.PackageName != "" {
		qualified = file.PackageName + "." + qualified
	}

	goName := exportName(name)
	if owner != nil {
		goName = owner.goName + exportName(name)
	}

	sym := &namedSymbol{
		kind:        kind,
		aidlName:    qualified,
		goName:      goName,
		decl:        decl,
		declaredIn:  current,
		packageName: file.PackageName,
	}
	if current != nil && (current != s.root || owner != nil || file.PackageName == s.file.PackageName) {
		current.symbols[name] = sym
	}
	s.byQName[qualified] = sym

	if file != nil && file.PackageName != "" {
		relative := strings.TrimPrefix(qualified, file.PackageName+".")
		s.byQName[relative] = sym
	}
	return sym
}

func (s *lowerState) lowerDecl(current *scope, owner *namedSymbol, decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.InterfaceDecl:
		s.lowerInterface(current, owner, d)
	case *ast.ParcelableDecl:
		s.lowerParcelable(current, owner, d)
	case *ast.EnumDecl:
		s.lowerEnum(current, owner, d)
	case *ast.UnionDecl:
		s.lowerUnion(current, owner, d)
	}
}

func (s *lowerState) lowerInterface(current *scope, owner *namedSymbol, decl *ast.InterfaceDecl) {
	if decl == nil {
		return
	}
	sym := current.symbols[decl.Name]
	if sym == nil {
		return
	}

	iface := &Interface{
		AIDLName:   sym.aidlName,
		GoName:     sym.goName,
		Descriptor: descriptorName(s.file.PackageName, decl.Name),
		Oneway:     decl.Oneway,
		Stable:     hasVintfStabilityAnnotation(decl.Annotations),
	}
	if iface.Stable {
		if cfg, ok := s.stableInterface(sym.aidlName); ok {
			iface.Version = cfg.Version
			iface.Hash = cfg.Hash
		}
	}

	txCode := uint32(1)
	for _, member := range decl.Members {
		switch m := member.(type) {
		case *ast.ConstDecl:
			typ := s.lowerType(sym.scope, m.Type)
			if typ == nil {
				continue
			}
			iface.Consts = append(iface.Consts, Const{
				Name:   m.Name,
				GoName: sym.goName + exportName(m.Name),
				Type:   typ,
				Value:  m.Value,
			})
		case *ast.MethodDecl:
			method := s.lowerMethod(sym.scope, iface, m, txCode)
			if method != nil {
				iface.Methods = append(iface.Methods, method)
				txCode++
			}
		case *ast.ParcelableDecl:
			s.lowerParcelable(sym.scope, sym, m)
		case *ast.EnumDecl:
			s.lowerEnum(sym.scope, sym, m)
		case *ast.UnionDecl:
			s.lowerUnion(sym.scope, sym, m)
		}
	}

	s.out.Interfaces = append(s.out.Interfaces, iface)
}

func (s *lowerState) lowerMethod(current *scope, iface *Interface, decl *ast.MethodDecl, txCode uint32) *Method {
	if decl == nil {
		return nil
	}

	method := &Method{
		Name:            decl.Name,
		GoName:          exportName(decl.Name),
		TransactionCode: txCode,
		Oneway:          iface.Oneway || decl.Oneway,
	}

	returnRef := decl.Return
	if hasNullableAnnotation(decl.Annotations) {
		returnRef.Nullable = true
	}

	if returnRef.Name != "void" {
		typ := s.lowerType(current, returnRef)
		if typ == nil {
			return nil
		}
		method.Return = &Field{
			Name:   "ret",
			GoName: "ret",
			Type:   typ,
		}
	}

	for _, arg := range decl.Args {
		typ := s.lowerType(current, applyFieldTypeAnnotations(arg))
		if typ == nil {
			return nil
		}
		field := Field{
			Name:      arg.Name,
			GoName:    lowerName(arg.Name),
			Direction: arg.Direction,
			Type:      typ,
		}
		switch arg.Direction {
		case ast.DirectionOut:
			method.Outputs = append(method.Outputs, field)
		case ast.DirectionInOut:
			method.Inputs = append(method.Inputs, field)
			outField := field
			outField.GoName = field.GoName + "Out"
			method.Outputs = append(method.Outputs, outField)
		default:
			method.Inputs = append(method.Inputs, field)
		}
	}

	if method.Oneway {
		if method.Return != nil {
			s.diags = append(s.diags, Diagnostic{
				Message: fmt.Sprintf("oneway method %s.%s must not return a value", iface.GoName, method.GoName),
			})
		}
		if len(method.Outputs) != 0 {
			s.diags = append(s.diags, Diagnostic{
				Message: fmt.Sprintf("oneway method %s.%s must not declare out or inout parameters", iface.GoName, method.GoName),
			})
		}
	}

	return method
}

func (s *lowerState) lowerParcelable(current *scope, owner *namedSymbol, decl *ast.ParcelableDecl) {
	if decl == nil {
		return
	}
	sym := current.symbols[decl.Name]
	if sym == nil {
		return
	}

	parc := &Parcelable{
		AIDLName:   sym.aidlName,
		GoName:     sym.goName,
		Structured: decl.Structured,
	}
	if !decl.Structured {
		if cfg, ok := s.customParcelable(sym.aidlName); ok {
			parc.Custom = &CustomParcelable{
				GoPackage:   cfg.GoPackage,
				GoType:      cfg.GoType,
				WriteFunc:   cfg.WriteFunc,
				ReadFunc:    cfg.ReadFunc,
				Nullable:    cfg.Nullable,
				ImportAlias: customImportAlias(cfg.GoPackage),
			}
		}
	}
	for _, field := range decl.Fields {
		typ := s.lowerType(current, applyFieldTypeAnnotations(field))
		if typ == nil {
			return
		}
		parc.Fields = append(parc.Fields, Field{
			Name:   field.Name,
			GoName: exportName(field.Name),
			Type:   typ,
		})
	}
	s.out.Parcelables = append(s.out.Parcelables, parc)
}

func (s *lowerState) lowerEnum(current *scope, owner *namedSymbol, decl *ast.EnumDecl) {
	if decl == nil {
		return
	}
	sym := current.symbols[decl.Name]
	if sym == nil {
		return
	}

	enum := &Enum{
		AIDLName:   sym.aidlName,
		GoName:     sym.goName,
		Underlying: s.enumBackingType(decl.Annotations),
	}
	for _, member := range decl.Members {
		enum.Members = append(enum.Members, EnumMember{
			Name:   member.Name,
			GoName: sym.goName + exportName(member.Name),
			Value:  member.Value,
		})
	}
	s.out.Enums = append(s.out.Enums, enum)
}

func (s *lowerState) enumBackingType(annotations []ast.Annotation) *Type {
	backing := "int"
	for _, ann := range annotations {
		if ann.Name != "Backing" && ann.Name != "android.annotation.Backing" {
			continue
		}
		for _, arg := range ann.Args {
			if arg.Name != "type" {
				continue
			}
			backing = strings.Trim(arg.Value, "\"")
		}
	}

	switch backing {
	case "byte":
		return builtinType("byte", false)
	case "int":
		return builtinType("int", false)
	case "long":
		return builtinType("long", false)
	default:
		s.diags = append(s.diags, Diagnostic{
			Message: fmt.Sprintf("unsupported enum backing type %q", backing),
		})
		return builtinType("int", false)
	}
}

func (s *lowerState) lowerUnion(current *scope, owner *namedSymbol, decl *ast.UnionDecl) {
	if decl == nil {
		return
	}
	sym := current.symbols[decl.Name]
	if sym == nil {
		return
	}

	union := &Union{
		AIDLName: sym.aidlName,
		GoName:   sym.goName,
	}
	for _, field := range decl.Fields {
		typ := s.lowerType(current, applyFieldTypeAnnotations(field))
		if typ == nil {
			return
		}
		union.Fields = append(union.Fields, Field{
			Name:   field.Name,
			GoName: exportName(field.Name),
			Type:   typ,
		})
	}
	s.out.Unions = append(s.out.Unions, union)
}

func (s *lowerState) lowerType(current *scope, ref ast.TypeRef) *Type {
	if ref.Name == "" {
		s.diags = append(s.diags, Diagnostic{Message: "empty type reference"})
		return nil
	}

	if ref.Name == "List" {
		if len(ref.TypeArgs) != 1 {
			s.diags = append(s.diags, Diagnostic{
				Message: fmt.Sprintf("List requires exactly one type argument, got %d", len(ref.TypeArgs)),
			})
			return nil
		}
		elem := s.lowerType(current, ref.TypeArgs[0])
		if elem == nil {
			return nil
		}
		return &Type{
			Kind:   TypeSlice,
			GoExpr: "[]" + elem.GoExpr,
			Elem:   elem,
			IsList: true,
		}
	}

	if ref.Array {
		base := ast.TypeRef{
			Name:     ref.Name,
			TypeArgs: ref.TypeArgs,
			Nullable: ref.Nullable,
			Array:    false,
		}
		elem := s.lowerType(current, base)
		if elem == nil {
			return nil
		}
		if ref.FixedArrayLen != nil {
			return &Type{
				Kind:     TypeArray,
				GoExpr:   fmt.Sprintf("[%d]%s", *ref.FixedArrayLen, elem.GoExpr),
				Elem:     elem,
				FixedLen: *ref.FixedArrayLen,
				IsArray:  true,
			}
		}
		return &Type{
			Kind:    TypeSlice,
			GoExpr:  "[]" + elem.GoExpr,
			Elem:    elem,
			IsArray: true,
		}
	}

	if builtin := builtinType(ref.Name, ref.Nullable); builtin != nil {
		return builtin
	}

	sym := s.lookupType(current, ref.Name)
	if sym == nil {
		s.diags = append(s.diags, Diagnostic{
			Message: fmt.Sprintf("unresolved type %q", ref.Name),
		})
		s.out.ExternalRefs = appendIfMissing(s.out.ExternalRefs, ref.Name)
		return &Type{
			Kind:     TypeExternal,
			AIDLName: ref.Name,
			GoExpr:   exportName(lastSegment(ref.Name)),
			Nullable: ref.Nullable,
			NamedGo:  exportName(lastSegment(ref.Name)),
		}
	}

	kind := TypeUnsupported
	switch sym.kind {
	case namedInterface:
		kind = TypeInterface
	case namedParcelable:
		kind = TypeParcelable
	case namedEnum:
		kind = TypeEnum
	case namedUnion:
		kind = TypeUnion
	}

	goExpr := sym.goName
	if sym.kind == namedParcelable {
		if decl, ok := sym.decl.(*ast.ParcelableDecl); ok && !decl.Structured {
			if cfg, ok := s.customParcelable(sym.aidlName); ok {
				goExpr = customImportAlias(cfg.GoPackage) + "." + cfg.GoType
				if ref.Nullable {
					if !cfg.Nullable {
						s.diags = append(s.diags, Diagnostic{
							Message: fmt.Sprintf("nullable custom parcelable %q requires sidecar nullable support", sym.aidlName),
						})
					}
					goExpr = "*" + goExpr
				}
				return &Type{
					Kind:     kind,
					AIDLName: sym.aidlName,
					GoExpr:   goExpr,
					Nullable: ref.Nullable,
					NamedGo:  sym.goName,
				}
			}
		}
	}
	if ref.Nullable {
		switch kind {
		case TypeParcelable, TypeUnion:
			goExpr = "*" + goExpr
		case TypeInterface:
			// Go interfaces are already nullable.
		default:
			s.diags = append(s.diags, Diagnostic{
				Message: fmt.Sprintf("nullable %s is not supported for type %q", kind, sym.aidlName),
			})
		}
	}

	return &Type{
		Kind:     kind,
		AIDLName: sym.aidlName,
		GoExpr:   goExpr,
		Nullable: ref.Nullable,
		NamedGo:  sym.goName,
	}
}

func (s *lowerState) customParcelable(aidlName string) (CustomParcelableConfig, bool) {
	if len(s.customParcelables) == 0 {
		return CustomParcelableConfig{}, false
	}
	cfg, ok := s.customParcelables[aidlName]
	return cfg, ok
}

func (s *lowerState) stableInterface(aidlName string) (StableInterfaceConfig, bool) {
	if len(s.stableInterfaces) == 0 {
		return StableInterfaceConfig{}, false
	}
	cfg, ok := s.stableInterfaces[aidlName]
	return cfg, ok
}

func (s *lowerState) lookupType(current *scope, name string) *namedSymbol {
	if strings.Contains(name, ".") {
		if sym := s.byQName[name]; sym != nil {
			return sym
		}
		if s.file.PackageName != "" {
			if sym := s.byQName[s.file.PackageName+"."+name]; sym != nil {
				return sym
			}
		}
		return nil
	}

	for scope := current; scope != nil; scope = scope.parent {
		if sym := scope.symbols[name]; sym != nil {
			return sym
		}
	}
	if s.file.PackageName != "" {
		if sym := s.byQName[s.file.PackageName+"."+name]; sym != nil {
			return sym
		}
	}
	for _, imp := range s.file.Imports {
		if lastSegment(imp.Path) != name {
			continue
		}
		if sym := s.byQName[imp.Path]; sym != nil {
			return sym
		}
	}
	return nil
}

func builtinType(name string, nullable bool) *Type {
	switch name {
	case "void":
		return &Type{Kind: TypeVoid, AIDLName: "void", GoExpr: ""}
	case "boolean":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeBool, AIDLName: name, GoExpr: "bool", IsBuiltin: true}
	case "byte":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeByte, AIDLName: name, GoExpr: "int8", IsBuiltin: true}
	case "char":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeChar, AIDLName: name, GoExpr: "uint16", IsBuiltin: true}
	case "int":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeInt32, AIDLName: name, GoExpr: "int32", IsBuiltin: true}
	case "long":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeInt64, AIDLName: name, GoExpr: "int64", IsBuiltin: true}
	case "float":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeFloat32, AIDLName: name, GoExpr: "float32", IsBuiltin: true}
	case "double":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeFloat64, AIDLName: name, GoExpr: "float64", IsBuiltin: true}
	case "String":
		goExpr := "string"
		if nullable {
			goExpr = "*string"
		}
		return &Type{Kind: TypeString, AIDLName: name, GoExpr: goExpr, Nullable: nullable, IsBuiltin: true}
	case "IBinder":
		return &Type{Kind: TypeBinder, AIDLName: name, GoExpr: "binder.Binder", Nullable: true, IsBuiltin: true}
	case "FileDescriptor":
		if nullable {
			return unsupportedNullableBuiltin(name)
		}
		return &Type{Kind: TypeFileDescriptor, AIDLName: name, GoExpr: "binder.FileDescriptor", IsBuiltin: true}
	case "ParcelFileDescriptor":
		goExpr := "binder.ParcelFileDescriptor"
		if nullable {
			goExpr = "*binder.ParcelFileDescriptor"
		}
		return &Type{
			Kind:      TypeParcelFileDescriptor,
			AIDLName:  name,
			GoExpr:    goExpr,
			Nullable:  nullable,
			IsBuiltin: true,
		}
	default:
		return nil
	}
}

func unsupportedNullableBuiltin(name string) *Type {
	return &Type{
		Kind:      TypeUnsupported,
		AIDLName:  name,
		GoExpr:    exportName(name),
		Nullable:  true,
		IsBuiltin: true,
	}
}

func descriptorName(pkg string, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

func defaultGoPackageName(aidlPkg string, sourcePath string) string {
	if aidlPkg != "" {
		return sanitizePackageName(lastSegment(aidlPkg))
	}
	if sourcePath != "" {
		base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
		if base != "" {
			return sanitizePackageName(base)
		}
	}
	return "aidl"
}

func sanitizePackageName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		out = "aidl"
	}
	if token.Lookup(out).IsKeyword() {
		out += "_aidl"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "aidl_" + out
	}
	return out
}

func exportName(name string) string {
	parts := splitIdentifier(name)
	if len(parts) == 0 {
		return "X"
	}
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	out := strings.Join(parts, "")
	if out == "" {
		out = "X"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "X" + out
	}
	return out
}

func lowerName(name string) string {
	exp := exportName(name)
	r, size := utf8.DecodeRuneInString(exp)
	if r == utf8.RuneError && size == 0 {
		return "x"
	}
	return string(unicode.ToLower(r)) + exp[size:]
}

func splitIdentifier(name string) []string {
	fields := strings.FieldsFunc(name, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
	if len(fields) == 0 {
		return nil
	}
	var out []string
	for _, field := range fields {
		if field == strings.ToUpper(field) {
			out = append(out, strings.ToLower(field))
			continue
		}
		var current []rune
		for i, r := range field {
			if i > 0 && unicode.IsUpper(r) && len(current) > 0 {
				out = append(out, strings.ToLower(string(current)))
				current = current[:0]
			}
			current = append(current, r)
		}
		if len(current) > 0 {
			out = append(out, strings.ToLower(string(current)))
		}
	}
	return out
}

func appendIfMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func lastSegment(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return name
}

func hasNullableAnnotation(annotations []ast.Annotation) bool {
	for _, ann := range annotations {
		if ann.Name == "nullable" || ann.Name == "android.annotation.Nullable" {
			return true
		}
	}
	return false
}

func hasVintfStabilityAnnotation(annotations []ast.Annotation) bool {
	for _, ann := range annotations {
		if ann.Name == "VintfStability" || ann.Name == "android.annotation.VintfStability" {
			return true
		}
	}
	return false
}

func applyFieldTypeAnnotations(field ast.Field) ast.TypeRef {
	ref := field.Type
	if hasNullableAnnotation(field.Annotations) {
		ref.Nullable = true
	}
	return ref
}

func parseIntLiteral(text string) (int64, error) {
	return strconv.ParseInt(text, 0, 64)
}
