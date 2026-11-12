// Command gettext extracts Go and server-template messages as a POT on stdout.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template/parse"
)

type entry struct {
	id, plural string
	refs       map[string]bool
}
type catalog map[string]*entry

func (c catalog) add(id, plural, ref string) {
	if id == "" {
		log.Fatalf("%s: empty or nonliteral Gettext message", ref)
	}
	e := c[id]
	if e == nil {
		e = &entry{id: id, plural: plural, refs: map[string]bool{}}
		c[id] = e
	}
	if e.plural != "" && plural != "" && e.plural != plural {
		log.Fatalf("%s: inconsistent plural for %q", ref, id)
	}
	if plural != "" {
		e.plural = plural
	}
	e.refs[ref] = true
}

func main() {
	domain := flag.String("domain", "backend", "backend or templates")
	flag.Parse()
	if *domain != "backend" && *domain != "templates" {
		log.Fatal("domain must be backend or templates")
	}
	c := catalog{}
	if err := extractGo(*domain, c); err != nil {
		log.Fatal(err)
	}
	if *domain == "templates" {
		for _, root := range []string{"static/public/web-templates", "static/email-templates"} {
			err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || !strings.HasSuffix(p, ".html") {
					return nil
				}
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				return extractTemplate(p, string(b), c)
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	fmt.Print("# LibreDesk " + *domain + " messages.\nmsgid \"\"\nmsgstr \"\"\n\"Project-Id-Version: LibreDesk\\n\"\n\"MIME-Version: 1.0\\n\"\n\"Content-Type: text/plain; charset=UTF-8\\n\"\n\"Content-Transfer-Encoding: 8bit\\n\"\n\n")
	ids := make([]string, 0, len(c))
	for id := range c {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e := c[id]
		refs := make([]string, 0, len(e.refs))
		for ref := range e.refs {
			refs = append(refs, ref)
		}
		sort.Strings(refs)
		fmt.Println("#: " + strings.Join(refs, " "))
		if strings.Contains(id+e.plural, "%") {
			fmt.Println("#, go-format")
		}
		fmt.Printf("msgid %s\n", strconv.Quote(id))
		if e.plural != "" {
			fmt.Printf("msgid_plural %s\nmsgstr[0] \"\"\nmsgstr[1] \"\"\n\n", strconv.Quote(e.plural))
		} else {
			fmt.Print("msgstr \"\"\n\n")
		}
	}
}

// Use type information so unrelated Get methods (SQL, Redis, etc.) never enter
// the catalog, and translator variable names do not affect extraction.
func extractGo(domain string, c catalog) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "list", "-deps", "-export", "-json", "./cmd", "./internal/...")
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return err
	}
	type pkg struct {
		Dir, ImportPath, Export string
		GoFiles                 []string
		DepOnly                 bool
	}
	var packages []pkg
	exports := map[string]string{}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	for {
		var p pkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		exports[p.ImportPath] = p.Export
		if !p.DepOnly {
			packages = append(packages, p)
		}
	}
	fset := token.NewFileSet()
	imp := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) { return os.Open(exports[path]) })
	for _, p := range packages {
		var files []*ast.File
		for _, name := range p.GoFiles {
			filename, err := filepath.Rel(root, filepath.Join(p.Dir, name))
			if err != nil {
				return err
			}
			f, err := parser.ParseFile(fset, filename, nil, 0)
			if err != nil {
				return err
			}
			files = append(files, f)
		}
		info := &types.Info{Selections: map[*ast.SelectorExpr]*types.Selection{}}
		conf := types.Config{Importer: imp}
		if _, err := conf.Check(p.ImportPath, fset, files, info); err != nil {
			return err
		}
		for _, f := range files {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				selection := info.Selections[sel]
				if selection == nil || selection.Obj().Pkg() == nil || selection.Obj().Pkg().Path() != "github.com/leonelquinteros/gotext" {
					return true
				}
				start, plural, target := 0, false, "backend"
				switch sel.Sel.Name {
				case "Get":
				case "GetN":
					plural = true
				case "GetD", "GetND":
					target = literal(call.Args[0])
					start = 1
					plural = sel.Sel.Name == "GetND"
				default:
					return true
				}
				ref := fset.Position(call.Pos()).String()
				if target != "backend" && target != "templates" {
					log.Fatalf("%s: unsupported Gettext domain %q", ref, target)
				}
				if target != domain {
					return true
				}
				id, pl := literal(call.Args[start]), ""
				if plural {
					pl = literal(call.Args[start+1])
					if pl == "" {
						log.Fatalf("%s: plural must be a nonempty literal", ref)
					}
				}
				c.add(id, pl, ref)
				return true
			})
		}
	}
	return nil
}

