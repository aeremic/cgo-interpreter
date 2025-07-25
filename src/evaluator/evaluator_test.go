package evaluator

import (
	"github.com/aeremic/cgo/parser"
	"github.com/aeremic/cgo/tokenizer"
	"github.com/aeremic/cgo/value"

	"testing"
)

func testEval(input string) value.Wrapper {
	t := tokenizer.New(input)
	p := parser.New(t)
	program := p.ParseProgram()

	env := value.NewEnvironment()

	return Eval(program, env)
}

func testIntegerValueWrapper(t *testing.T, v value.Wrapper, expected int64) bool {
	result, ok := v.(*value.Integer)
	if !ok {
		t.Errorf("v is not Integer type. Got %T", v)

		return false
	}

	if result.Value != expected {
		t.Errorf("v has wrong value. Got %d instead of %d",
			result.Value, expected)

		return false
	}

	return true
}

func testNullValueWrapper(t *testing.T, v value.Wrapper) bool {
	if v != NULL {
		t.Errorf("value is not NULL. Got %T (%+v)", v, v)
		return false
	}

	return true
}

func testBooleanValueWrapper(t *testing.T, v value.Wrapper, expected bool) bool {
	result, ok := v.(*value.Boolean)
	if !ok {
		t.Errorf("v is not Boolean type. Got %T", v)

		return false
	}

	if result.Value != expected {
		t.Errorf("v has wrong value. Got %t instead of %t",
			result.Value, expected)

		return false
	}

	return true
}

func TestEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 + -50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"3 * (3 * 3) + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		testIntegerValueWrapper(t, evaluated, test.expected)
	}
}

func TestEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"false != true", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		testBooleanValueWrapper(t, evaluated, test.expected)
	}
}

func TestBangOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"!true", false},
		{"!false", true},
		{"!5", false},
		{"!!true", true},
		{"!!false", false},
		{"!!5", true},
		{"!!!5", false},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		testBooleanValueWrapper(t, evaluated, test.expected)
	}
}

func TestIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"if (true) { 10 }", 10},
		{"if (false) { 10 }", nil},
		{"if (1) { 10 }", 10},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 > 2) { 10 }", nil},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		{"if (1 < 2) { 10 } else { 20 }", 10},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		integer, ok := test.expected.(int)
		if ok {
			testIntegerValueWrapper(t, evaluated, int64(integer))
		} else {
			testNullValueWrapper(t, evaluated)
		}
	}
}

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"return 10;", 10},
		{"return 10; 9;", 10},
		{"return 2 * 5; 9;", 10},
		{"9; return 2 * 5; 9;", 10},
		{
			`if (true) {
				if (true) {
					if (true) {
						return 20;
					}

					return 10;
				}

				return 1;
			}
			`, 20,
		},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		testIntegerValueWrapper(t, evaluated, test.expected)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
	}{

		{
			"5 + true;",
			"type mismatch: INTEGER + BOOLEAN",
		},
		{
			"5 + true; 5;",
			"type mismatch: INTEGER + BOOLEAN",
		},
		{
			"-true",
			"unknown operator: -BOOLEAN",
		},
		{
			"true + false;",
			"unknown operator: BOOLEAN + BOOLEAN",
		},
		{
			"5; true + false; 5",
			"unknown operator: BOOLEAN + BOOLEAN",
		},
		{
			"if (10 > 1) { true + false; }",
			"unknown operator: BOOLEAN + BOOLEAN",
		},
		{
			`if (10 > 1) {
				if (10 > 1) {
					return true + false;
				}
				return 1;
				}
			`, "unknown operator: BOOLEAN + BOOLEAN",
		},
		{
			"foobar",
			"identifier not found: foobar",
		},
		{
			`"hello" - "world"`,
			"unknown operator: STRING - STRING",
		},
		{
			`{"name": "Monkey"}[fn(x) { x }];`,
			"unusable as hash key: FUNCTION",
		},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		errorWrapped, ok := evaluated.(*value.Error)
		if !ok {
			t.Errorf("No error returned. Got %T(%+v)", evaluated, evaluated)
			continue
		}

		if errorWrapped.Message != test.expectedMessage {
			t.Errorf("Invalid message. Got %s instead of %s",
				errorWrapped.Message, test.expectedMessage)
		}
	}
}

func TestLetStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let a = 5; a;", 5},
		{"let a = 5 * 5; a;", 25},
		{"let a = 5; let b = a; b;", 5},
		{"let a = 5; let b = a; let c = a + b + 5; c;", 15},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		testIntegerValueWrapper(t, evaluated, test.expected)
	}
}

func TestFunction(t *testing.T) {
	input := "fn(x) { x + 2; };"

	evaluated := testEval(input)
	fn, ok := evaluated.(*value.Function)
	if !ok {
		t.Fatalf("value is not function type. Got %T (%+v)", evaluated, evaluated)
	}

	if len(fn.Parameters) != 1 {
		t.Fatalf("Invalid number of params in function. Got %d instead of %d. Params: %+v ",
			len(fn.Parameters), 1, fn.Parameters)
	}

	if fn.Parameters[0].String() != "x" {
		t.Fatalf("Invalid function param literal. Got %s instead of %x", fn.Parameters[0], "x")
	}

	expectedBody := "(x + 2)"
	if fn.Body.String() != expectedBody {
		t.Fatalf("Invalid function body. Got %s instead of %s",
			fn.Body.String(), expectedBody)
	}
}

func TestFunctionApplication(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let identity = fn(x) { x; }; identity(5);", 5},
		{"let identity = fn(x) { return x; }; identity(5);", 5},
		{"let double = fn(x) { x * 2; }; double(5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5, 5);", 10},
		{"let add = fn(x, y) { x + y; }; add(5 + 5, add(5, 5));", 20},
		{"fn(x) { x; }(5)", 5},
	}

	for _, test := range tests {
		val := testEval(test.input)
		testIntegerValueWrapper(t, val, test.expected)
	}
}

func TestClosures(t *testing.T) {
	input := `
let addNumbers = fn(x) {
	fn(y) { x + y };
};
let addTwo = addNumbers(2);
addTwo(2);`

	evaluated := testEval(input)
	testIntegerValueWrapper(t, evaluated, 4)
}

func TestStringLiteral(t *testing.T) {
	input := `"hello world";`

	evaluated := testEval(input)
	str, ok := evaluated.(*value.String)
	if !ok {
		t.Fatalf("value is not String type. Got %T(%+v)",
			evaluated, evaluated)
	}

	if str.Value != "hello world" {
		t.Fatalf("str.Value has wrong value. Got %s instead of %s",
			str.Value, "hello world")
	}
}

func TestStringConcatenation(t *testing.T) {
	input := `"hello" + " " + "world";`

	evaluated := testEval(input)
	str, ok := evaluated.(*value.String)
	if !ok {
		t.Fatalf("value is not String type. Got %T (%+v)",
			evaluated, evaluated)
	}

	if str.Value != "hello world" {
		t.Fatalf("String wrong value. Got %s instead of %s",
			str.Value, "hello world")
	}
}

func TestBuiltInFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{`len("")`, 0},
		{`len("four")`, 4},
		{`len("hello world")`, 11},
		{`len(1)`, "argument to `len` not supported, got INTEGER"},
		{`len("one", "two")`, "wrong number of arguments. got=2, want=1"},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)

		switch expected := test.expected.(type) {
		case int:
			testIntegerValueWrapper(t, evaluated, int64(expected))
		case string:
			e, ok := evaluated.(*value.Error)

			if !ok {
				t.Errorf("object is not Error type. Got %T (%+v)",
					evaluated, evaluated)
			}

			if e.Message != expected {
				t.Errorf("wrong error message. Got %q instead of %q",
					e.Message, expected)
			}
		}
	}
}

