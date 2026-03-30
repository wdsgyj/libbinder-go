package ast

type File struct {
	PackageName string   `json:"package_name,omitempty"`
	Imports     []Import `json:"imports,omitempty"`
	Decls       []Decl   `json:"decls,omitempty"`
}

type Import struct {
	Path string `json:"path"`
}

type Decl interface {
	declNode()
}

type InterfaceMember interface {
	interfaceMemberNode()
}

type Annotation struct {
	Name string          `json:"name"`
	Args []AnnotationArg `json:"args,omitempty"`
}

type AnnotationArg struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}

type Direction string

const (
	DirectionIn    Direction = "in"
	DirectionOut   Direction = "out"
	DirectionInOut Direction = "inout"
)

type TypeRef struct {
	Annotations   []Annotation `json:"annotations,omitempty"`
	Name          string       `json:"name"`
	TypeArgs      []TypeRef    `json:"type_args,omitempty"`
	Array         bool         `json:"array,omitempty"`
	FixedArrayLen *int         `json:"fixed_array_len,omitempty"`
	Nullable      bool         `json:"nullable,omitempty"`
}

type Field struct {
	Annotations  []Annotation `json:"annotations,omitempty"`
	Direction    Direction    `json:"direction,omitempty"`
	Type         TypeRef      `json:"type"`
	Name         string       `json:"name"`
	DefaultValue string       `json:"default_value,omitempty"`
}

type InterfaceDecl struct {
	Annotations []Annotation      `json:"annotations,omitempty"`
	Oneway      bool              `json:"oneway,omitempty"`
	Name        string            `json:"name"`
	Members     []InterfaceMember `json:"members,omitempty"`
}

func (*InterfaceDecl) declNode() {}

type MethodDecl struct {
	Annotations []Annotation `json:"annotations,omitempty"`
	Oneway      bool         `json:"oneway,omitempty"`
	Return      TypeRef      `json:"return"`
	Name        string       `json:"name"`
	Args        []Field      `json:"args,omitempty"`
	Transaction string       `json:"transaction,omitempty"`
}

func (*MethodDecl) interfaceMemberNode() {}

type ConstDecl struct {
	Annotations []Annotation `json:"annotations,omitempty"`
	Type        TypeRef      `json:"type"`
	Name        string       `json:"name"`
	Value       string       `json:"value"`
}

func (*ConstDecl) interfaceMemberNode() {}

type ParcelableDecl struct {
	Annotations []Annotation `json:"annotations,omitempty"`
	Name        string       `json:"name"`
	TypeParams  []string     `json:"type_params,omitempty"`
	Structured  bool         `json:"structured,omitempty"`
	Consts      []ConstDecl  `json:"consts,omitempty"`
	Fields      []Field      `json:"fields,omitempty"`
	Decls       []Decl       `json:"decls,omitempty"`
}

func (*ParcelableDecl) declNode()            {}
func (*ParcelableDecl) interfaceMemberNode() {}

type EnumDecl struct {
	Annotations []Annotation `json:"annotations,omitempty"`
	Name        string       `json:"name"`
	Members     []EnumMember `json:"members,omitempty"`
}

func (*EnumDecl) declNode()            {}
func (*EnumDecl) interfaceMemberNode() {}

type EnumMember struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type UnionDecl struct {
	Annotations []Annotation `json:"annotations,omitempty"`
	Name        string       `json:"name"`
	Fields      []Field      `json:"fields,omitempty"`
}

func (*UnionDecl) declNode()            {}
func (*UnionDecl) interfaceMemberNode() {}
