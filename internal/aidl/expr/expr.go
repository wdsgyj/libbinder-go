package expr

import (
	"fmt"
	goast "go/ast"
	"go/constant"
	goparser "go/parser"
	"go/token"
	"regexp"
	"strings"
)

type Env map[string]constant.Value

func Normalize(src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}
	src = strings.ReplaceAll(src, "~", "^")
	src = numericLiteralSuffixRE.ReplaceAllString(src, "$1")
	return src
}

var numericLiteralSuffixRE = regexp.MustCompile(`(?i)\b((?:\d+\.\d*|\.\d+|\d+)(?:[eE][+\-]?\d+)?|0[xX][0-9a-fA-F]+)[fLdD]\b`)

func Parse(src string) (goast.Expr, error) {
	src = Normalize(src)
	if src == "" {
		return nil, fmt.Errorf("empty expression")
	}
	expr, err := goparser.ParseExpr(src)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

func Eval(src string, env Env) (constant.Value, error) {
	node, err := Parse(src)
	if err != nil {
		return nil, err
	}
	return evalNode(node, env)
}

func EvalInt64(src string, env Env) (int64, error) {
	v, err := Eval(src, env)
	if err != nil {
		return 0, err
	}
	if v == nil {
		return 0, fmt.Errorf("nil constant")
	}
	if v.Kind() != constant.Int {
		return 0, fmt.Errorf("not an integer constant")
	}
	out, ok := constant.Int64Val(v)
	if !ok {
		return 0, fmt.Errorf("integer constant out of int64 range")
	}
	return out, nil
}

func EvalBool(src string, env Env) (bool, error) {
	v, err := Eval(src, env)
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, fmt.Errorf("nil constant")
	}
	if v.Kind() != constant.Bool {
		return false, fmt.Errorf("not a boolean constant")
	}
	return constant.BoolVal(v), nil
}

func evalNode(node goast.Expr, env Env) (constant.Value, error) {
	switch n := node.(type) {
	case *goast.BasicLit:
		return constant.MakeFromLiteral(n.Value, n.Kind, 0), nil
	case *goast.Ident:
		switch n.Name {
		case "true":
			return constant.MakeBool(true), nil
		case "false":
			return constant.MakeBool(false), nil
		}
		v, ok := env[n.Name]
		if !ok {
			return nil, fmt.Errorf("unresolved identifier %q", n.Name)
		}
		return v, nil
	case *goast.SelectorExpr:
		name, ok := selectorName(n)
		if !ok {
			return nil, fmt.Errorf("unsupported selector expression")
		}
		v, ok := env[name]
		if !ok {
			return nil, fmt.Errorf("unresolved selector %q", name)
		}
		return v, nil
	case *goast.ParenExpr:
		return evalNode(n.X, env)
	case *goast.UnaryExpr:
		x, err := evalNode(n.X, env)
		if err != nil {
			return nil, err
		}
		return constant.UnaryOp(n.Op, x, 0), nil
	case *goast.BinaryExpr:
		x, err := evalNode(n.X, env)
		if err != nil {
			return nil, err
		}
		y, err := evalNode(n.Y, env)
		if err != nil {
			return nil, err
		}
		if n.Op == token.SHL || n.Op == token.SHR {
			shift, ok := constant.Uint64Val(y)
			if !ok {
				return nil, fmt.Errorf("invalid shift count")
			}
			return constant.Shift(x, n.Op, uint(shift)), nil
		}
		return constant.BinaryOp(x, n.Op, y), nil
	default:
		return nil, fmt.Errorf("unsupported expression %T", node)
	}
}

func selectorName(node *goast.SelectorExpr) (string, bool) {
	if node == nil {
		return "", false
	}
	parts := []string{node.Sel.Name}
	for {
		switch x := node.X.(type) {
		case *goast.Ident:
			parts = append(parts, x.Name)
			for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
				parts[i], parts[j] = parts[j], parts[i]
			}
			return strings.Join(parts, "."), true
		case *goast.SelectorExpr:
			node = x
			parts = append(parts, node.Sel.Name)
		default:
			return "", false
		}
	}
}
