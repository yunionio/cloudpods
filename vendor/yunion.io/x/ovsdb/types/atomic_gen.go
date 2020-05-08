package types

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type AtomicGen struct {
	outDir string
	gopkg  string
}

func NewAtomicGen() *AtomicGen {
	return &AtomicGen{}
}

func (g *AtomicGen) OutDir(outDir string) *AtomicGen {
	d, gopkg := filepath.Split(outDir)
	gopkg = strings.ReplaceAll(gopkg, "-", "_")
	outDir = filepath.Join(d, gopkg)
	g.outDir = outDir
	g.gopkg = gopkg
	return g
}

func (ag *AtomicGen) Gen() error {
	if err := ag.genEnsures(); err != nil {
		return err
	}
	if err := ag.genCmdArgs(); err != nil {
		return err
	}
	if err := ag.genMatches(); err != nil {
		return err
	}
	return nil
}

func (ag *AtomicGen) prepGen(fname string) (writer, error) {
	os.MkdirAll(ag.outDir, os.FileMode(0755))
	fpath := filepath.Join(ag.outDir, fname)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return writer{}, errors.Wrapf(err, "open %s", fpath)
	}
	w := writer{
		w: f,
	}
	w.Writef(`// DO NOT EDIT: automatically generated code`)
	w.Writef(``)
	w.Writef(`package %s`, ag.gopkg)
	return w, nil
}

func (ag *AtomicGen) genEnsures() error {
	w, err := ag.prepGen("atomic_gen_ensures_zz_generated.go")
	if err != nil {
		return err
	}
	ag.genEnsures_(w)
	return nil
}

