package resolve

import (
	"fmt"
	"go/constant"
	"strings"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
	aidlexpr "github.com/wdsgyj/libbinder-go/internal/aidl/expr"
)

type Diagnostic struct {
	Message string `json:"message"`
}

func ValidateFile(file *ast.File) []Diagnostic {
	if file == nil {
		return []Diagnostic{{Message: "nil file"}}
	}

	v := validator{
		file:         file,
		declsByQName: map[string]ast.Decl{},
		declsByName:  map[string]ast.Decl{},
	}
	v.collectDecls(nil, file.Decls)
	v.validateDecls(nil, file.Decls)
	return v.diags
}

type validator struct {
	file         *ast.File
	diags        []Diagnostic
	declsByQName map[string]ast.Decl
	declsByName  map[string]ast.Decl
}

func (v *validator) collectDecls(scope []string, decls []ast.Decl) {
	for _, decl := range decls {
		name := declName(decl)
		if name == "" {
			continue
		}
		qname := v.qualifiedName(scope, name)
		if _, ok := v.declsByQName[qname]; !ok {
			v.declsByQName[qname] = decl
			registerDeclAliases(v.declsByName, qname, decl)
		}
		if iface, ok := decl.(*ast.InterfaceDecl); ok {
			nestedScope := append(append([]string{}, scope...), iface.Name)
			var nested []ast.Decl
			for _, member := range iface.Members {
				switch m := member.(type) {
				case *ast.ParcelableDecl:
					nested = append(nested, m)
				case *ast.EnumDecl:
					nested = append(nested, m)
				case *ast.UnionDecl:
					nested = append(nested, m)
				}
			}
			v.collectDecls(nestedScope, nested)
			continue
		}
		if parc, ok := decl.(*ast.ParcelableDecl); ok && len(parc.Decls) != 0 {
			nestedScope := append(append([]string{}, scope...), parc.Name)
			v.collectDecls(nestedScope, parc.Decls)
		}
	}
}

func (v *validator) validateDecls(scope []string, decls []ast.Decl) {
	seen := map[string]struct{}{}
	for _, decl := range decls {
		name := declName(decl)
		if name != "" {
			if _, ok := seen[name]; ok {
				v.addf("duplicate declaration %q", name)
				continue
			}
			seen[name] = struct{}{}
		}
		v.validateDecl(scope, decl)
	}
}

func (v *validator) validateDecl(scope []string, decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.InterfaceDecl:
		v.validateInterface(scope, d)
	case *ast.ParcelableDecl:
		v.validateParcelable(scope, d)
	case *ast.EnumDecl:
		v.validateEnum(scope, d)
	case *ast.UnionDecl:
		v.validateUnion(scope, d)
	}
}

func (v *validator) validateInterface(scope []string, decl *ast.InterfaceDecl) {
	if decl == nil {
		return
	}
	v.validateDeclAnnotations("interface", decl.Annotations, true)

	memberSeen := map[string]struct{}{}
	constEnv := aidlexpr.Env{}

	for _, member := range decl.Members {
		memberName := interfaceMemberName(member)
		if memberName != "" {
			if _, ok := memberSeen[memberName]; ok {
				v.addf("duplicate interface member %q in %s", memberName, decl.Name)
				continue
			}
			memberSeen[memberName] = struct{}{}
		}

		switch m := member.(type) {
		case *ast.MethodDecl:
			v.validateMethod(scope, decl, m)
		case *ast.ConstDecl:
			v.validateConst(scope, decl, m, constEnv)
		case *ast.ParcelableDecl:
			v.validateParcelable(append(append([]string{}, scope...), decl.Name), m)
		case *ast.EnumDecl:
			v.validateEnum(append(append([]string{}, scope...), decl.Name), m)
		case *ast.UnionDecl:
			v.validateUnion(append(append([]string{}, scope...), decl.Name), m)
		}
	}
}

func (v *validator) validateMethod(scope []string, owner *ast.InterfaceDecl, decl *ast.MethodDecl) {
	if decl == nil {
		return
	}
	v.validateNullableAnnotation(decl.Annotations, decl.Return, fmt.Sprintf("method %s.%s return", owner.Name, decl.Name))
	v.validateTypeRef(scope, decl.Return, fmt.Sprintf("method %s.%s return", owner.Name, decl.Name))
	for _, arg := range decl.Args {
		v.validateField(scope, arg, fmt.Sprintf("method %s.%s argument %s", owner.Name, decl.Name, arg.Name))
	}
}

