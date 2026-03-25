package ir

import "github.com/wdsgyj/libbinder-go/internal/aidl/ast"

type File struct {
	PackageName string       `json:"package_name,omitempty"`
	Decls       []Decl       `json:"decls,omitempty"`
	Imports     []ast.Import `json:"imports,omitempty"`
}

type Decl struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func Lower(file *ast.File) *File {
	if file == nil {
		return nil
	}

	out := &File{
		PackageName: file.PackageName,
		Imports:     append([]ast.Import(nil), file.Imports...),
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.InterfaceDecl:
			out.Decls = append(out.Decls, Decl{Kind: "interface", Name: d.Name})
		case *ast.ParcelableDecl:
			kind := "parcelable"
			if d.Structured {
				kind = "structured_parcelable"
			}
			out.Decls = append(out.Decls, Decl{Kind: kind, Name: d.Name})
		case *ast.EnumDecl:
			out.Decls = append(out.Decls, Decl{Kind: "enum", Name: d.Name})
		case *ast.UnionDecl:
			out.Decls = append(out.Decls, Decl{Kind: "union", Name: d.Name})
		}
	}
	return out
}