func (ag *AtomicGen) genEnsures_(w writer) {
	w.Writeln(`import "github.com/pkg/errors"`)

	w.Writeln(`func panicErr(msg string) {`)
	w.Writeln(`	panic(errors.Wrap(ErrBadType, msg))`)
	w.Writeln(`}`)
	w.Writeln(``)

	w.Writeln(`func panicErrf(fmtStr string, s ...interface{}) {`)
	w.Writeln(`	panic(errors.Wrapf(ErrBadType, fmtStr, s...))`)
	w.Writeln(`}`)
	w.Writeln(``)

	w.Writeln(`func ensureTypedPair(val interface{}) (string, interface{}) {`)
	w.Writeln(`	arr, ok := val.([]interface{})`)
	w.Writeln(`	if !ok {`)
	w.Writeln(`		panicErr("ensureTypedPair: not an array")`)
	w.Writeln(`	}`)
	w.Writeln(`	if len(arr) != 2 {`)
	w.Writeln(`		panicErrf("ensureTypedPair: length is %d, want 2", len(arr))`)
	w.Writeln(`	}`)
	w.Writeln(`	typ, ok := arr[0].(string)`)
	w.Writeln(`	if !ok {`)
	w.Writeln(`		panicErr("ensureTypedPair: type not a string")`)
	w.Writeln(`	}`)
	w.Writeln(`	return typ, arr[1]`)
	w.Writeln(`}`)
	w.Writeln(``)

	w.Writeln(`func ensureTyped(val interface{}, typ string) interface{} {`)
	w.Writeln(`	gotTyp, r := ensureTypedPair(val)`)
	w.Writeln(`	if gotTyp != typ {`)
	w.Writeln(`		panicErrf("ensureMultiples: got %s, want %s", gotTyp, typ)`)
	w.Writeln(`	}`)
	w.Writeln(`	return r`)
	w.Writeln(`}`)
	w.Writeln(``)

	w.Writeln(`func ensureMultiples(val interface{}, typ string) []interface{} {`)
	w.Writeln(`	val = ensureTyped(val, typ)`)
	w.Writeln(`	mulVal, ok := val.([]interface{})`)
	w.Writeln(`	if !ok {`)
	w.Writeln(`		panicErr("ensureMultiples: val is not an array")`)
	w.Writeln(`	}`)
	w.Writeln(`	return mulVal`)
	w.Writeln(`}`)
	w.Writeln(``)

	w.Writeln(`func probeEmpty(val interface{}) (r bool) {`)
	w.Writeln(`	defer func() {`)
	w.Writeln(`		recover()`)
	w.Writeln(`	}()`)
	w.Writeln(`	empty := ensureMultiples(val, "set")`)
	w.Writeln(`	if len(empty) != 0 {`)
	w.Writeln(`		return false`)
	w.Writeln(`	}`)
	w.Writeln(`	return true`)
	w.Writeln(`}`)
	w.Writeln(``)

	w.Writeln(`func EnsureUuid(val interface{}) string {`)
	w.Writeln(`	val = ensureTyped(val, "uuid")`)
	w.Writeln(`	r, ok := val.(string)`)
	w.Writeln(`	if !ok {`)
	w.Writeln(`		panicErr("bad uuid value")`)
	w.Writeln(`	}`)
	w.Writeln(`	return r`)
	w.Writeln(`}`)
	w.Writeln(``)

	w.Writeln(`func EnsureUuidMultiples(val interface{}) []string {`)
	w.Writeln(`	typ, val1 := ensureTypedPair(val)`)
	w.Writeln(`	if typ == "uuid" {`)
	w.Writeln(`		r, ok := val1.(string)`)
	w.Writeln(`		if !ok {`)
	w.Writeln(`			panicErr("uuid multiples: expect a string")`)
	w.Writeln(`		}`)
	w.Writeln(`		return []string{r}`)
	w.Writeln(`	}`)
	w.Writeln(`	if typ == "set" {`)
	w.Writeln(`		mulVal, ok := val1.([]interface{})`)
	w.Writeln(`		if !ok {`)
	w.Writeln(`			panicErr("uuid multiples: expect an array")`)
	w.Writeln(`		}`)
	w.Writeln(`		if len(mulVal) == 0 {`)
	w.Writeln(`			return nil`)
	w.Writeln(`		}`)
	w.Writeln(`		r := make([]string, len(mulVal))`)
	w.Writeln(`		for i, val := range mulVal {`)
	w.Writeln(`			r[i] = EnsureUuid(val)`)
	w.Writeln(`		}`)
	w.Writeln(`		return r`)
	w.Writeln(`	}`)
	w.Writeln(`	panic("uuid multiple: unexpected type: " + typ)`)
	w.Writeln(`}`)
	w.Writeln(``)

	for _, atom0 := range atomics {
		var (
			name0   = atom0.exportName()
			gotyp0  = atomicGoMap[atom0]
			unmTyp0 = atomicUnmarshalTyp[atom0]
		)

		if atom0 != Uuid {
			w.Writef(`func Ensure%s(val interface{}) %s {`, name0, gotyp0)
			w.Writef(`	if r, ok := val.(%s); ok {`, unmTyp0)
			w.Writef(`		return %s(r)`, gotyp0)
			w.Writef(`	}`)
			w.Writef(`	panic(ErrBadType)`)
			w.Writef(`}`)
			w.Writef(``)

			w.Writef(`func Ensure%sMultiples(val interface{}) []%s {`, name0, gotyp0)
			w.Writef(`	if ok := probeEmpty(val); ok {`)
			w.Writef(`		return nil`)
			w.Writef(`	}`)
			w.Writef(`	if r, ok := val.(%s); ok {`, unmTyp0)
			w.Writef(`		return []%s{%s(r)}`, gotyp0, gotyp0)
			w.Writef(`	}`)
			w.Writef(`	mulVal := ensureMultiples(val, "set")`)
			w.Writef(`	if len(mulVal) == 0 {`)
			w.Writef(`		return nil`)
			w.Writef(`	}`)
			w.Writef(`	r := make([]%s, len(mulVal))`, gotyp0)
			w.Writef(`	for i, val := range mulVal {`)
			w.Writef(`		r[i] = Ensure%s(val)`, name0)
			w.Writef(`	}`)
			w.Writef(`	return r`)
			w.Writef(`}`)
			w.Writef(``)
		}

		w.Writef(`func Ensure%sOptional(val interface{}) *%s {`, name0, gotyp0)
		w.Writef(`	if ok := probeEmpty(val); ok {`)
		w.Writef(`		return nil`)
		w.Writef(`	}`)
		w.Writef(`	r := Ensure%s(val)`, name0)
		w.Writef(`	return &r`)
		w.Writef(`}`)
		w.Writef(``)

		for _, atom1 := range atomics {
			var (
				name1  = atom1.exportName()
				gotyp1 = atomicGoMap[atom1]
			)
			w.Writef(`func EnsureMap%s%s(val interface{}) map[%s]%s {`, name0, name1, gotyp0, gotyp1)
			w.Writef(`	mulVal := ensureMultiples(val, "map")`)
			w.Writef(`	if len(mulVal) == 0 {`)
			w.Writef(`		return nil`)
			w.Writef(`	}`)
			w.Writef(`	r := map[%s]%s{}`, gotyp0, gotyp1)
			w.Writef(`	for _, pairVal := range mulVal {`)
			w.Writef(`		pair, ok := pairVal.([]interface{})`)
			w.Writef(`		if !ok {`)
			w.Writef(`			panicErr("map: not an array")`)
			w.Writef(`		}`)
			w.Writef(`		if len(pair) != 2 {`)
			w.Writef(`			panicErr("map: not a pair")`)
			w.Writef(`		}`)
			w.Writef(`		k := Ensure%s(pair[0])`, name0)
			w.Writef(`		v := Ensure%s(pair[1])`, name1)
			w.Writef(`		r[k] = v`)
			w.Writef(`	}`)
			w.Writef(`	return r`)
			w.Writef(`}`)
			w.Writef(``)
		}
	}
}

