package value

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aeremic/cgo/ast"
)

type Type string
type BuiltInFunction func(args ...Wrapper) Wrapper

const (
	INTEGER  = "INTEGER"
	STRING   = "STRING"
	BOOLEAN  = "BOOLEAN"
	NULL     = "NULL"
	RETURN   = "RETURN"
	ERROR    = "ERROR"
	FUNCTION = "FUNCTION"
	BUILTIN  = "BUILTIN"
	ARRAY    = "ARRAY"
)

type Wrapper interface {
	Type() Type
	Sprintf() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() Type {
	return INTEGER
}

func (i *Integer) Sprintf() string {
	return fmt.Sprintf("%d", i.Value)
}

type String struct {
	Value string
}

func (s *String) Type() Type {
	return STRING
}

func (s *String) Sprintf() string {
	return s.Value
}

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() Type {
	return BOOLEAN
}

func (b *Boolean) Sprintf() string {
	return fmt.Sprintf("%t", b.Value)
}

type Null struct{}

func (n *Null) Type() Type {
	return NULL
}

func (n *Null) Sprintf() string {
	return "null"
}

type ReturnValue struct {
	Value Wrapper
}

func (rv *ReturnValue) Type() Type {
	return RETURN
}

func (rv *ReturnValue) Sprintf() string {
	return rv.Value.Sprintf()
}

type Error struct {
	Message string
}

func (e *Error) Type() Type {
	return ERROR
}

func (e *Error) Sprintf() string {
	return "ERROR: " + e.Message
}

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() Type {
	return FUNCTION
}

func (f *Function) Sprintf() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("fn")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")

	return out.String()
}

type BuiltIn struct {
	Fn BuiltInFunction
}

func (bi *BuiltIn) Type() Type {
	return BUILTIN
}

func (bi *BuiltIn) Sprintf() string {
	return "builtin function"
}

type Array struct {
	Elements []Wrapper
}

func (a *Array) Type() Type {
	return ARRAY
}

func (a *Array) Sprintf() string {
	var out bytes.Buffer

	elements := []string{}
	for _, element := range a.Elements {
		elements = append(elements, element.Sprintf())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}
