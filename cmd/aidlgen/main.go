package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wdsgyj/libbinder-go/internal/aidl/ast"
	"github.com/wdsgyj/libbinder-go/internal/aidl/codegen"
	"github.com/wdsgyj/libbinder-go/internal/aidl/gomodel"
	"github.com/wdsgyj/libbinder-go/internal/aidl/ir"
	"github.com/wdsgyj/libbinder-go/internal/aidl/parser"
	"github.com/wdsgyj/libbinder-go/internal/aidl/resolve"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("aidlgen", flag.ContinueOnError)
	fs.SetOutput(stderr)

	format := fs.String("format", "summary", "output format: summary, ast, model, or go")
	outDir := fs.String("out", "", "output directory for generated files when -format go")
	typesPath := fs.String("types", "", "JSON sidecar for custom parcelable type mappings")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: aidlgen [-format summary|ast|model|go] [-out dir] <file.aidl> [more.aidl ...]")
		return 2
	}
	rootPaths := fs.Args()

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	var customParcelables map[string]gomodel.CustomParcelableConfig
	var stableInterfaces map[string]gomodel.StableInterfaceConfig
	var err error
	if *typesPath != "" {
		customParcelables, err = gomodel.LoadCustomParcelableMappings(*typesPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		stableInterfaces, err = gomodel.LoadStableInterfaceMappings(*typesPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	switch *format {
	case "ast":
		files, err := parseRootFiles(rootPaths)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := encodeRootFiles(enc, rootPaths, files); err != nil {
			fmt.Fprintf(stderr, "encode ast: %v\n", err)
			return 1
		}
	case "summary":
		files, err := parseRootFiles(rootPaths)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if len(rootPaths) == 1 {
			if err := enc.Encode(ir.Lower(files[rootPaths[0]])); err != nil {
				fmt.Fprintf(stderr, "encode summary: %v\n", err)
				return 1
			}
			break
		}
		summaries := make([]*ir.File, 0, len(rootPaths))
		for _, path := range rootPaths {
			summaries = append(summaries, ir.Lower(files[path]))
		}
		if err := enc.Encode(summaries); err != nil {
			fmt.Fprintf(stderr, "encode summary: %v\n", err)
			return 1
		}
	case "model":
		ordered, files, err := loadAIDLGraph(rootPaths)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := validateParsedFiles(stderr, ordered, files); err != nil {
			return 1
		}
		models := make([]*gomodel.File, 0, len(rootPaths))
		for _, path := range rootPaths {
			model, diags := gomodel.Lower(files[path], gomodel.LowerOptions{
				SourcePath:        path,
				CustomParcelables: customParcelables,
				StableInterfaces:  stableInterfaces,
				DependencyFiles:   dependencyFiles(path, ordered, files),
			})
			if len(diags) != 0 {
				_ = enc.Encode(diags)
				return 1
			}
			models = append(models, model)
		}
		if len(models) == 1 {
			if err := enc.Encode(models[0]); err != nil {
				fmt.Fprintf(stderr, "encode model: %v\n", err)
				return 1
			}
			break
		}
		if err := enc.Encode(models); err != nil {
			fmt.Fprintf(stderr, "encode model: %v\n", err)
			return 1
		}
	case "go":
		ordered, files, err := loadAIDLGraph(rootPaths)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := validateParsedFiles(stderr, ordered, files); err != nil {
			return 1
		}
		var outputs []codegen.OutputFile
		for _, path := range ordered {
			model, diags := gomodel.Lower(files[path], gomodel.LowerOptions{
				SourcePath:        path,
				CustomParcelables: customParcelables,
				StableInterfaces:  stableInterfaces,
				DependencyFiles:   dependencyFiles(path, ordered, files),
			})
			if len(diags) != 0 {
				diagEnc := json.NewEncoder(stderr)
				diagEnc.SetIndent("", "  ")
				_ = diagEnc.Encode(diags)
				return 1
			}
			rendered, err := codegen.RenderGo(model, codegen.GoOptions{
				TypeMappingsPath: *typesPath,
			})
			if err != nil {
				fmt.Fprintf(stderr, "generate go %s: %v\n", path, err)
				return 1
			}
			outputs = append(outputs, rendered...)
		}
		if *outDir == "" {
			if len(outputs) != 1 {
				fmt.Fprintf(stderr, "generate go: expected single output, got %d; use -out for multi-file generation\n", len(outputs))
				return 1
			}
			if _, err := stdout.Write(outputs[0].Content); err != nil {
				fmt.Fprintf(stderr, "write go output: %v\n", err)
				return 1
			}
			break
		}
		for _, output := range outputs {
			dst := filepath.Join(*outDir, output.Path)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				fmt.Fprintf(stderr, "mkdir %s: %v\n", filepath.Dir(dst), err)
				return 1
			}
			if err := os.WriteFile(dst, output.Content, 0o644); err != nil {
				fmt.Fprintf(stderr, "write %s: %v\n", dst, err)
				return 1
			}
		}
	default:
		fmt.Fprintf(stderr, "unknown format %q\n", *format)
		return 2
	}

	return 0
}

func parseRootFiles(paths []string) (map[string]*ast.File, error) {
	files := make(map[string]*ast.File, len(paths))
	for _, path := range paths {
		file, err := parseAIDLFile(path)
		if err != nil {
			return nil, err
		}
		files[path] = file
	}
	return files, nil
}

func encodeRootFiles(enc *json.Encoder, paths []string, files map[string]*ast.File) error {
	if len(paths) == 1 {
		return enc.Encode(files[paths[0]])
	}
	ordered := make([]*ast.File, 0, len(paths))
	for _, path := range paths {
		ordered = append(ordered, files[path])
	}
	return enc.Encode(ordered)
}

func validateParsedFiles(stderr io.Writer, ordered []string, files map[string]*ast.File) error {
	for _, path := range ordered {
		if diags := resolve.ValidateFile(files[path]); len(diags) != 0 {
			enc := json.NewEncoder(stderr)
			enc.SetIndent("", "  ")
			_ = enc.Encode(diags)
			return fmt.Errorf("validation failed")
		}
	}
	return nil
}

func dependencyFiles(current string, ordered []string, files map[string]*ast.File) []*ast.File {
	deps := make([]*ast.File, 0, len(ordered)-1)
	for _, path := range ordered {
		if path == current {
			continue
		}
		deps = append(deps, files[path])
	}
	return deps
}

func parseAIDLFile(path string) (*ast.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %v", path, err)
	}
	file, err := parser.Parse(path, string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %v", path, err)
	}
	return file, nil
}

