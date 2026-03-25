package resolve

import (
	"testing"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
)

func TestValidateFileDetectsDuplicateTopLevelDecls(t *testing.T) {
	file := &ast.File{
		Decls: []ast.Decl{
			&ast.ParcelableDecl{Name: "Foo"},
			&ast.EnumDecl{Name: "Foo"},
		},
	}

	diags := ValidateFile(file)
	if len(diags) != 1 {
		t.Fatalf("len(diags) = %d, want 1", len(diags))
	}
}

func TestValidateFileDetectsDuplicateInterfaceMembers(t *testing.T) {
	file := &ast.File{
		Decls: []ast.Decl{
			&ast.InterfaceDecl{
				Name: "IFoo",
				Members: []ast.InterfaceMember{
					&ast.MethodDecl{Name: "Call"},
					&ast.ConstDecl{Name: "Call"},
				},
			},
		},
	}

	diags := ValidateFile(file)
	if len(diags) != 1 {
		t.Fatalf("len(diags) = %d, want 1", len(diags))
	}
}
