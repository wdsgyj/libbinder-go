package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
)

type token struct {
	text string
	kind rune
	pos  scanner.Position
}

type Error struct {
	Pos scanner.Position
	Msg string
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s: %s", e.Pos, e.Msg)
}

type Parser struct {
	s   scanner.Scanner
	tok token
}

func Parse(filename string, src string) (*ast.File, error) {
	p := &Parser{}
	p.s.Init(strings.NewReader(src))
	p.s.Filename = filename
	p.s.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings | scanner.ScanComments | scanner.SkipComments
	p.next()
	return p.parseFile()
}

func ParseReader(filename string, r io.Reader) (*ast.File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Parse(filename, string(data))
}

func (p *Parser) parseFile() (*ast.File, error) {
	file := &ast.File{}

	if p.matchIdent("package") {
		p.next()
		name, err := p.parseQualifiedName()
		if err != nil {
			return nil, err
		}
		file.PackageName = name
		if err := p.expect(";"); err != nil {
			return nil, err
		}
	}

	for p.matchIdent("import") {
		p.next()
		name, err := p.parseQualifiedName()
		if err != nil {
			return nil, err
		}
		file.Imports = append(file.Imports, ast.Import{Path: name})
		if err := p.expect(";"); err != nil {
			return nil, err
		}
	}

	for p.tok.kind != scanner.EOF {
		decl, err := p.parseTopDecl()
		if err != nil {
			return nil, err
		}
		file.Decls = append(file.Decls, decl)
	}

	return file, nil
}

func (p *Parser) parseTopDecl() (ast.Decl, error) {
	annotations, err := p.parseAnnotations()
	if err != nil {
		return nil, err
	}

	oneway := false
	if p.matchIdent("oneway") {
		oneway = true
		p.next()
	}

	switch {
	case p.matchIdent("interface"):
		return p.parseInterfaceDecl(annotations, oneway)
	case p.matchIdent("parcelable"):
		return p.parseParcelableDecl(annotations)
	case p.matchIdent("enum"):
		return p.parseEnumDecl(annotations)
	case p.matchIdent("union"):
		return p.parseUnionDecl(annotations)
	default:
		return nil, p.unexpected("declaration")
	}
}

func (p *Parser) parseInterfaceDecl(annotations []ast.Annotation, oneway bool) (*ast.InterfaceDecl, error) {
	p.next()
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}
	if err := p.expect("{"); err != nil {
		return nil, err
	}

	decl := &ast.InterfaceDecl{
		Annotations: annotations,
		Oneway:      oneway,
		Name:        name,
	}

	for !p.match("}") {
		member, err := p.parseInterfaceMember()
		if err != nil {
			return nil, err
		}
		decl.Members = append(decl.Members, member)
	}
	if err := p.expect("}"); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseInterfaceMember() (ast.InterfaceMember, error) {
	annotations, err := p.parseAnnotations()
	if err != nil {
		return nil, err
	}

	switch {
	case p.matchIdent("const"):
		return p.parseConstDecl(annotations)
	case p.matchIdent("parcelable"):
		return p.parseParcelableDecl(annotations)
	case p.matchIdent("enum"):
		return p.parseEnumDecl(annotations)
	case p.matchIdent("union"):
		return p.parseUnionDecl(annotations)
	default:
		return p.parseMethodDecl(annotations)
	}
}

func (p *Parser) parseMethodDecl(annotations []ast.Annotation) (*ast.MethodDecl, error) {
	oneway := false
	if p.matchIdent("oneway") {
		oneway = true
		p.next()
	}

	ret, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}
	if err := p.expect("("); err != nil {
		return nil, err
	}

	var args []ast.Field
	if !p.match(")") {
		for {
			arg, err := p.parseField(true, false)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.match(")") {
				break
			}
			if err := p.expect(","); err != nil {
				return nil, err
			}
		}
	}
	if err := p.expect(")"); err != nil {
		return nil, err
	}
	transaction := ""
	if p.match("=") {
		p.next()
		transaction, err = p.parseExpression(";")
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(";"); err != nil {
		return nil, err
	}

	return &ast.MethodDecl{
		Annotations: annotations,
		Oneway:      oneway,
		Return:      ret,
		Name:        name,
		Args:        args,
		Transaction: transaction,
	}, nil
}

func (p *Parser) parseConstDecl(annotations []ast.Annotation) (*ast.ConstDecl, error) {
	p.next()
	typ, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}
	if err := p.expect("="); err != nil {
		return nil, err
	}
	value, err := p.parseExpression(";")
	if err != nil {
		return nil, err
	}
	if err := p.expect(";"); err != nil {
		return nil, err
	}
	return &ast.ConstDecl{
		Annotations: annotations,
		Type:        typ,
		Name:        name,
		Value:       value,
	}, nil
}