func loadAIDLGraph(rootPaths []string) ([]string, map[string]*ast.File, error) {
	ordered := make([]string, 0, len(rootPaths))
	files := map[string]*ast.File{}
	var searchRoots []string

	var visit func(string) error
	visit = func(path string) error {
		path = filepath.Clean(path)
		if _, ok := files[path]; ok {
			return nil
		}
		file, err := parseAIDLFile(path)
		if err != nil {
			return err
		}
		files[path] = file
		ordered = append(ordered, path)
		searchRoots = append(searchRoots, sourceRootsFor(path, file)...)
		searchRoots = uniqueStrings(searchRoots)
		localRoots := uniqueStrings(append([]string{}, append(searchRoots, sourceRootsFor(path, file)...)...))
		for _, imp := range file.Imports {
			depPath, err := resolveImportFile(imp.Path, localRoots)
			if err != nil {
				continue
			}
			if err := visit(depPath); err != nil {
				return err
			}
		}
		return nil
	}

	for _, path := range rootPaths {
		if err := visit(path); err != nil {
			return nil, nil, err
		}
	}
	return ordered, files, nil
}

func sourceRootsFor(path string, file *ast.File) []string {
	roots := []string{filepath.Clean(filepath.Dir(path))}
	if file == nil || file.PackageName == "" {
		return uniqueStrings(roots)
	}
	pkgDir := filepath.Join(strings.Split(file.PackageName, ".")...)
	dir := filepath.Clean(filepath.Dir(path))
	if hasPathSuffix(dir, pkgDir) {
		root := strings.TrimSuffix(dir, pkgDir)
		root = strings.TrimSuffix(root, string(filepath.Separator))
		if root == "" {
			root = "."
		}
		roots = append(roots, filepath.Clean(root))
	}
	return uniqueStrings(roots)
}

func hasPathSuffix(path string, suffix string) bool {
	path = filepath.Clean(path)
	suffix = filepath.Clean(suffix)
	if path == suffix {
		return true
	}
	return strings.HasSuffix(path, string(filepath.Separator)+suffix)
}

func resolveImportFile(importPath string, searchRoots []string) (string, error) {
	rel := filepath.Join(strings.Split(importPath, ".")...) + ".aidl"
	for _, root := range searchRoots {
		candidate := filepath.Clean(filepath.Join(root, rel))
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("not found under search roots %v", searchRoots)
}

func uniqueStrings(values []string) []string {
	set := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		value = filepath.Clean(value)
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
