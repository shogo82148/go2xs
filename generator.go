package go2xs

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
)

type Generator struct {
	funcGenerators []*FuncGenerator
}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) ParseFile(path string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			fg := NewFuncGenerator(fd)
			if fg != nil {
				g.funcGenerators = append(g.funcGenerators, fg)
			}
		}
	}
}

func (g *Generator) Generate() {
	for _, fg := range g.funcGenerators {
		fg.Generate()
	}
}

func (g *Generator) Output(name string) {
	os.MkdirAll("lib", 0755)
	ioutil.WriteFile("ppport.h", []byte(ppport), 0644)
	ioutil.WriteFile("Makefile.PL", []byte(`use 5.010000;
use ExtUtils::MakeMaker;

my $ext;
$ext = "dylib" if $^O eq 'darwin';
$ext = "so" if $^O eq 'linux';
system("go build -buildmode=c-shared -o lib`+name+`.$ext *.go") and die;

# See lib/ExtUtils/MakeMaker.pm for details of how to influence
# the contents of the Makefile that is written.
WriteMakefile(
    NAME              => '`+name+`',
    VERSION_FROM      => 'lib/`+name+`.pm', # finds $VERSION
    PREREQ_PM         => {}, # e.g., Module::Name => 1.1
    ($] >= 5.005 ?     ## Add these new keywords supported since 5.005
      (ABSTRACT_FROM  => 'lib/`+name+`.pm', # retrieve abstract from module
       AUTHOR         => 'Ichinose Shogo <shogo@local>') : ()),
    LIBS              => ['-L. -l`+name+`'], # e.g., '-lm'
    DEFINE            => '', # e.g., '-DHAVE_SOMETHING'
    INC               => '-I.', # e.g., '-I. -I/usr/include/other'
# Un-comment this if you add C files to link with later:
    # OBJECT            => '$(O_FILES)', # link all the C files too
);
`), 0644)
	ioutil.WriteFile(path.Join("lib", name+".pm"), []byte(`package SomeModule;
use 5.010000;
use strict;
use warnings;
our $VERSION = '0.01';
require XSLoader;
XSLoader::load('`+name+`', $VERSION);
1;
__END__
# Below is stub documentation for your module. You'd better edit it!
=head1 NAME
SomeModule - Perl extension for blah blah blah
=head1 SYNOPSIS
  use `+name+`;
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

	xsFile, _ := os.OpenFile(name+".xs", os.O_WRONLY|os.O_CREATE, 0644)
	defer xsFile.Close()
	goFile, _ := os.OpenFile("go2xs.go", os.O_WRONLY|os.O_CREATE, 0644)
	defer goFile.Close()

	fmt.Fprintln(xsFile, `#define PERL_NO_GET_CONTEXT
#include "EXTERN.h"
#include "perl.h"
#include "XSUB.h"
#include "ppport.h"`)
	fmt.Fprintf(xsFile, "#include \"lib%s.h\"\n", name)
	fmt.Fprintf(xsFile, "MODULE = %s    PACKAGE = %s\n\n", name, name)

	fmt.Fprint(goFile, `package main

import "C"

import "unsafe"

var _ unsafe.Pointer

func main() {}
`)

	for _, fg := range g.funcGenerators {
		fmt.Fprintln(xsFile, fg.XSCode())
		fmt.Fprintln(goFile, fg.GoCode())
	}
}
