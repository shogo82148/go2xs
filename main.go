package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Variable struct {
	Type string
	Name string
}

type Generator struct {
	name      string
	xsbuf     *bytes.Buffer
	variables []*Variable
}

func NewGenerator(name string) *Generator {
	gen := &Generator{
		name:      name,
		xsbuf:     &bytes.Buffer{},
		variables: []*Variable{},
	}

	fmt.Fprintln(gen.xsbuf, `#define PERL_NO_GET_CONTEXT
#include "EXTERN.h"
#include "perl.h"
#include "XSUB.h"

#include "ppport.h"`)
	fmt.Fprintf(gen.xsbuf, "#include \"lib%s.h\"\n", name)

	fmt.Fprintln(gen.xsbuf, `
#define convert_sv_to_int8(sv)    (GoInt8)SvIV(sv)
#define convert_sv_to_uint8(sv)   (GoUint8)SvUV(sv)
#define convert_sv_to_int16(sv)   (GoInt16)SvIV(sv)
#define convert_sv_to_uint16(sv)  (GoUint16)SvUV(sv)
#define convert_sv_to_int32(sv)   (GoInt32)SvIV(sv)
#define convert_sv_to_uint32(sv)  (GoUint32)SvUV(sv)
#define convert_sv_to_int64(sv)   (GoInt64)SvIV(sv)
#define convert_sv_to_uint64(sv)  (GoUint64)SvUV(sv)
#define convert_sv_to_int(sv)     (GoInt)SvIV(sv)
#define convert_sv_to_uint(sv)    (GoUint)SvUV(sv)
#define convert_sv_to_uint(sv)    (GoFloat32)SvNV(sv)
#define convert_sv_to_float64(sv) (GoFloat64)SvNV(sv)

static GoString convert_sv_to_string(sv *SV) {
    STRLEN len;
    GoString str;
    str.p = SvPV(sv, len);
    str.n = (GoInt)len;
    return str;
}
`)

	fmt.Fprintf(gen.xsbuf, "MODULE = %s    PACKAGE = %s\n\n", name, name)

	return gen
}

func getGenName(doc *ast.CommentGroup) (xsName string, goName string) {
	if doc == nil {
		return
	}

	for _, item := range doc.List {
		l := strings.Split(item.Text, " ")
		if len(l) >= 2 && l[0] == "//go2xs" {
			xsName = l[1]
		}
		if len(l) >= 2 && l[0] == "//export" {
			goName = l[1]
		}
	}
	return
}

func (gen *Generator) AddFunc(fd *ast.FuncDecl) {
	xsName, goName := getGenName(fd.Doc)
	if xsName == "" || goName == "" {
		return
	}

	fmt.Fprintf(gen.xsbuf, `void
%s (...)
    PPCODE:
{
`, xsName)

	strParams := []string{}
	if params := fd.Type.Params; params != nil {
		for i, p := range params.List {
			strParams = append(strParams, gen.addParam(i, p))
		}
	}

	if results := fd.Type.Results; results != nil {
		decls := []string{}
		rets := []string{}
		for i, r := range results.List {
			decl, ret := gen.addResult(i, r)
			decls = append(decls, decl)
			rets = append(rets, ret)
		}

		// print variable declarations
		for _, d := range decls {
			fmt.Fprintln(gen.xsbuf, d)
		}

		if len(results.List) == 1 {
			fmt.Fprintf(gen.xsbuf, "result0 = %s(%s);\n", goName, strings.Join(strParams, ", "))
		} else {
			fmt.Fprintf(gen.xsbuf, "struct %s_return result = %s(%s);\n", goName, goName, strings.Join(strParams, ", "))
			for i := range results.List {
				fmt.Fprintf(gen.xsbuf, "result%d = result.r%d;\n", i, i)
			}
		}

		// push result into Perl stack
		for _, r := range rets {
			fmt.Fprintln(gen.xsbuf, r)
		}
		fmt.Fprintf(gen.xsbuf, "XSRETURN(%d);\n", len(results.List))
	} else {
		fmt.Fprintf(gen.xsbuf, "%s(%s);\n", goName, strings.Join(strParams, ", "))
		fmt.Fprintln(gen.xsbuf, "XSRETURN(0);")
	}

	fmt.Fprint(gen.xsbuf, "}\n\n")
}

func (gen *Generator) newVariable(t string, name string) string {
	v := &Variable{Type: t, Name: name}
	gen.variables = append(gen.variables, v)
	return name
}

func (gen *Generator) ST(n int) string {
	return fmt.Sprintf("ST(%d)", n)
}

func (gen *Generator) SV2GoType(expr string, t ast.Expr) string {
	if ident, ok := t.(*ast.Ident); ok {
		return fmt.Sprintf("convert_sv_to_%s(%s)", ident.Name, expr)
	}
	if array, ok := t.(*ast.ArrayType); ok {
		if array.Len == nil {
			fmt.Println(array.Elt)
		}
	}
	return expr
}

