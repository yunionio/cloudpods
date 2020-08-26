package types

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Generator struct {
	schema *Schema

	outDir string
	gopkg  string
}

func NewGenerator() *Generator {
	g := &Generator{}
	return g
}

func (g *Generator) OutDir(outDir string) *Generator {
	d, gopkg := filepath.Split(outDir)
	gopkg = strings.ReplaceAll(gopkg, "-", "_")
	outDir = filepath.Join(d, gopkg)
	g.outDir = outDir
	g.gopkg = gopkg
	return g
}

func (g *Generator) Schema(schema *Schema) *Generator {
	g.schema = schema
	return g
}

func (g *Generator) Gen() error {
	os.MkdirAll(g.outDir, os.FileMode(0755))
	fname := filepath.Join(g.outDir, "schema.go")
	fSchema, err := os.OpenFile(fname, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return errors.Wrapf(err, "open %s", fname)
	}
	w := writer{
		w: fSchema,
	}
	w.Writef(`// DO NOT EDIT: automatically generated code`)
	w.Writef(``)
	w.Writef(`package %s`, g.gopkg)
	g.schema.gen(w)
	return nil
}