func (v *validator) validateConst(scope []string, owner *ast.InterfaceDecl, decl *ast.ConstDecl, env aidlexpr.Env) {
	if decl == nil {
		return
	}
	v.validateTypeRef(scope, decl.Type, fmt.Sprintf("const %s.%s", owner.Name, decl.Name))
	value, err := aidlexpr.Eval(decl.Value, env)
	if err != nil {
		v.addf("const %s.%s has invalid expression %q: %v", owner.Name, decl.Name, decl.Value, err)
		return
	}
	registerValueAliases(env, v.qualifiedName([]string{owner.Name}, decl.Name), value)
}

func (v *validator) validateParcelable(scope []string, decl *ast.ParcelableDecl) {
	if decl == nil {
		return
	}
	v.validateDeclAnnotations("parcelable", decl.Annotations, decl.Structured)
	memberSeen := map[string]struct{}{}
	constEnv := aidlexpr.Env{}
	for _, c := range decl.Consts {
		if _, ok := memberSeen[c.Name]; ok {
			v.addf("duplicate parcelable member %q in %s", c.Name, decl.Name)
			continue
		}
		memberSeen[c.Name] = struct{}{}
		v.validateParcelableConst(scope, decl, &c, constEnv)
	}
	fieldScope := append(append([]string{}, scope...), decl.Name)
	for _, field := range decl.Fields {
		if _, ok := memberSeen[field.Name]; ok {
			v.addf("duplicate parcelable member %q in %s", field.Name, decl.Name)
			continue
		}
		memberSeen[field.Name] = struct{}{}
		v.validateField(fieldScope, field, fmt.Sprintf("parcelable %s field %s", decl.Name, field.Name))
	}
	if len(decl.Decls) != 0 {
		for _, nested := range decl.Decls {
			if name := declName(nested); name != "" {
				if _, ok := memberSeen[name]; ok {
					v.addf("duplicate parcelable member %q in %s", name, decl.Name)
					continue
				}
				memberSeen[name] = struct{}{}
			}
		}
		v.validateDecls(fieldScope, decl.Decls)
	}
	if hasAnnotation(decl.Annotations, "FixedSize") || hasAnnotation(decl.Annotations, "android.annotation.FixedSize") {
		for _, field := range decl.Fields {
			if !v.isFixedSizeType(fieldScope, applyFieldAnnotations(field), map[string]struct{}{}) {
				v.addf("parcelable %s annotated @FixedSize but field %s is not fixed-size", decl.Name, field.Name)
			}
		}
	}
}

func (v *validator) validateParcelableConst(scope []string, owner *ast.ParcelableDecl, decl *ast.ConstDecl, env aidlexpr.Env) {
	if decl == nil {
		return
	}
	v.validateTypeRef(scope, decl.Type, fmt.Sprintf("parcelable %s const %s", owner.Name, decl.Name))
	value, err := aidlexpr.Eval(decl.Value, env)
	if err != nil {
		v.addf("parcelable %s const %s has invalid expression %q: %v", owner.Name, decl.Name, decl.Value, err)
		return
	}
	registerValueAliases(env, v.qualifiedName(append(append([]string{}, scope...), owner.Name), decl.Name), value)
}

func (v *validator) validateEnum(scope []string, decl *ast.EnumDecl) {
	if decl == nil {
		return
	}
	v.validateEnumAnnotations(decl)

	env := aidlexpr.Env{}
	var next int64
	nextKnown := true
	for _, member := range decl.Members {
		if member.Value != "" {
			value, err := aidlexpr.EvalInt64(member.Value, env)
			if err != nil {
				v.addf("enum %s member %s has invalid value %q: %v", decl.Name, member.Name, member.Value, err)
				nextKnown = false
			} else {
				registerValueAliases(env, v.qualifiedName(scope, decl.Name)+"."+member.Name, constant.MakeInt64(value))
				next = value + 1
				nextKnown = true
			}
			continue
		}
		if !nextKnown {
			v.addf("enum %s member %s requires an explicit value after a non-evaluable expression", decl.Name, member.Name)
			continue
		}
		registerValueAliases(env, v.qualifiedName(scope, decl.Name)+"."+member.Name, constant.MakeInt64(next))
		next++
	}
}

func (v *validator) validateUnion(scope []string, decl *ast.UnionDecl) {
	if decl == nil {
		return
	}
	v.validateDeclAnnotations("union", decl.Annotations, true)
	for _, field := range decl.Fields {
		v.validateField(scope, field, fmt.Sprintf("union %s field %s", decl.Name, field.Name))
	}
	if hasAnnotation(decl.Annotations, "FixedSize") || hasAnnotation(decl.Annotations, "android.annotation.FixedSize") {
		for _, field := range decl.Fields {
			if !v.isFixedSizeType(scope, applyFieldAnnotations(field), map[string]struct{}{}) {
				v.addf("union %s annotated @FixedSize but field %s is not fixed-size", decl.Name, field.Name)
			}
		}
	}
}

