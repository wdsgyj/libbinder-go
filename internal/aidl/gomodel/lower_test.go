package gomodel

import (
	"testing"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
	"github.com/wdsgyj/libbinder-go/internal/aidl/parser"
)

func TestLowerInterfaceWithNestedTypes(t *testing.T) {
	src := `
package android.test.demo;

interface IEcho {
  const int VERSION = 3;
  @nullable String Echo(in String msg, out int code, inout Payload payload);
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

	file, err := parser.Parse("demo.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{SourcePath: "demo.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}

	if model.GoPackage != "demo" {
		t.Fatalf("GoPackage = %q, want demo", model.GoPackage)
	}
	if len(model.Interfaces) != 1 {
		t.Fatalf("len(Interfaces) = %d, want 1", len(model.Interfaces))
	}
	if len(model.Parcelables) != 1 {
		t.Fatalf("len(Parcelables) = %d, want 1", len(model.Parcelables))
	}
	if len(model.Enums) != 1 {
		t.Fatalf("len(Enums) = %d, want 1", len(model.Enums))
	}
	if len(model.Unions) != 1 {
		t.Fatalf("len(Unions) = %d, want 1", len(model.Unions))
	}

	iface := model.Interfaces[0]
	if iface.GoName != "IEcho" {
		t.Fatalf("iface.GoName = %q, want IEcho", iface.GoName)
	}
	if iface.Descriptor != "android.test.demo.IEcho" {
		t.Fatalf("iface.Descriptor = %q, want android.test.demo.IEcho", iface.Descriptor)
	}
	if len(iface.Consts) != 1 {
		t.Fatalf("len(Consts) = %d, want 1", len(iface.Consts))
	}
	if len(iface.Methods) != 1 {
		t.Fatalf("len(Methods) = %d, want 1", len(iface.Methods))
	}

	method := iface.Methods[0]
	if method.TransactionCode != 1 {
		t.Fatalf("TransactionCode = %d, want 1", method.TransactionCode)
	}
	if method.Return == nil || method.Return.Type.GoExpr != "*string" {
		t.Fatalf("Return = %#v, want nullable string", method.Return)
	}
	if len(method.Inputs) != 2 {
		t.Fatalf("len(Inputs) = %d, want 2", len(method.Inputs))
	}
	if method.Inputs[1].Type.GoExpr != "IEchoPayload" {
		t.Fatalf("payload input type = %q, want IEchoPayload", method.Inputs[1].Type.GoExpr)
	}
	if len(method.Outputs) != 2 {
		t.Fatalf("len(Outputs) = %d, want 2", len(method.Outputs))
	}
	if method.Outputs[1].GoName != "payloadOut" {
		t.Fatalf("inout output GoName = %q, want payloadOut", method.Outputs[1].GoName)
	}

	if model.Parcelables[0].GoName != "IEchoPayload" {
		t.Fatalf("nested parcelable GoName = %q, want IEchoPayload", model.Parcelables[0].GoName)
	}
	if model.Enums[0].GoName != "IEchoKind" {
		t.Fatalf("nested enum GoName = %q, want IEchoKind", model.Enums[0].GoName)
	}
	if model.Unions[0].GoName != "IEchoResult" {
		t.Fatalf("nested union GoName = %q, want IEchoResult", model.Unions[0].GoName)
	}
}

func TestLowerOnewayDiagnostics(t *testing.T) {
	src := `
package demo;

oneway interface IFoo {
  int Call(out int value);
}
`

	file, err := parser.Parse("foo.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	_, diags := Lower(file, LowerOptions{SourcePath: "foo.aidl"})
	if len(diags) == 0 {
		t.Fatal("Lower diagnostics = 0, want non-zero")
	}
}

func TestLowerBuiltinFileDescriptorTypes(t *testing.T) {
	src := `
package demo;

parcelable Payload {
  FileDescriptor fd;
  @nullable ParcelFileDescriptor pfd;
}
`

	file, err := parser.Parse("fd.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{SourcePath: "fd.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}
	if len(model.Parcelables) != 1 {
		t.Fatalf("len(Parcelables) = %d, want 1", len(model.Parcelables))
	}

	fields := model.Parcelables[0].Fields
	if len(fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(fields))
	}
	if fields[0].Type.Kind != TypeFileDescriptor || fields[0].Type.GoExpr != "binder.FileDescriptor" {
		t.Fatalf("fd type = %#v, want binder.FileDescriptor", fields[0].Type)
	}
	if fields[1].Type.Kind != TypeParcelFileDescriptor || fields[1].Type.GoExpr != "*binder.ParcelFileDescriptor" || !fields[1].Type.Nullable {
		t.Fatalf("pfd type = %#v, want nullable binder.ParcelFileDescriptor", fields[1].Type)
	}
}

func TestLowerNullablePrimitiveArrayKeepsPrimitiveElementType(t *testing.T) {
	src := `
package demo;

interface IService {
  @nullable int[] GetIds();
}
`

	file, err := parser.Parse("array.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{SourcePath: "array.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}

	if len(model.Interfaces) != 1 || len(model.Interfaces[0].Methods) != 1 {
		t.Fatalf("Interfaces = %#v, want single interface with single method", model.Interfaces)
	}
	ret := model.Interfaces[0].Methods[0].Return.Type
	if ret.Kind != TypeSlice || ret.GoExpr != "[]int32" {
		t.Fatalf("Return = %#v, want []int32 slice", ret)
	}
	if ret.Elem == nil || ret.Elem.Kind != TypeInt32 || ret.Elem.GoExpr != "int32" {
		t.Fatalf("Return.Elem = %#v, want int32 element", ret.Elem)
	}
	if !ret.Nullable {
		t.Fatalf("Return.Nullable = false, want true")
	}
}

func TestLowerMethodArgsAvoidReservedGoIdentifiers(t *testing.T) {
	src := `
package demo;

interface IService {
  void Ping(in int map, in String type, in String binder, in int err);
}
`

	file, err := parser.Parse("keywords.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{SourcePath: "keywords.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}

	method := model.Interfaces[0].Methods[0]
	if got := method.Inputs[0].GoName; got != "map_" {
		t.Fatalf("first arg GoName = %q, want map_", got)
	}
	if got := method.Inputs[1].GoName; got != "type_" {
		t.Fatalf("second arg GoName = %q, want type_", got)
	}
	if got := method.Inputs[2].GoName; got != "binder_" {
		t.Fatalf("third arg GoName = %q, want binder_", got)
	}
	if got := method.Inputs[3].GoName; got != "err_" {
		t.Fatalf("fourth arg GoName = %q, want err_", got)
	}
}

func TestLowerCustomParcelableWithMappings(t *testing.T) {
	src := `
package demo;

parcelable Foo;

interface IService {
  Foo Echo(in Foo value);
  @nullable Foo Maybe();
}
`

	file, err := parser.Parse("custom.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{
		SourcePath: "custom.aidl",
		CustomParcelables: map[string]CustomParcelableConfig{
			"demo.Foo": {
				AIDLName:  "demo.Foo",
				GoPackage: "example.com/custom/foo",
				GoType:    "NativeHandle",
				WriteFunc: "WriteNativeHandleToParcel",
				ReadFunc:  "ReadNativeHandleFromParcel",
				Nullable:  true,
			},
		},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}
	if len(model.Parcelables) != 1 || model.Parcelables[0].Custom == nil {
		t.Fatalf("custom parcelable = %#v, want mapped custom parcelable", model.Parcelables)
	}
	if model.Parcelables[0].Custom.ImportAlias != "custom_foo" {
		t.Fatalf("ImportAlias = %q, want custom_foo", model.Parcelables[0].Custom.ImportAlias)
	}

	iface := model.Interfaces[0]
	if got := iface.Methods[0].Inputs[0].Type.GoExpr; got != "custom_foo.NativeHandle" {
		t.Fatalf("Echo input GoExpr = %q, want custom_foo.NativeHandle", got)
	}
	if got := iface.Methods[1].Return.Type.GoExpr; got != "*custom_foo.NativeHandle" {
		t.Fatalf("Maybe return GoExpr = %q, want *custom_foo.NativeHandle", got)
	}
}

func TestLowerStableInterfaceMetadata(t *testing.T) {
	src := `
package demo;

@VintfStability
interface IEcho {
  void Ping();
}
`

	file, err := parser.Parse("stable.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{
		SourcePath: "stable.aidl",
		StableInterfaces: map[string]StableInterfaceConfig{
			"demo.IEcho": {
				AIDLName: "demo.IEcho",
				Version:  3,
				Hash:     "abcdef",
			},
		},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}

	iface := model.Interfaces[0]
	if !iface.Stable {
		t.Fatal("iface.Stable = false, want true")
	}
	if iface.Version != 3 || iface.Hash != "abcdef" {
		t.Fatalf("stable metadata = (%d, %q), want (3, abcdef)", iface.Version, iface.Hash)
	}
}

func TestLowerQualifiedParcelableForwardDeclaration(t *testing.T) {
	src := `
package android.app;

parcelable ActivityManager.MemoryInfo;

interface IService {
  ActivityManager.MemoryInfo Get();
}
`

	file, err := parser.Parse("activity_manager.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{SourcePath: "activity_manager.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}
	if len(model.Parcelables) != 1 {
		t.Fatalf("len(Parcelables) = %d, want 1", len(model.Parcelables))
	}
	if got := model.Parcelables[0].AIDLName; got != "android.app.ActivityManager.MemoryInfo" {
		t.Fatalf("Parcelable AIDLName = %q, want android.app.ActivityManager.MemoryInfo", got)
	}
	if got := model.Parcelables[0].GoName; got != "ActivityManagerMemoryInfo" {
		t.Fatalf("Parcelable GoName = %q, want ActivityManagerMemoryInfo", got)
	}
	if len(model.Interfaces) != 1 || len(model.Interfaces[0].Methods) != 1 {
		t.Fatalf("Interfaces = %#v, want single interface with single method", model.Interfaces)
	}
	if got := model.Interfaces[0].Methods[0].Return.Type.GoExpr; got != "ActivityManagerMemoryInfo" {
		t.Fatalf("Return.GoExpr = %q, want ActivityManagerMemoryInfo", got)
	}
}

func TestLowerRewritesConstExpressions(t *testing.T) {
	src := `
package demo;

interface IFoo {
  const int A = 1 << 0;
  const int B = A | (1 << 1);
}
`

	file, err := parser.Parse("consts.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{SourcePath: "consts.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}

	if got := model.Interfaces[0].Consts[1].Value; got != "(IFooA|((1<<1)))" {
		t.Fatalf("const B value = %q, want (IFooA|((1<<1)))", got)
	}
}

func TestLowerParcelableNestedEnumDefaultValue(t *testing.T) {
	src := `
package demo;

parcelable Holder {
  enum Kind {
    ONE,
    TWO,
  }
  const int Mask = 1 << 3;
  Kind kind = Kind.TWO;
}
`

	file, err := parser.Parse("holder.aidl", src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	model, diags := Lower(file, LowerOptions{SourcePath: "holder.aidl"})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}
	if len(model.Enums) != 1 {
		t.Fatalf("len(model.Enums) = %d, want 1", len(model.Enums))
	}
	if len(model.Parcelables[0].Consts) != 1 || model.Parcelables[0].Consts[0].Value != "(1<<3)" {
		t.Fatalf("parcelable consts = %#v, want Mask=(1<<3)", model.Parcelables[0].Consts)
	}
	if got := model.Parcelables[0].Fields[0].DefaultValue; got != "HolderKindTwo" {
		t.Fatalf("default value = %q, want HolderKindTwo", got)
	}
}

func TestLowerResolvesImportedDependencyFiles(t *testing.T) {
	callbackSrc := `
package demo;

interface ICallback {
  void Ping();
}
`
	serviceSrc := `
package demo;

import demo.ICallback;

interface IService {
  ICallback Bind();
}
`

	callbackFile, err := parser.Parse("ICallback.aidl", callbackSrc)
	if err != nil {
		t.Fatalf("Parse(callback): %v", err)
	}
	serviceFile, err := parser.Parse("IService.aidl", serviceSrc)
	if err != nil {
		t.Fatalf("Parse(service): %v", err)
	}

	model, diags := Lower(serviceFile, LowerOptions{
		SourcePath:      "IService.aidl",
		DependencyFiles: []*ast.File{callbackFile},
	})
	if len(diags) != 0 {
		t.Fatalf("Lower diagnostics = %#v, want none", diags)
	}
	if len(model.ExternalRefs) != 0 {
		t.Fatalf("ExternalRefs = %#v, want none", model.ExternalRefs)
	}
	if got := model.Interfaces[0].Methods[0].Return.Type.GoExpr; got != "ICallback" {
		t.Fatalf("Bind return GoExpr = %q, want ICallback", got)
	}
}
