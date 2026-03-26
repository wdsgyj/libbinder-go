package parser

import (
	"testing"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
)

func TestParseFileWithInterfaceAndNestedTypes(t *testing.T) {
	src := `
package android.test.demo;
import android.os.ParcelFileDescriptor;

@VintfStability
oneway interface IEcho {
  const int VERSION = 3;
  @nullable String Echo(in String msg, out int code);
  parcelable Payload {
    int id;
    @nullable String name;
  }
  enum Kind { ONE = 1, TWO = 2 }
  union Result {
    int code;
    String text;
  }
}
`

	file, err := Parse("demo.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if file.PackageName != "android.test.demo" {
		t.Fatalf("PackageName = %q, want %q", file.PackageName, "android.test.demo")
	}
	if len(file.Imports) != 1 || file.Imports[0].Path != "android.os.ParcelFileDescriptor" {
		t.Fatalf("Imports = %#v", file.Imports)
	}
	if len(file.Decls) != 1 {
		t.Fatalf("len(Decls) = %d, want 1", len(file.Decls))
	}

	iface, ok := file.Decls[0].(*ast.InterfaceDecl)
	if !ok {
		t.Fatalf("decl type = %T, want *ast.InterfaceDecl", file.Decls[0])
	}
	if !iface.Oneway {
		t.Fatal("interface should be oneway")
	}
	if iface.Name != "IEcho" {
		t.Fatalf("Name = %q, want IEcho", iface.Name)
	}
	if len(iface.Members) != 5 {
		t.Fatalf("len(Members) = %d, want 5", len(iface.Members))
	}

	method, ok := iface.Members[1].(*ast.MethodDecl)
	if !ok {
		t.Fatalf("member[1] type = %T, want *ast.MethodDecl", iface.Members[1])
	}
	if method.Return.Name != "String" {
		t.Fatalf("method.Return.Name = %q, want String", method.Return.Name)
	}
	if len(method.Args) != 2 {
		t.Fatalf("len(method.Args) = %d, want 2", len(method.Args))
	}
	if method.Args[0].Direction != ast.DirectionIn {
		t.Fatalf("arg0 direction = %q, want in", method.Args[0].Direction)
	}
	if method.Args[1].Direction != ast.DirectionOut {
		t.Fatalf("arg1 direction = %q, want out", method.Args[1].Direction)
	}
}

func TestParseAnnotationArgs(t *testing.T) {
	src := `
package demo;

parcelable Foo {
  @nullable(heap=true) String value;
}
`

	file, err := Parse("foo.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	decl, ok := file.Decls[0].(*ast.ParcelableDecl)
	if !ok {
		t.Fatalf("decl type = %T, want *ast.ParcelableDecl", file.Decls[0])
	}
	if len(decl.Fields) != 1 {
		t.Fatalf("len(Fields) = %d, want 1", len(decl.Fields))
	}
	field := decl.Fields[0]
	if len(field.Annotations) != 1 {
		t.Fatalf("len(Annotations) = %d, want 1", len(field.Annotations))
	}
	if field.Annotations[0].Name != "nullable" {
		t.Fatalf("annotation name = %q, want nullable", field.Annotations[0].Name)
	}
	if len(field.Annotations[0].Args) != 1 {
		t.Fatalf("len(annotation.Args) = %d, want 1", len(field.Annotations[0].Args))
	}
	if field.Annotations[0].Args[0].Name != "heap" || field.Annotations[0].Args[0].Value != "true" {
		t.Fatalf("annotation args = %#v, want heap=true", field.Annotations[0].Args)
	}
}

func TestParseStructuredParcelableAndArrays(t *testing.T) {
	src := `
package demo;

parcelable Foo {
  byte[] raw;
  int[4] ids;
  List<String> names;
}
`

	file, err := Parse("foo.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	decl, ok := file.Decls[0].(*ast.ParcelableDecl)
	if !ok {
		t.Fatalf("decl type = %T, want *ast.ParcelableDecl", file.Decls[0])
	}
	if !decl.Structured {
		t.Fatal("Structured = false, want true")
	}
	if len(decl.Fields) != 3 {
		t.Fatalf("len(Fields) = %d, want 3", len(decl.Fields))
	}
	if !decl.Fields[0].Type.Array {
		t.Fatal("raw should be an array type")
	}
	if decl.Fields[1].Type.FixedArrayLen == nil || *decl.Fields[1].Type.FixedArrayLen != 4 {
		t.Fatalf("ids.FixedArrayLen = %#v, want 4", decl.Fields[1].Type.FixedArrayLen)
	}
	if decl.Fields[2].Type.Name != "List" || len(decl.Fields[2].Type.TypeArgs) != 1 {
		t.Fatalf("names type = %#v, want List<String>", decl.Fields[2].Type)
	}
}

func TestParseConstExpressionsAndParcelableDefaults(t *testing.T) {
	src := `
package demo;

interface IFoo {
  const int A = 1 << 0;
  const int B = A | (1 << 1);
}

parcelable Holder {
  enum Kind {
    ONE,
    TWO,
  }
  const int Mask = 1 << 3;
  Kind kind = Kind.TWO;
}
`

	file, err := Parse("exprs.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(file.Decls) != 2 {
		t.Fatalf("len(Decls) = %d, want 2", len(file.Decls))
	}

	iface, ok := file.Decls[0].(*ast.InterfaceDecl)
	if !ok {
		t.Fatalf("decl[0] type = %T, want *ast.InterfaceDecl", file.Decls[0])
	}
	constA, ok := iface.Members[0].(*ast.ConstDecl)
	if !ok {
		t.Fatalf("member[0] type = %T, want *ast.ConstDecl", iface.Members[0])
	}
	if constA.Value != "1<<0" {
		t.Fatalf("const A value = %q, want 1<<0", constA.Value)
	}
	constB := iface.Members[1].(*ast.ConstDecl)
	if constB.Value != "A|(1<<1)" {
		t.Fatalf("const B value = %q, want A|(1<<1)", constB.Value)
	}

	holder, ok := file.Decls[1].(*ast.ParcelableDecl)
	if !ok {
		t.Fatalf("decl[1] type = %T, want *ast.ParcelableDecl", file.Decls[1])
	}
	if len(holder.Decls) != 1 {
		t.Fatalf("len(holder.Decls) = %d, want 1", len(holder.Decls))
	}
	if _, ok := holder.Decls[0].(*ast.EnumDecl); !ok {
		t.Fatalf("holder.Decls[0] type = %T, want *ast.EnumDecl", holder.Decls[0])
	}
	if len(holder.Consts) != 1 || holder.Consts[0].Value != "1<<3" {
		t.Fatalf("holder.Consts = %#v, want Mask=1<<3", holder.Consts)
	}
	if len(holder.Fields) != 1 {
		t.Fatalf("len(holder.Fields) = %d, want 1", len(holder.Fields))
	}
	if got := holder.Fields[0].DefaultValue; got != "Kind.TWO" {
		t.Fatalf("holder.Fields[0].DefaultValue = %q, want Kind.TWO", got)
	}
}