func (ag *AtomicGen) genCmdArgs() error {
	w, err := ag.prepGen("atomic_gen_cmdargs_zz_generated.go")
	if err != nil {
		return err
	}
	ag.genCmdArgs_(w)
	return nil
}

func (ag *AtomicGen) genCmdArgs_(w writer) {
	w.Writeln(`import "fmt"`)
	w.Writeln(`import "strings"`)

	for _, atom0 := range atomics {
		var (
			name0  = atom0.exportName()
			gotyp0 = atomicGoMap[atom0]
			fmt0   = atom0.fmtStr()
		)

		w.Writef(`func OvsdbCmdArg%s(a %s) string {`, name0, gotyp0)
		w.Writef(`	return fmt.Sprintf("%s", a)`, fmt0)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func OvsdbCmdArgs%s(field string, a %s) []string {`, name0, gotyp0)
		w.Writef(`	return []string{fmt.Sprintf("%%s=%%s", field, OvsdbCmdArg%s(a))}`, name0)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func OvsdbCmdArgs%sOptional(field string, a *%s) []string {`, name0, gotyp0)
		w.Writeln(`	if a == nil {`)
		w.Writeln(`		return nil`)
		w.Writeln(`	}`)
		w.Writef(`	return OvsdbCmdArgs%s(field, *a)`, name0)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func OvsdbCmdArgs%sMultiples(field string, a []%s) []string {`, name0, gotyp0)
		w.Writeln(`	if len(a) == 0 {`)
		w.Writeln(`		return nil`)
		w.Writeln(`	}`)
		w.Writeln(`	elArgs := make([]string, len(a))`)
		w.Writeln(`	for i, el := range a {`)
		w.Writef(`		elArgs[i] = OvsdbCmdArg%s(el)`, name0)
		w.Writeln(`	}`)
		w.Writeln(`	arg := fmt.Sprintf("%s=[%s]", field, strings.Join(elArgs, ","))`)
		w.Writeln(`	return []string{arg}`)
		w.Writeln(`}`)
		w.Writeln(``)

		for _, atom1 := range atomics {
			var (
				name1  = atom1.exportName()
				gotyp1 = atomicGoMap[atom1]
			)
			w.Writef(`func OvsdbCmdArgsMap%s%s(field string, a map[%s]%s) []string {`, name0, name1, gotyp0, gotyp1)
			w.Writeln(`	if len(a) == 0 {`)
			w.Writeln(`		return nil`)
			w.Writeln(`	}`)
			w.Writeln(`	r := make([]string, 0, len(a))`)
			w.Writeln(`	for aK, aV := range a {`)
			w.Writef(`		r = append(r, fmt.Sprintf("%%s:%%s=%%s", field, OvsdbCmdArg%s(aK), OvsdbCmdArg%s(aV)))`, name0, name1)
			w.Writeln(`	}`)
			w.Writeln(`	return r`)
			w.Writeln(`}`)
			w.Writeln(``)
		}
	}
}

func (ag *AtomicGen) genMatches() error {
	w, err := ag.prepGen("atomic_gen_matches_zz_generated.go")
	if err != nil {
		return err
	}
	ag.genMatches_(w)
	return nil
}

func (ag *AtomicGen) genMatches_(w writer) {
	for _, atom0 := range atomics {
		var (
			name0  = atom0.exportName()
			gotyp0 = atomicGoMap[atom0]
		)

		w.Writef(`func Match%sIfNonZero(a, b %s) bool {`, name0, gotyp0)
		w.Writef(`	var z %s`, gotyp0)
		w.Writeln(`	if b == z {`)
		w.Writeln(`		return true`)
		w.Writeln(`	}`)
		w.Writef(`	return Match%s(a, b)`, name0)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func Match%s(a, b %s) bool {`, name0, gotyp0)
		w.Writeln(`	return a == b`)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func Match%sOptionalIfNonZero(a, b *%s) bool {`, name0, gotyp0)
		w.Writeln(`	if b == nil {`)
		w.Writeln(`		return true`)
		w.Writeln(`	}`)
		w.Writef(`	return Match%sOptional(a, b)`, name0)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func Match%sOptional(a, b *%s) bool {`, name0, gotyp0)
		w.Writeln(`	if a == nil && b == nil {`)
		w.Writeln(`		return true`)
		w.Writeln(`	} else if a != nil && b != nil {`)
		w.Writeln(`		return *a == *b`)
		w.Writeln(`	}`)
		w.Writeln(`	return false`)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func Match%sMultiplesIfNonZero(a, b []%s) bool {`, name0, gotyp0)
		w.Writeln(`	if b == nil {`)
		w.Writeln(`		return true`)
		w.Writeln(`	}`)
		w.Writef(`	return Match%sMultiples(a, b)`, name0)
		w.Writeln(`}`)
		w.Writeln(``)

		w.Writef(`func Match%sMultiples(a, b []%s) bool {`, name0, gotyp0)
		w.Writeln(`	if len(a) != len(b) {`)
		w.Writeln(`		return false`)
		w.Writeln(`	}`)
		w.Writef(`	bCopy := make([]%s, len(b))`, gotyp0)
		w.Writeln(`	copy(bCopy, b)`)
		w.Writeln(`	for _, elA := range a {`)
		w.Writeln(`		for i := len(bCopy) - 1; i >= 0; i-- {`)
		w.Writeln(`			elB := bCopy[i]`)
		w.Writeln(`			if elA == elB {`)
		w.Writeln(`				bCopy = append(bCopy[:i], bCopy[i+1:]...)`)
		w.Writeln(`			}`)
		w.Writeln(`		}`)
		w.Writeln(`	}`)
		w.Writeln(`	if len(bCopy) == 0 {`)
		w.Writeln(`		return true`)
		w.Writeln(`	}`)
		w.Writeln(`	return false`)
		w.Writeln(`}`)
		w.Writeln(``)

		for _, atom1 := range atomics {
			var (
				name1  = atom1.exportName()
				gotyp1 = atomicGoMap[atom1]
			)
			w.Writef(`func MatchMap%s%sIfNonZero(a, b map[%s]%s) bool {`, name0, name1, gotyp0, gotyp1)
			w.Writeln(`	if b == nil {`)
			w.Writeln(`		return true`)
			w.Writeln(`	}`)
			w.Writef(`	return MatchMap%s%s(a, b)`, name0, name1)
			w.Writeln(`}`)
			w.Writeln(``)

			w.Writef(`func MatchMap%s%s(a, b map[%s]%s) bool {`, name0, name1, gotyp0, gotyp1)
			w.Writeln(`	if len(a) != len(b) {`)
			w.Writeln(`		return false`)
			w.Writeln(`	}`)
			w.Writef(`	bCopy := map[%s]%s{}`, gotyp0, gotyp1)
			w.Writeln(`	for k, v := range b {`)
			w.Writeln(`		bCopy[k] = v`)
			w.Writeln(`	}`)
			w.Writeln(`	for aK, aV := range a {`)
			w.Writeln(`		if bV, ok := bCopy[aK]; !ok || aV != bV {`)
			w.Writeln(`			return false`)
			w.Writeln(`		} else {`)
			w.Writeln(`			delete(bCopy, aK)`)
			w.Writeln(`		}`)
			w.Writeln(`	}`)
			w.Writeln(`	if len(bCopy) == 0 {`)
			w.Writeln(`		return true`)
			w.Writeln(`	}`)
			w.Writeln(`	return false`)
			w.Writeln(`}`)
			w.Writeln(``)
		}
	}
}
