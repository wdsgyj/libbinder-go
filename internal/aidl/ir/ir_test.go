package ir

import (
	"testing"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
)

func TestLower(t *testing.T) {
	file := &ast.File{
		PackageName: "demo",
		Decls: []ast.Decl{
			&ast.InterfaceDecl{Name: "IFoo"},
			&ast.ParcelableDecl{Name: "Bar", Structured: true},
			&ast.EnumDecl{Name: "Kind"},
			&ast.UnionDecl{Name: "Value"},
		},
	}

	got := Lower(file)
	if got == nil {
		t.Fatal("Lower returned nil")
	}
	if len(got.Decls) != 4 {
		t.Fatalf("len(Decls) = %d, want 4", len(got.Decls))
	}
	if got.Decls[1].Kind != "structured_parcelable" {
		t.Fatalf("Decls[1].Kind = %q, want structured_parcelable", got.Decls[1].Kind)
	}
}
