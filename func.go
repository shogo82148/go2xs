package go2xs

import (
	"bytes"
	"fmt"
	"go/ast"
	"strings"
)

type FuncGenerator struct {
	xsName string
	fd     *ast.FuncDecl

	xsBefore *bytes.Buffer
	xsAfter  *bytes.Buffer
	goBefore *bytes.Buffer
	goAfter  *bytes.Buffer

	// parameters declaration for Go Glue code
	goGlueParamDecls []string

	// parameters for calling Go func
	goParams []string

	// parameter for calling glue code
	xsParams []string

	numXsReturn int
}

func getXSName(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	for _, item := range doc.List {
		l := strings.Split(item.Text, " ")
		if len(l) >= 2 && l[0] == "//go2xs" {
			return l[1]
		}
	}
	return ""
}

func NewFuncGenerator(fd *ast.FuncDecl) *FuncGenerator {
	xsName := getXSName(fd.Doc)
	if xsName == "" {
		return nil
	}

	return &FuncGenerator{
		xsName:           xsName,
		fd:               fd,
		xsBefore:         &bytes.Buffer{},
		xsAfter:          &bytes.Buffer{},
		goBefore:         &bytes.Buffer{},
		goAfter:          &bytes.Buffer{},
		goGlueParamDecls: []string{},
		goParams:         []string{},
		xsParams:         []string{},
		numXsReturn:      0,
	}
}

func (fg *FuncGenerator) Generate() {
	fmt.Fprintf(fg.xsBefore, `void
%s (...)
    PPCODE:
{
`, fg.xsName)

	if params := fg.fd.Type.Params; params != nil {
		for i, p := range params.List {
			fg.addParam(i, p)
		}
	}

	fmt.Fprintf(fg.xsAfter, "XSRETURN(%d);\n", fg.numXsReturn)
	fmt.Fprint(fg.xsAfter, "}\n\n")
}

// Glue code written in XS
func (fg *FuncGenerator) XSCode() string {
	return fg.xsBefore.String() + fg.xsCall() + fg.xsAfter.String()
}

// Glue code written in Go
func (fg *FuncGenerator) GoCode() string {
	return fg.goGlueDecl() + "{\n" + fg.goBefore.String() + fg.goCall() + fg.goAfter.String() + "}\n"
}

// Declaration for Go glue code
func (fg *FuncGenerator) goGlueDecl() string {
	return "//export go2xs" + fg.xsName + "\n" +
		"func go2xs" + fg.xsName + "(" + strings.Join(fg.goGlueParamDecls, ", ") + ")"
}

// Go code for calling original Go function
func (fg *FuncGenerator) goCall() string {
	return fg.fd.Name.Name + "(" + strings.Join(fg.goParams, ", ") + ")\n"
}

func (fg *FuncGenerator) xsCall() string {
	return "go2xs" + fg.xsName + "(" + strings.Join(fg.xsParams, ", ") + ");\n"
}

func (fg *FuncGenerator) addParam(index int, param *ast.Field) {
	if ident, ok := param.Type.(*ast.Ident); ok {
		switch ident.Name {
		case "int8":
			fg.addParamPrimitive(index, "int8", "GoInt8", "SvIV")
		case "uint8":
			fg.addParamPrimitive(index, "uint8", "GoUint8", "SvUV")
		case "int16":
			fg.addParamPrimitive(index, "int16", "GoInt16", "SvIV")
		case "uint16":
			fg.addParamPrimitive(index, "uint16", "GoUint16", "SvUV")
		case "int32":
			fg.addParamPrimitive(index, "int32", "GoInt32", "SvIV")
		case "uint32":
			fg.addParamPrimitive(index, "uint32", "GoUint32", "SvUV")
		case "int64":
			fg.addParamPrimitive(index, "int64", "GoInt64", "SvIV")
		case "uint64":
			fg.addParamPrimitive(index, "uint64", "GoUint64", "SvUV")
		case "int":
			fg.addParamPrimitive(index, "int", "GoInt", "SvIV")
		case "uint":
			fg.addParamPrimitive(index, "uint", "GoUint", "SvUV")
		case "float32":
			fg.addParamPrimitive(index, "float32", "GoFloat32", "SvNV")
		case "float64":
			fg.addParamPrimitive(index, "float64", "GoFloat64", "SvNV")
		case "string":
			return
		}
	}
}

// addParamPrimitive converts XS primitive types
func (fg *FuncGenerator) addParamPrimitive(index int, goType, xsType, svType string) {
	fg.goGlueParamDecls = append(fg.goGlueParamDecls, fmt.Sprintf("param%d %s", index, goType))
	fg.goParams = append(fg.goParams, fmt.Sprintf("param%d", index))
	fg.xsParams = append(fg.xsParams, fmt.Sprintf("param%d", index))
	fmt.Fprintf(fg.xsBefore, "%s param%d = (%s)%s(ST(%d))\n", xsType, index, xsType, svType, index)
}
