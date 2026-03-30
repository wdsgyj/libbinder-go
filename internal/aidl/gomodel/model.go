package gomodel

import "github.com/wdsgyj/libbinder-go/internal/aidl/ast"

type Diagnostic struct {
	Message string `json:"message"`
}

type File struct {
	SourcePath   string        `json:"source_path,omitempty"`
	AIDLPackage  string        `json:"aidl_package,omitempty"`
	GoPackage    string        `json:"go_package"`
	Imports      []ast.Import  `json:"imports,omitempty"`
	Interfaces   []*Interface  `json:"interfaces,omitempty"`
	Parcelables  []*Parcelable `json:"parcelables,omitempty"`
	Enums        []*Enum       `json:"enums,omitempty"`
	Unions       []*Union      `json:"unions,omitempty"`
	ExternalRefs []string      `json:"external_refs,omitempty"`
}

type Interface struct {
	AIDLName   string    `json:"aidl_name"`
	GoName     string    `json:"go_name"`
	Descriptor string    `json:"descriptor"`
	Oneway     bool      `json:"oneway,omitempty"`
	Stable     bool      `json:"stable,omitempty"`
	Version    int32     `json:"version,omitempty"`
	Hash       string    `json:"hash,omitempty"`
	Consts     []Const   `json:"consts,omitempty"`
	Methods    []*Method `json:"methods,omitempty"`
}

type Method struct {
	Name            string  `json:"name"`
	GoName          string  `json:"go_name"`
	TransactionCode uint32  `json:"transaction_code"`
	Oneway          bool    `json:"oneway,omitempty"`
	Inputs          []Field `json:"inputs,omitempty"`
	Return          *Field  `json:"return,omitempty"`
	Outputs         []Field `json:"outputs,omitempty"`
}

type Parcelable struct {
	AIDLName   string            `json:"aidl_name"`
	GoName     string            `json:"go_name"`
	Structured bool              `json:"structured,omitempty"`
	FixedSize  bool              `json:"fixed_size,omitempty"`
	Custom     *CustomParcelable `json:"custom,omitempty"`
	Consts     []Const           `json:"consts,omitempty"`
	Fields     []Field           `json:"fields,omitempty"`
}

type CustomParcelable struct {
	GoPackage   string `json:"go_package"`
	GoType      string `json:"go_type"`
	WriteFunc   string `json:"write_func"`
	ReadFunc    string `json:"read_func"`
	Nullable    bool   `json:"nullable,omitempty"`
	ImportAlias string `json:"import_alias,omitempty"`
}

type CustomParcelableConfig struct {
	AIDLName  string `json:"aidl_name"`
	GoPackage string `json:"go_package"`
	GoType    string `json:"go_type"`
	WriteFunc string `json:"write_func"`
	ReadFunc  string `json:"read_func"`
	Nullable  bool   `json:"nullable,omitempty"`
}

type TypeMappingFile struct {
	Version     int                      `json:"version"`
	Parcelables []CustomParcelableConfig `json:"parcelables,omitempty"`
	Interfaces  []StableInterfaceConfig  `json:"interfaces,omitempty"`
}

type StableInterfaceConfig struct {
	AIDLName string `json:"aidl_name"`
	Version  int32  `json:"version"`
	Hash     string `json:"hash"`
}

type Enum struct {
	AIDLName   string       `json:"aidl_name"`
	GoName     string       `json:"go_name"`
	Underlying *Type        `json:"underlying"`
	Members    []EnumMember `json:"members,omitempty"`
}

type EnumMember struct {
	Name   string `json:"name"`
	GoName string `json:"go_name"`
	Value  string `json:"value,omitempty"`
}

type Union struct {
	AIDLName  string  `json:"aidl_name"`
	GoName    string  `json:"go_name"`
	FixedSize bool    `json:"fixed_size,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
}

type Const struct {
	Name   string `json:"name"`
	GoName string `json:"go_name"`
	Type   *Type  `json:"type"`
	Value  string `json:"value"`
}

type Field struct {
	Name         string        `json:"name"`
	GoName       string        `json:"go_name"`
	Direction    ast.Direction `json:"direction,omitempty"`
	Type         *Type         `json:"type"`
	DefaultValue string        `json:"default_value,omitempty"`
}

type TypeKind string

const (
	TypeVoid                 TypeKind = "void"
	TypeBool                 TypeKind = "bool"
	TypeByte                 TypeKind = "byte"
	TypeChar                 TypeKind = "char"
	TypeInt32                TypeKind = "int32"
	TypeInt64                TypeKind = "int64"
	TypeFloat32              TypeKind = "float32"
	TypeFloat64              TypeKind = "float64"
	TypeString               TypeKind = "string"
	TypeBinder               TypeKind = "binder"
	TypeFileDescriptor       TypeKind = "file_descriptor"
	TypeParcelFileDescriptor TypeKind = "parcel_file_descriptor"
	TypeInterface            TypeKind = "interface"
	TypeParcelable           TypeKind = "parcelable"
	TypeEnum                 TypeKind = "enum"
	TypeUnion                TypeKind = "union"
	TypeSlice                TypeKind = "slice"
	TypeArray                TypeKind = "array"
	TypeMap                  TypeKind = "map"
	TypeExternal             TypeKind = "external"
	TypeUnsupported          TypeKind = "unsupported"
)

type Type struct {
	Kind        TypeKind `json:"kind"`
	AIDLName    string   `json:"aidl_name,omitempty"`
	DeclPackage string   `json:"decl_package,omitempty"`
	GoExpr      string   `json:"go_expr"`
	Nullable    bool     `json:"nullable,omitempty"`
	FixedLen    int      `json:"fixed_len,omitempty"`
	Elem        *Type    `json:"elem,omitempty"`
	Key         *Type    `json:"key,omitempty"`
	Value       *Type    `json:"value,omitempty"`
	NamedGo     string   `json:"named_go,omitempty"`
	IsList      bool     `json:"is_list,omitempty"`
	IsArray     bool     `json:"is_array,omitempty"`
	IsBuiltin   bool     `json:"is_builtin,omitempty"`
}