func literal(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, _ := strconv.Unquote(lit.Value)
		return s
	}
	return ""
}

// Parse Go template syntax, including nested pipelines, variables, raw strings,
// and definitions. SkipFuncCheck allows the application's custom functions.
func extractTemplate(filename, source string, c catalog) error {
	tree := parse.New(filename)
	tree.Mode = parse.SkipFuncCheck
	trees := map[string]*parse.Tree{}
	if _, err := tree.Parse(source, "", "", trees); err != nil {
		return err
	}
	var walk func(parse.Node) error
	walk = func(node parse.Node) error {
		if node == nil {
			return nil
		}
		switch n := node.(type) {
		case *parse.ListNode:
			if n != nil {
				for _, child := range n.Nodes {
					if err := walk(child); err != nil {
						return err
					}
				}
			}
		case *parse.ActionNode:
			return walk(n.Pipe)
		case *parse.PipeNode:
			if n != nil {
				for _, cmd := range n.Cmds {
					if err := walk(cmd); err != nil {
						return err
					}
				}
			}
		case *parse.CommandNode:
			method := ""
			switch head := n.Args[0].(type) {
			case *parse.FieldNode:
				method = head.Ident[len(head.Ident)-1]
			case *parse.VariableNode:
				method = head.Ident[len(head.Ident)-1]
			case *parse.ChainNode:
				method = head.Field[len(head.Field)-1]
			}
			if method == "GetD" || method == "GetND" {
				ref := fmt.Sprintf("%s:%d", filename, 1+strings.Count(source[:int(n.Position())], "\n"))
				value := func(i int) string {
					if i < len(n.Args) {
						if s, ok := n.Args[i].(*parse.StringNode); ok {
							return s.Text
						}
					}
					return ""
				}
				if value(1) != "templates" {
					return fmt.Errorf("%s: template calls must use the literal templates domain", ref)
				}
				pl := ""
				if method == "GetND" {
					pl = value(3)
					if pl == "" {
						return fmt.Errorf("%s: plural must be a nonempty literal", ref)
					}
				}
				c.add(value(2), pl, ref)
			} else if method == "Get" || method == "GetN" {
				return fmt.Errorf("%s: template translations must use GetD/GetND with templates domain", filename)
			}
			for _, arg := range n.Args {
				if err := walk(arg); err != nil {
					return err
				}
			}
		case *parse.IfNode:
			for _, child := range []parse.Node{n.Pipe, n.List, n.ElseList} {
				if err := walk(child); err != nil {
					return err
				}
			}
		case *parse.RangeNode:
			for _, child := range []parse.Node{n.Pipe, n.List, n.ElseList} {
				if err := walk(child); err != nil {
					return err
				}
			}
		case *parse.WithNode:
			for _, child := range []parse.Node{n.Pipe, n.List, n.ElseList} {
				if err := walk(child); err != nil {
					return err
				}
			}
		case *parse.TemplateNode:
			return walk(n.Pipe)
		case *parse.ChainNode:
			return walk(n.Node)
		}
		return nil
	}
	for _, t := range trees {
		if err := walk(t.Root); err != nil {
			return err
		}
	}
	return nil
}