func (p *Parser) parseParcelableDecl(annotations []ast.Annotation) (*ast.ParcelableDecl, error) {
	p.next()
	name, err := p.parseQualifiedName()
	if err != nil {
		return nil, err
	}
	decl := &ast.ParcelableDecl{
		Annotations: annotations,
		Name:        name,
	}
	if p.match("<") {
		params, err := p.parseTypeParamNames()
		if err != nil {
			return nil, err
		}
		decl.TypeParams = params
	}
	if !p.match(";") && !p.match("{") {
		if _, err := p.parseExpression(";", "{"); err != nil {
			return nil, err
		}
	}
	if p.match(";") {
		p.next()
		return decl, nil
	}
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	decl.Structured = true
	for !p.match("}") {
		annotations, err := p.parseAnnotations()
		if err != nil {
			return nil, err
		}
		switch {
		case p.matchIdent("const"):
			member, err := p.parseConstDecl(annotations)
			if err != nil {
				return nil, err
			}
			decl.Consts = append(decl.Consts, *member)
		case p.matchIdent("parcelable"):
			nested, err := p.parseParcelableDecl(annotations)
			if err != nil {
				return nil, err
			}
			decl.Decls = append(decl.Decls, nested)
		case p.matchIdent("enum"):
			nested, err := p.parseEnumDecl(annotations)
			if err != nil {
				return nil, err
			}
			decl.Decls = append(decl.Decls, nested)
		case p.matchIdent("union"):
			nested, err := p.parseUnionDecl(annotations)
			if err != nil {
				return nil, err
			}
			decl.Decls = append(decl.Decls, nested)
		default:
			field, err := p.parseFieldWithAnnotations(annotations, false, true)
			if err != nil {
				return nil, err
			}
			decl.Fields = append(decl.Fields, field)
			if err := p.expect(";"); err != nil {
				return nil, err
			}
		}
	}
	if err := p.expect("}"); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseTypeParamNames() ([]string, error) {
	if err := p.expect("<"); err != nil {
		return nil, err
	}
	var params []string
	for {
		name, err := p.parseName()
		if err != nil {
			return nil, err
		}
		params = append(params, name)
		if p.match(">") {
			break
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	if err := p.expect(">"); err != nil {
		return nil, err
	}
	return params, nil
}

func (p *Parser) parseEnumDecl(annotations []ast.Annotation) (*ast.EnumDecl, error) {
	p.next()
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	decl := &ast.EnumDecl{
		Annotations: annotations,
		Name:        name,
	}
	for !p.match("}") {
		memberName, err := p.parseName()
		if err != nil {
			return nil, err
		}
		member := ast.EnumMember{Name: memberName}
		if p.match("=") {
			p.next()
			member.Value, err = p.parseExpression(",", "}")
			if err != nil {
				return nil, err
			}
		}
		decl.Members = append(decl.Members, member)
		if p.match("}") {
			break
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	if err := p.expect("}"); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseUnionDecl(annotations []ast.Annotation) (*ast.UnionDecl, error) {
	p.next()
	name, err := p.parseName()
	if err != nil {
		return nil, err
	}
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	decl := &ast.UnionDecl{
		Annotations: annotations,
		Name:        name,
	}
	for !p.match("}") {
		field, err := p.parseField(false, false)
		if err != nil {
			return nil, err
		}
		decl.Fields = append(decl.Fields, field)
		if err := p.expect(";"); err != nil {
			return nil, err
		}
	}
	if err := p.expect("}"); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseField(allowDirection bool, allowDefault bool) (ast.Field, error) {
	annotations, err := p.parseAnnotations()
	if err != nil {
		return ast.Field{}, err
	}
	return p.parseFieldWithAnnotations(annotations, allowDirection, allowDefault)
}

func (p *Parser) parseFieldWithAnnotations(annotations []ast.Annotation, allowDirection bool, allowDefault bool) (ast.Field, error) {
	var err error

	var dir ast.Direction
	if allowDirection {
		switch {
		case p.matchIdent("in"):
			dir = ast.DirectionIn
			p.next()
		case p.matchIdent("out"):
			dir = ast.DirectionOut
			p.next()
		case p.matchIdent("inout"):
			dir = ast.DirectionInOut
			p.next()
		}
	}

	typ, err := p.parseTypeRef()
	if err != nil {
		return ast.Field{}, err
	}
	name, err := p.parseName()
	if err != nil {
		return ast.Field{}, err
	}
	field := ast.Field{
		Annotations: annotations,
		Direction:   dir,
		Type:        typ,
		Name:        name,
	}
	if allowDefault && p.match("=") {
		p.next()
		field.DefaultValue, err = p.parseExpression(";")
		if err != nil {
			return ast.Field{}, err
		}
	}
	return field, nil
}

func (p *Parser) parseTypeRef() (ast.TypeRef, error) {
	annotations, err := p.parseAnnotations()
	if err != nil {
		return ast.TypeRef{}, err
	}

	name, err := p.parseQualifiedName()
	if err != nil {
		return ast.TypeRef{}, err
	}
	typ := ast.TypeRef{
		Annotations: annotations,
		Name:        name,
	}

	for _, ann := range annotations {
		if ann.Name == "nullable" {
			typ.Nullable = true
		}
	}

	if p.match("<") {
		p.next()
		for {
			arg, err := p.parseTypeRef()
			if err != nil {
				return ast.TypeRef{}, err
			}
			typ.TypeArgs = append(typ.TypeArgs, arg)
			if p.match(">") {
				break
			}
			if err := p.expect(","); err != nil {
				return ast.TypeRef{}, err
			}
		}
		if err := p.expect(">"); err != nil {
			return ast.TypeRef{}, err
		}
	}

	if p.match("[") {
		p.next()
		typ.Array = true
		if !p.match("]") {
			value, err := p.parseIntLiteral()
			if err != nil {
				return ast.TypeRef{}, err
			}
			typ.FixedArrayLen = &value
		}
		if err := p.expect("]"); err != nil {
			return ast.TypeRef{}, err
		}
	}

	return typ, nil
}

func (p *Parser) parseAnnotations() ([]ast.Annotation, error) {
	var annotations []ast.Annotation
	for p.match("@") {
		p.next()
		name, err := p.parseQualifiedName()
		if err != nil {
			return nil, err
		}
		ann := ast.Annotation{Name: name}
		if p.match("(") {
			p.next()
			if !p.match(")") {
				for {
					arg := ast.AnnotationArg{}
					if p.tok.kind == scanner.Ident {
						next := p.peek()
						if next.text == "=" {
							arg.Name = p.tok.text
							p.next()
							p.next()
						}
					}
					value, err := p.parseExpression(",", ")")
					if err != nil {
						return nil, err
					}
					arg.Value = value
					ann.Args = append(ann.Args, arg)
					if p.match(")") {
						break
					}
					if err := p.expect(","); err != nil {
						return nil, err
					}
				}
			}
			if err := p.expect(")"); err != nil {
				return nil, err
			}
		}
		annotations = append(annotations, ann)
	}
	return annotations, nil
}

func (p *Parser) parseQualifiedName() (string, error) {
	name, err := p.parseName()
	if err != nil {
		return "", err
	}
	parts := []string{name}
	for p.match(".") {
		p.next()
		part, err := p.parseName()
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "."), nil
}

func (p *Parser) parseName() (string, error) {
	if p.tok.kind != scanner.Ident {
		return "", p.unexpected("identifier")
	}
	name := p.tok.text
	p.next()
	return name, nil
}

func (p *Parser) parseIntLiteral() (int, error) {
	value, err := p.parseExpression("]")
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(value, 0, 64)
	if err != nil {
		return 0, &Error{Pos: p.tok.pos, Msg: fmt.Sprintf("invalid integer literal %q", value)}
	}
	return int(n), nil
}

func (p *Parser) parseExpression(stoppers ...string) (string, error) {
	if len(stoppers) == 0 {
		return "", fmt.Errorf("parseExpression requires at least one stopper")
	}

	stopSet := make(map[string]struct{}, len(stoppers))
	for _, stopper := range stoppers {
		stopSet[stopper] = struct{}{}
	}

	var b strings.Builder
	depthParen := 0
	depthBracket := 0
	depthBrace := 0
	parsed := false

	for p.tok.kind != scanner.EOF {
		if depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
			if _, ok := stopSet[p.tok.text]; ok {
				break
			}
		}

		parsed = true
		switch p.tok.text {
		case "(":
			depthParen++
		case ")":
			if depthParen == 0 {
				return "", p.unexpected("expression")
			}
			depthParen--
		case "[":
			depthBracket++
		case "]":
			if depthBracket == 0 {
				return "", p.unexpected("expression")
			}
			depthBracket--
		case "{":
			depthBrace++
		case "}":
			if depthBrace == 0 {
				return "", p.unexpected("expression")
			}
			depthBrace--
		}

		b.WriteString(p.tok.text)
		p.next()
	}

	if !parsed {
		return "", p.unexpected("expression")
	}
	return strings.TrimSpace(b.String()), nil
}

func (p *Parser) match(text string) bool {
	return p.tok.text == text
}

func (p *Parser) matchIdent(text string) bool {
	return p.tok.kind == scanner.Ident && p.tok.text == text
}

func (p *Parser) expect(text string) error {
	if !p.match(text) {
		return p.unexpected(text)
	}
	p.next()
	return nil
}

func (p *Parser) next() {
	r := p.s.Scan()
	p.tok = token{
		text: p.s.TokenText(),
		kind: r,
		pos:  p.s.Pos(),
	}
}

func (p *Parser) peek() token {
	s := p.s
	r := s.Scan()
	return token{text: s.TokenText(), kind: r, pos: s.Pos()}
}

func (p *Parser) unexpected(want string) error {
	return &Error{
		Pos: p.tok.pos,
		Msg: fmt.Sprintf("unexpected token %q, want %s", p.tok.text, want),
	}
}