func TestArrayLiterals(t *testing.T) {
	input := "[1, 2 * 2, 3 + 3]"

	evaluated := testEval(input)
	result, ok := evaluated.(*value.Array)
	if !ok {
		t.Fatalf("invalid type. got %T (%+v) instead of %s",
			evaluated, evaluated, "Array")
	}

	if len(result.Elements) != 3 {
		t.Fatalf("invalid len of elements in the array. got %d instead of %d",
			len(result.Elements), 3)
	}

	testIntegerValueWrapper(t, result.Elements[0], 1)
	testIntegerValueWrapper(t, result.Elements[1], 4)
	testIntegerValueWrapper(t, result.Elements[2], 6)
}

func TestArrayIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"[1, 2, 3][0]", 1},
		{"[1, 2, 3][1]", 2},
		{"[1, 2, 3][2]", 3},
		{"let i = 0; [1][i];", 1},
		{"[1, 2, 3][1 + 1];", 3},
		{"let myArray = [1, 2, 3]; myArray[2];", 3},
		{"let myArray = [1, 2, 3]; myArray[0] + myArray[1] + myArray[2];", 6},
		{"let myArray = [1, 2, 3]; let i = myArray[0]; myArray[i]", 2},
		{"[1, 2, 3][3]", nil},
		{"[1, 2, 3][-1]", nil},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		integer, ok := test.expected.(int)
		if ok {
			testIntegerValueWrapper(t, evaluated, int64(integer))
		} else {
			testNullValueWrapper(t, evaluated)
		}
	}
}

func TestStringDictKey(t *testing.T) {
	hello1 := &value.String{Value: "hello"}
	hello2 := &value.String{Value: "hello"}
	world1 := &value.String{Value: "world"}
	world2 := &value.String{Value: "world"}

	if hello1.HashKey() != hello2.HashKey() {
		t.Errorf("strings with same content have different hash keys %s[%d] %s[%d]",
			hello1.Value, hello1.HashKey().Value, hello2.Value, hello2.HashKey().Value)
	}

	if world1.HashKey() != world2.HashKey() {
		t.Errorf("strings with same content have different hash keys %s[%d] %s[%d]",
			world1.Value, world1.HashKey().Value, world2.Value, world2.HashKey().Value)
	}

	if hello1.HashKey() == world1.HashKey() {
		t.Errorf("strings with different content have same hash keys %d %d",
			hello1.HashKey().Value, world1.HashKey().Value)
	}
}

func TestDictLiterals(t *testing.T) {
	input := `let two = "two";
	{
		"one": 10 - 9,
		two: 1 + 1,
		"thr" + "ee": 6 / 2,
		4: 4,
		true: 5,
		false: 6
	}`

	evaluated := testEval(input)
	result, ok := evaluated.(*value.Dict)
	if !ok {
		t.Fatalf("invalid type. expected Dict got %T (%+v)", evaluated, evaluated)
	}

	expected := map[value.HashKey]int64{
		(&value.String{Value: "one"}).HashKey():   1,
		(&value.String{Value: "two"}).HashKey():   2,
		(&value.String{Value: "three"}).HashKey(): 3,
		(&value.Integer{Value: 4}).HashKey():      4,
		TRUE.HashKey():                            5,
		FALSE.HashKey():                           6,
	}

	if len(result.Elements) != len(expected) {
		t.Fatalf("invalid number of elements. got %d instead of %d",
			len(result.Elements), len(expected))
	}

	for expectedKey, expectedValue := range expected {
		actualValue, ok := result.Elements[expectedKey]
		if !ok {
			t.Errorf("no element for expected key %d", expectedKey.Value)
		}

		testIntegerValueWrapper(t, actualValue.Value, expectedValue)
	}
}

func TestDictIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{
			`{"foo": 5}["foo"]`,
			5,
		},
		{
			`{"foo": 5}["bar"]`,
			nil,
		},
		{
			`let key = "foo"; {"foo": 5}[key]`,
			5,
		},
		{
			`{}["foo"]`,
			nil,
		},
		{
			`{5: 5}[5]`,
			5,
		},
		{
			`{true: 5}[true]`,
			5,
		},
		{
			`{false: 5}[false]`,
			5,
		},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		expectedInteger, ok := test.expected.(int)
		if ok {
			testIntegerValueWrapper(t, evaluated, int64(expectedInteger))
		} else {
			testNullValueWrapper(t, evaluated)
		}
	}
}
