package resolve

import (
	"testing"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
	"github.com/wdsgyj/libbinder-go/internal/aidl/parser"
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

func TestValidateFileAcceptsExpressionsAndParcelableDefaults(t *testing.T) {
	src := `
package demo;

interface IFoo {
  const int A = 1 << 0;
  const int B = A | (1 << 1);
}

@FixedSize
parcelable Holder {
  enum Kind {
    ONE,
    TWO,
  }
  const int Mask = 1 << 3;
  int id;
  int[2] pair;
}

parcelable UseDefault {
  Holder.Kind kind = Holder.Kind.TWO;
}
`

	file, err := parser.Parse("ok.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	diags := ValidateFile(file)
	if len(diags) != 0 {
		t.Fatalf("ValidateFile diags = %#v, want none", diags)
	}
}

func TestValidateFileRejectsInvalidAnnotations(t *testing.T) {
	src := `
package demo;

enum Kind {
  ONE,
}

@Backing(type="string")
enum BadKind {
  ONE,
}

parcelable Foo {
  @nullable int bad;
}

@FixedSize
parcelable BadFixed {
  String text;
}
`

	file, err := parser.Parse("bad.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	diags := ValidateFile(file)
	if len(diags) < 3 {
		t.Fatalf("len(diags) = %d, want at least 3; diags = %#v", len(diags), diags)
	}
}
