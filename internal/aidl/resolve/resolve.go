package resolve

import (
	"fmt"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
)

type Diagnostic struct {
	Message string `json:"message"`
}

func ValidateFile(file *ast.File) []Diagnostic {
	if file == nil {
		return []Diagnostic{{Message: "nil file"}}
	}

	var diags []Diagnostic
	seen := map[string]struct{}{}
	for _, decl := range file.Decls {
		name := declName(decl)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			diags = append(diags, Diagnostic{Message: fmt.Sprintf("duplicate declaration %q", name)})
			continue
		}
		seen[name] = struct{}{}

		if iface, ok := decl.(*ast.InterfaceDecl); ok {
			memberSeen := map[string]struct{}{}
			for _, member := range iface.Members {
				memberName := interfaceMemberName(member)
				if memberName == "" {
					continue
				}
				if _, ok := memberSeen[memberName]; ok {
					diags = append(diags, Diagnostic{
						Message: fmt.Sprintf("duplicate interface member %q in %s", memberName, iface.Name),
					})
					continue
				}
				memberSeen[memberName] = struct{}{}
			}
		}
	}
	return diags
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