func (gen *Generator) getGoType(t ast.Expr) string {
	if ident, ok := t.(*ast.Ident); ok {
		switch ident.Name {
		case "int8":
			return "GoInt8"
		case "uint8":
			return "GoUint8"
		case "int16":
			return "GoInt16"
		case "uint16":
			return "GoUint16"
		case "int32":
			return "GoInt32"
		case "uint32":
			return "GoUint32"
		case "int64":
			return "GoInt64"
		case "uint64":
			return "GoUint64"
		case "int":
			return "GoInt"
		case "uint":
			return "GoUint"
		case "float32":
			return "GoFloat32"
		case "float64":
			return "GoFloat64"
		case "string":
			return "GoString"
		}
	}
	return ""
}

func (gen *Generator) addParam(index int, param *ast.Field) string {
	v := gen.newVariable(gen.getGoType(param.Type), fmt.Sprintf("param%d", index))
	fmt.Fprintf(gen.xsbuf, "%s = %s;\n", v, gen.SV2GoType(gen.ST(index), param.Type))
	return v
}

func (gen *Generator) addResult(index int, result *ast.Field) (decl string, ret string) {
	fmt.Println(result.Names, result.Type)
	if ident, ok := result.Type.(*ast.Ident); ok {
		switch ident.Name {
		case "int":
			decl = fmt.Sprintf("GoInt result%d;", index)
			ret = fmt.Sprintf("XPUSHs(sv_2mortal(newSViv(result%d)));", index)
		case "uint":
			decl = fmt.Sprintf("GoUint result%d;", index)
			ret = fmt.Sprintf("XPUSHs(sv_2mortal(newSVuv(result%d)));", index)
		case "float32":
			decl = fmt.Sprintf("GoFloat32 result%d;", index)
			ret = fmt.Sprintf("XPUSHs(sv_2mortal(newSVnv(result%d)));", index)
		case "float64":
			decl = fmt.Sprintf("GoFloat64 result%d;", index)
			ret = fmt.Sprintf("XPUSHs(sv_2mortal(newSVnv(result%d)));", index)
		case "string":
			decl = fmt.Sprintf("GoString result%d;", index)
			ret = fmt.Sprintf("XPUSHs(sv_2mortal(newSVpvn(result%d.p, result%d.n)));", index, index)
		}
	}
	return
}

func (gen *Generator) Output() {
	os.MkdirAll(gen.name, 0755)
	os.MkdirAll(path.Join(gen.name, "lib"), 0755)
	ioutil.WriteFile(path.Join(gen.name, gen.name+".xs"), gen.xsbuf.Bytes(), 0644)
	ioutil.WriteFile(path.Join(gen.name, "ppport.h"), []byte(ppport), 0644)
	ioutil.WriteFile(path.Join(gen.name, "Makefile.PL"), []byte(`use 5.010000;
use ExtUtils::MakeMaker;

system(qw(go build -buildmode=c-shared -o lib`+gen.name+`.dylib test.go)) and die;

# See lib/ExtUtils/MakeMaker.pm for details of how to influence
# the contents of the Makefile that is written.
WriteMakefile(
    NAME              => '`+gen.name+`',
    VERSION_FROM      => 'lib/`+gen.name+`.pm', # finds $VERSION
    PREREQ_PM         => {}, # e.g., Module::Name => 1.1
    ($] >= 5.005 ?     ## Add these new keywords supported since 5.005
      (ABSTRACT_FROM  => 'lib/`+gen.name+`.pm', # retrieve abstract from module
       AUTHOR         => 'Ichinose Shogo <shogo@local>') : ()),
    LIBS              => ['-L. -l`+gen.name+`'], # e.g., '-lm'
    DEFINE            => '', # e.g., '-DHAVE_SOMETHING'
    INC               => '-I.', # e.g., '-I. -I/usr/include/other'
	# Un-comment this if you add C files to link with later:
    # OBJECT            => '$(O_FILES)', # link all the C files too
);
`), 0644)
	ioutil.WriteFile(path.Join(gen.name, "lib", gen.name+".pm"), []byte(`package SomeModule;

use 5.010000;
use strict;
use warnings;

our $VERSION = '0.01';

require XSLoader;
XSLoader::load('`+gen.name+`', $VERSION);

1;
__END__
# Below is stub documentation for your module. You'd better edit it!

=head1 NAME

SomeModule - Perl extension for blah blah blah

=head1 SYNOPSIS

  use `+gen.name+`;
  blah blah blah

=head1 DESCRIPTION

Blah blah blah.

=head2 EXPORT

None by default.



=head1 SEE ALSO


=head1 AUTHOR

Ichinose Shogo, E<lt>shogo@localE<gt>

=head1 COPYRIGHT AND LICENSE

=cut
`), 0644)
}

func main() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "hoge/test.go", nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	gen := NewGenerator("hoge")

	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			gen.AddFunc(fd)
		}
	}

	gen.Output()
}
