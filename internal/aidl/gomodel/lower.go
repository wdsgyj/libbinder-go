package gomodel

import (
	"fmt"
	goast "go/ast"
	"go/constant"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
	aidlexpr "github.com/wdsgyj/libbinder-go/internal/aidl/expr"
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
		importAliases:     importAliases(file),
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
	importAliases     map[string]string
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
		sym := s.registerDecl(file, current, owner, d.Name, namedParcelable, d)
		if len(d.Decls) != 0 {
			nestedScope := &scope{
				parent:  current,
				prefix:  sym.aidlName,
				symbols: map[string]*namedSymbol{},
			}
			sym.scope = nestedScope
			for _, nested := range d.Decls {
				s.collectDecl(file, nestedScope, sym, nested)
			}
		}
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

	constNames := map[string]string{}
	constValues := aidlexpr.Env{}
	txCode := uint32(1)
	for _, member := range decl.Members {
		switch m := member.(type) {
		case *ast.ConstDecl:
			typ := s.lowerType(sym.scope, m.Type)
			if typ == nil {
				continue
			}
			value := s.lowerValueExpression(sym.scope, m.Value, constNames)
			if value == "" {
				value = aidlexpr.Normalize(m.Value)
			}
			goName := sym.goName + exportName(m.Name)
			iface.Consts = append(iface.Consts, Const{
				Name:   m.Name,
				GoName: goName,
				Type:   typ,
				Value:  value,
			})
			if v, err := aidlexpr.Eval(m.Value, constValues); err == nil {
				s.registerValueAliases(constValues, sym.aidlName+"."+m.Name, v)
			}
			s.registerNameAliases(constNames, sym.aidlName+"."+m.Name, goName)
		case *ast.MethodDecl:
			methodTxCode := txCode
			if m.Transaction != "" {
				valueExpr := s.lowerValueExpression(sym.scope, m.Transaction, constNames)
				if valueExpr == "" {
					valueExpr = aidlexpr.Normalize(m.Transaction)
				}
				value, err := aidlexpr.EvalInt64(m.Transaction, constValues)
				if err != nil {
					s.diags = append(s.diags, Diagnostic{
						Message: fmt.Sprintf("method %s.%s has invalid transaction id %q: %v", iface.GoName, m.Name, m.Transaction, err),
					})
				} else if value <= 0 {
					s.diags = append(s.diags, Diagnostic{
						Message: fmt.Sprintf("method %s.%s has invalid non-positive transaction id %q", iface.GoName, m.Name, valueExpr),
					})
				} else {
					methodTxCode = uint32(value)
					if next := uint32(value) + 1; next > txCode {
						txCode = next
					}
				}
			}
			method := s.lowerMethod(sym.scope, iface, m, methodTxCode)
			if method != nil {
				iface.Methods = append(iface.Methods, method)
				if m.Transaction == "" {
					txCode++
				}
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
		FixedSize:  hasFixedSizeAnnotation(decl.Annotations),
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
	fieldScope := current
	if sym.scope != nil {
		fieldScope = sym.scope
	}
	constNames := map[string]string{}
	constValues := aidlexpr.Env{}
	for _, c := range decl.Consts {
		typ := s.lowerType(fieldScope, c.Type)
		if typ == nil {
			return
		}
		value := s.lowerValueExpression(fieldScope, c.Value, constNames)
		if value == "" {
			value = aidlexpr.Normalize(c.Value)
		}
		goName := sym.goName + exportName(c.Name)
		parc.Consts = append(parc.Consts, Const{
			Name:   c.Name,
			GoName: goName,
			Type:   typ,
			Value:  value,
		})
		if v, err := aidlexpr.Eval(c.Value, constValues); err == nil {
			s.registerValueAliases(constValues, sym.aidlName+"."+c.Name, v)
		}
		s.registerNameAliases(constNames, sym.aidlName+"."+c.Name, goName)
	}
	for _, field := range decl.Fields {
		typ := s.lowerType(fieldScope, applyFieldTypeAnnotations(field))
		if typ == nil {
			return
		}
		defaultValue := ""
		if field.DefaultValue != "" {
			defaultValue = s.lowerValueExpression(fieldScope, field.DefaultValue, constNames)
			if defaultValue == "" {
				defaultValue = aidlexpr.Normalize(field.DefaultValue)
			}
		}
		parc.Fields = append(parc.Fields, Field{
			Name:         field.Name,
			GoName:       exportName(field.Name),
			Type:         typ,
			DefaultValue: defaultValue,
		})
	}
	s.out.Parcelables = append(s.out.Parcelables, parc)
	for _, nested := range decl.Decls {
		s.lowerDecl(fieldScope, sym, nested)
	}
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
	memberNames := map[string]string{}
	memberValues := aidlexpr.Env{}
	var nextValue int64
	nextKnown := true
	for _, member := range decl.Members {
		goName := sym.goName + exportName(member.Name)
		value := member.Value
		if value != "" {
			value = s.lowerValueExpression(current, value, memberNames)
			if value == "" {
				value = aidlexpr.Normalize(member.Value)
			}
			if v, err := aidlexpr.EvalInt64(member.Value, memberValues); err == nil {
				nextValue = v + 1
				nextKnown = true
				s.registerValueAliases(memberValues, sym.aidlName+"."+member.Name, constant.MakeInt64(v))
			} else {
				nextKnown = false
			}
		} else {
			if !nextKnown {
				s.diags = append(s.diags, Diagnostic{
					Message: fmt.Sprintf("enum %s member %s requires an explicit value after a non-evaluable expression", sym.goName, member.Name),
				})
				value = "0"
			} else {
				value = fmt.Sprintf("%d", nextValue)
				s.registerValueAliases(memberValues, sym.aidlName+"."+member.Name, constant.MakeInt64(nextValue))
				nextValue++
			}
		}
		enum.Members = append(enum.Members, EnumMember{
			Name:   member.Name,
			GoName: goName,
			Value:  value,
		})
		s.registerNameAliases(memberNames, sym.aidlName+"."+member.Name, goName)
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
		AIDLName:  sym.aidlName,
		GoName:    sym.goName,
		FixedSize: hasFixedSizeAnnotation(decl.Annotations),
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

	if ref.Name == "Map" {
		switch len(ref.TypeArgs) {
		case 0:
			return &Type{
				Kind:     TypeMap,
				GoExpr:   "map[any]any",
				Key:      &Type{Kind: TypeUnsupported, GoExpr: "any"},
				Value:    &Type{Kind: TypeUnsupported, GoExpr: "any"},
				Nullable: true,
			}
		case 2:
			key := s.lowerType(current, ref.TypeArgs[0])
			if key == nil {
				return nil
			}
			value := s.lowerType(current, ref.TypeArgs[1])
			if value == nil {
				return nil
			}
			return &Type{
				Kind:     TypeMap,
				GoExpr:   fmt.Sprintf("map[%s]%s", key.GoExpr, value.GoExpr),
				Key:      key,
				Value:    value,
				Nullable: true,
			}
		default:
			s.diags = append(s.diags, Diagnostic{
				Message: fmt.Sprintf("Map requires zero or two type arguments, got %d", len(ref.TypeArgs)),
			})
			return nil
		}
	}

	if ref.Array {
		base := ast.TypeRef{
			Name:     ref.Name,
			TypeArgs: ref.TypeArgs,
			Array:    false,
		}
		elem := s.lowerType(current, base)
		if elem == nil {
			return nil
		}
		if ref.FixedArrayLen != nil {
			if ref.Nullable {
				s.diags = append(s.diags, Diagnostic{
					Message: fmt.Sprintf("nullable fixed-size array %q is not supported", ref.Name),
				})
			}
			return &Type{
				Kind:     TypeArray,
				GoExpr:   "[]" + elem.GoExpr,
				Elem:     elem,
				FixedLen: *ref.FixedArrayLen,
				IsArray:  true,
				Nullable: ref.Nullable,
			}
		}
		return &Type{
			Kind:     TypeSlice,
			GoExpr:   "[]" + elem.GoExpr,
			Elem:     elem,
			IsArray:  true,
			Nullable: ref.Nullable,
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
			Kind:        TypeExternal,
			AIDLName:    ref.Name,
			GoExpr:      exportName(lastSegment(ref.Name)),
			Nullable:    ref.Nullable,
			NamedGo:     exportName(lastSegment(ref.Name)),
			DeclPackage: packageName(ref.Name),
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
					Kind:        kind,
					AIDLName:    sym.aidlName,
					DeclPackage: sym.packageName,
					GoExpr:      goExpr,
					Nullable:    ref.Nullable,
					NamedGo:     sym.goName,
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
		Kind:        kind,
		AIDLName:    sym.aidlName,
		DeclPackage: sym.packageName,
		GoExpr:      goExpr,
		Nullable:    ref.Nullable,
		NamedGo:     sym.goName,
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
	if full, ok := s.importAliases[name]; ok {
		if sym := s.byQName[full]; sym != nil {
			return sym
		}
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

func importAliases(file *ast.File) map[string]string {
	if file == nil || len(file.Imports) == 0 {
		return nil
	}
	out := make(map[string]string, len(file.Imports))
	for _, imp := range file.Imports {
		if imp.Path == "" {
			continue
		}
		out[lastSegment(imp.Path)] = imp.Path
	}
	return out
}

func packageName(name string) string {
	if idx := strings.LastIndex(name, "."); idx > 0 {
		return name[:idx]
	}
	return ""
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
		dir := filepath.Base(filepath.Dir(sourcePath))
		if dir != "" && dir != "." && dir != string(filepath.Separator) {
			return sanitizePackageName(dir)
		}
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
	out := string(unicode.ToLower(r)) + exp[size:]
	if token.Lookup(out).IsKeyword() || isReservedGeneratedIdentifier(out) {
		return out + "_"
	}
	return out
}

func isReservedGeneratedIdentifier(name string) bool {
	switch name {
	case "binder", "context", "fmt", "errors", "sync":
		return true
	case "ctx", "req", "resp", "reply", "parcel", "registrar":
		return true
	case "err", "ret":
		return true
	default:
		return false
	}
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

func hasFixedSizeAnnotation(annotations []ast.Annotation) bool {
	for _, ann := range annotations {
		if ann.Name == "FixedSize" || ann.Name == "android.annotation.FixedSize" {
			return true
		}
	}
	return false
}

func applyFieldTypeAnnotations(field ast.Field) ast.TypeRef {
	ref := field.Type
	if hasNullableAnnotation(field.Annotations) || hasNullableAnnotation(ref.Annotations) {
		ref.Nullable = true
	}
	return ref
}

func (s *lowerState) lowerValueExpression(current *scope, src string, names map[string]string) string {
	if strings.TrimSpace(src) == "" {
		return ""
	}
	node, err := aidlexpr.Parse(src)
	if err != nil {
		s.diags = append(s.diags, Diagnostic{
			Message: fmt.Sprintf("invalid expression %q: %v", src, err),
		})
		return ""
	}
	value, err := s.renderValueExpr(current, node, names)
	if err != nil {
		s.diags = append(s.diags, Diagnostic{
			Message: fmt.Sprintf("lower expression %q: %v", src, err),
		})
		return ""
	}
	return value
}

func (s *lowerState) renderValueExpr(current *scope, node goast.Expr, names map[string]string) (string, error) {
	switch n := node.(type) {
	case *goast.BasicLit:
		return n.Value, nil
	case *goast.Ident:
		switch n.Name {
		case "true", "false", "nil":
			return n.Name, nil
		}
		if names != nil {
			if value, ok := names[n.Name]; ok {
				return value, nil
			}
		}
		return n.Name, nil
	case *goast.SelectorExpr:
		full, ok := selectorQName(n)
		if !ok {
			return "", fmt.Errorf("unsupported selector")
		}
		if names != nil {
			if value, ok := names[full]; ok {
				return value, nil
			}
		}
		if value, ok := s.lookupEnumMemberGoName(current, full); ok {
			return value, nil
		}
		return full, nil
	case *goast.ParenExpr:
		inner, err := s.renderValueExpr(current, n.X, names)
		if err != nil {
			return "", err
		}
		return "(" + inner + ")", nil
	case *goast.UnaryExpr:
		inner, err := s.renderValueExpr(current, n.X, names)
		if err != nil {
			return "", err
		}
		return n.Op.String() + inner, nil
	case *goast.BinaryExpr:
		left, err := s.renderValueExpr(current, n.X, names)
		if err != nil {
			return "", err
		}
		right, err := s.renderValueExpr(current, n.Y, names)
		if err != nil {
			return "", err
		}
		return "(" + left + n.Op.String() + right + ")", nil
	default:
		return "", fmt.Errorf("unsupported expression %T", node)
	}
}

func (s *lowerState) lookupEnumMemberGoName(current *scope, qname string) (string, bool) {
	dot := strings.LastIndexByte(qname, '.')
	if dot < 0 {
		return "", false
	}
	typeName := qname[:dot]
	memberName := qname[dot+1:]
	sym := s.lookupType(current, typeName)
	if sym == nil || sym.kind != namedEnum {
		return "", false
	}
	decl, ok := sym.decl.(*ast.EnumDecl)
	if !ok {
		return "", false
	}
	for _, member := range decl.Members {
		if member.Name == memberName {
			return sym.goName + exportName(member.Name), true
		}
	}
	return "", false
}

func registerAliases(full string, fn func(string)) {
	if full == "" || fn == nil {
		return
	}
	parts := strings.Split(full, ".")
	for i := range parts {
		fn(strings.Join(parts[i:], "."))
	}
}

func (s *lowerState) registerNameAliases(names map[string]string, full string, goName string) {
	if names == nil {
		return
	}
	registerAliases(full, func(alias string) {
		if _, exists := names[alias]; !exists {
			names[alias] = goName
		}
	})
}

func (s *lowerState) registerValueAliases(env aidlexpr.Env, full string, value constant.Value) {
	if env == nil || value == nil {
		return
	}
	registerAliases(full, func(alias string) {
		if _, exists := env[alias]; !exists {
			env[alias] = value
		}
	})
}

func selectorQName(node *goast.SelectorExpr) (string, bool) {
	if node == nil {
		return "", false
	}
	parts := []string{node.Sel.Name}
	for {
		switch x := node.X.(type) {
		case *goast.Ident:
			parts = append(parts, x.Name)
			for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
				parts[i], parts[j] = parts[j], parts[i]
			}
			return strings.Join(parts, "."), true
		case *goast.SelectorExpr:
			node = x
			parts = append(parts, node.Sel.Name)
		default:
			return "", false
		}
	}
}

func parseIntLiteral(text string) (int64, error) {
	return strconv.ParseInt(text, 0, 64)
}