func (v *validator) validateField(scope []string, field ast.Field, location string) {
	v.validateNullableAnnotation(field.Annotations, field.Type, location)
	v.validateTypeRef(scope, field.Type, location)
	if field.DefaultValue != "" {
		if _, err := aidlexpr.Parse(field.DefaultValue); err != nil {
			v.addf("%s has invalid default expression %q: %v", location, field.DefaultValue, err)
		}
	}
}

func (v *validator) validateTypeRef(scope []string, ref ast.TypeRef, location string) {
	v.validateNullableAnnotation(ref.Annotations, ref, location)
	for _, arg := range ref.TypeArgs {
		v.validateTypeRef(scope, arg, location)
	}
	if ref.Array && ref.FixedArrayLen != nil && *ref.FixedArrayLen < 0 {
		v.addf("%s has invalid fixed array length %d", location, *ref.FixedArrayLen)
	}
}

func (v *validator) validateDeclAnnotations(kind string, annotations []ast.Annotation, structured bool) {
	for _, ann := range annotations {
		switch ann.Name {
		case "VintfStability", "android.annotation.VintfStability":
			if len(ann.Args) != 0 {
				v.addf("@%s does not take arguments", ann.Name)
			}
			if kind != "interface" && kind != "parcelable" && kind != "union" {
				v.addf("@%s is not valid on %s", ann.Name, kind)
			}
		case "FixedSize", "android.annotation.FixedSize":
			if len(ann.Args) != 0 {
				v.addf("@%s does not take arguments", ann.Name)
			}
			if kind != "parcelable" && kind != "union" {
				v.addf("@%s is only valid on parcelable or union declarations", ann.Name)
			}
			if kind == "parcelable" && !structured {
				v.addf("@%s requires a structured parcelable declaration", ann.Name)
			}
		case "nullable", "android.annotation.Nullable", "Backing", "android.annotation.Backing":
			v.addf("@%s is not valid on %s declarations", ann.Name, kind)
		}
	}
}

func (v *validator) validateEnumAnnotations(decl *ast.EnumDecl) {
	for _, ann := range decl.Annotations {
		switch ann.Name {
		case "Backing", "android.annotation.Backing":
			var backing string
			for _, arg := range ann.Args {
				if arg.Name == "" || arg.Name == "type" {
					backing = strings.Trim(arg.Value, "\"")
					continue
				}
				v.addf("@%s only supports the named argument type=...", ann.Name)
			}
			if backing == "" {
				v.addf("@%s requires type=\"byte\"|\"int\"|\"long\"", ann.Name)
				continue
			}
			switch backing {
			case "byte", "int", "long":
			default:
				v.addf("enum %s has unsupported backing type %q", decl.Name, backing)
			}
		case "VintfStability", "android.annotation.VintfStability":
			v.addf("@%s is not valid on enum declarations", ann.Name)
		case "nullable", "android.annotation.Nullable":
			v.addf("@%s is not valid on enum declarations", ann.Name)
		case "FixedSize", "android.annotation.FixedSize":
			v.addf("@%s is not valid on enum declarations", ann.Name)
		}
	}
}

func (v *validator) validateNullableAnnotation(annotations []ast.Annotation, ref ast.TypeRef, location string) {
	for _, ann := range annotations {
		if ann.Name != "nullable" && ann.Name != "android.annotation.Nullable" {
			continue
		}
		if !v.isNullableType(ref) {
			v.addf("%s uses @%s on non-nullable type %s", location, ann.Name, ref.Name)
		}
		for _, arg := range ann.Args {
			switch arg.Name {
			case "heap":
				if _, err := aidlexpr.EvalBool(arg.Value, nil); err != nil {
					v.addf("%s uses invalid @%s(heap=%s): %v", location, ann.Name, arg.Value, err)
				}
			case "":
				if _, err := aidlexpr.EvalBool(arg.Value, nil); err != nil {
					v.addf("%s uses invalid @%s argument %s: %v", location, ann.Name, arg.Value, err)
				}
			default:
				v.addf("%s uses unsupported @%s argument %s", location, ann.Name, arg.Name)
			}
		}
	}
}

func (v *validator) isNullableType(ref ast.TypeRef) bool {
	if ref.Array || ref.Name == "List" {
		return true
	}
	switch ref.Name {
	case "String", "IBinder", "ParcelFileDescriptor":
		return true
	case "void", "boolean", "byte", "char", "int", "long", "float", "double", "FileDescriptor":
		return false
	}
	decl := v.lookupType(ref.Name)
	switch decl.(type) {
	case *ast.EnumDecl:
		return false
	default:
		return true
	}
}

func (v *validator) isFixedSizeType(scope []string, ref ast.TypeRef, seen map[string]struct{}) bool {
	if ref.Nullable {
		return false
	}
	if ref.Array {
		if ref.FixedArrayLen == nil {
			return false
		}
		base := ref
		base.Array = false
		base.FixedArrayLen = nil
		return v.isFixedSizeType(scope, base, seen)
	}
	if ref.Name == "List" {
		return false
	}
	switch ref.Name {
	case "boolean", "byte", "char", "int", "long", "float", "double":
		return true
	case "String", "IBinder", "FileDescriptor", "ParcelFileDescriptor":
		return false
	}

	decl := v.lookupType(ref.Name)
	switch d := decl.(type) {
	case *ast.EnumDecl:
		return true
	case *ast.ParcelableDecl:
		if !hasAnnotation(d.Annotations, "FixedSize") && !hasAnnotation(d.Annotations, "android.annotation.FixedSize") {
			return false
		}
		qname := v.lookupQName(scope, ref.Name)
		if _, ok := seen[qname]; ok {
			return true
		}
		seen[qname] = struct{}{}
		for _, field := range d.Fields {
			if !v.isFixedSizeType(scope, applyFieldAnnotations(field), seen) {
				return false
			}
		}
		return true
	case *ast.UnionDecl:
		if !hasAnnotation(d.Annotations, "FixedSize") && !hasAnnotation(d.Annotations, "android.annotation.FixedSize") {
			return false
		}
		qname := v.lookupQName(scope, ref.Name)
		if _, ok := seen[qname]; ok {
			return true
		}
		seen[qname] = struct{}{}
		for _, field := range d.Fields {
			if !v.isFixedSizeType(scope, applyFieldAnnotations(field), seen) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (v *validator) lookupType(name string) ast.Decl {
	if name == "" {
		return nil
	}
	if decl, ok := v.declsByQName[name]; ok {
		return decl
	}
	if v.file != nil && v.file.PackageName != "" {
		if decl, ok := v.declsByQName[v.file.PackageName+"."+name]; ok {
			return decl
		}
	}
	return v.declsByName[name]
}

func (v *validator) lookupQName(scope []string, name string) string {
	if strings.Contains(name, ".") {
		if _, ok := v.declsByQName[name]; ok {
			return name
		}
		if v.file != nil && v.file.PackageName != "" {
			return v.file.PackageName + "." + name
		}
		return name
	}
	return v.qualifiedName(scope, name)
}

func (v *validator) qualifiedName(scope []string, name string) string {
	parts := make([]string, 0, 2+len(scope))
	if v.file != nil && v.file.PackageName != "" {
		parts = append(parts, v.file.PackageName)
	}
	parts = append(parts, scope...)
	parts = append(parts, name)
	return strings.Join(parts, ".")
}

func (v *validator) addf(format string, args ...any) {
	v.diags = append(v.diags, Diagnostic{Message: fmt.Sprintf(format, args...)})
}

func registerDeclAliases(dst map[string]ast.Decl, qname string, decl ast.Decl) {
	registerNameAliases(qname, func(alias string) {
		if _, ok := dst[alias]; !ok {
			dst[alias] = decl
		}
	})
}

func registerValueAliases(dst aidlexpr.Env, qname string, value constant.Value) {
	registerNameAliases(qname, func(alias string) {
		if _, ok := dst[alias]; !ok {
			dst[alias] = value
		}
	})
}

func registerNameAliases(qname string, fn func(string)) {
	if qname == "" || fn == nil {
		return
	}
	parts := strings.Split(qname, ".")
	for i := range parts {
		fn(strings.Join(parts[i:], "."))
	}
}

func applyFieldAnnotations(field ast.Field) ast.TypeRef {
	ref := field.Type
	if hasAnnotation(field.Annotations, "nullable") || hasAnnotation(field.Annotations, "android.annotation.Nullable") {
		ref.Nullable = true
	}
	return ref
}

func hasAnnotation(annotations []ast.Annotation, names ...string) bool {
	for _, ann := range annotations {
		for _, name := range names {
			if ann.Name == name {
				return true
			}
		}
	}
	return false
}

func declName(decl ast.Decl) string {
	switch d := decl.(type) {
	case *ast.InterfaceDecl:
		return d.Name
	case *ast.ParcelableDecl:
		return d.Name
	case *ast.EnumDecl:
		return d.Name
	case *ast.UnionDecl:
		return d.Name
	default:
		return ""
	}
}

func interfaceMemberName(member ast.InterfaceMember) string {
	switch m := member.(type) {
	case *ast.MethodDecl:
		return m.Name
	case *ast.ConstDecl:
		return m.Name
	case *ast.ParcelableDecl:
		return m.Name
	case *ast.EnumDecl:
		return m.Name
	case *ast.UnionDecl:
		return m.Name
	default:
		return ""
	}
}
