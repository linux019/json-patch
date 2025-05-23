package jsonpatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func mergePatch(t *testing.T, doc, patch string) string {
	t.Helper()
	out, err := MergePatch([]byte(doc), []byte(patch))

	if err != nil {
		t.Errorf(fmt.Sprintf("%s: %s", err, patch))
	}

	return string(out)
}

func TestMergePatchReplaceKey(t *testing.T) {
	doc := `{ "title": "hello" }`
	pat := `{ "title": "goodbye" }`

	res := mergePatch(t, doc, pat)

	if !compareJSON(pat, res) {
		t.Fatalf("Key was not replaced")
	}
}

func TestMergePatchIgnoresOtherValues(t *testing.T) {
	doc := `{ "title": "hello", "age": 18 }`
	pat := `{ "title": "goodbye" }`

	res := mergePatch(t, doc, pat)

	exp := `{ "title": "goodbye", "age": 18 }`

	if !compareJSON(exp, res) {
		t.Fatalf("Key was not replaced")
	}
}

func TestMergePatchNilDoc(t *testing.T) {
	doc := `{ "title": null }`
	pat := `{ "title": {"foo": "bar"} }`

	res := mergePatch(t, doc, pat)

	exp := `{ "title": {"foo": "bar"} }`

	if !compareJSON(exp, res) {
		t.Fatalf("Key was not replaced")
	}
}

type arrayCases struct {
	original, patch, res string
}

func TestMergePatchNilArray(t *testing.T) {

	cases := []arrayCases{
		{`{"a": [ {"b":"c"} ] }`, `{"a": [1]}`, `{"a": [1]}`},
		{`{"a": [ {"b":"c"} ] }`, `{"a": [null, 1]}`, `{"a": [null, 1]}`},
		{`["a",null]`, `[null]`, `[null]`},
		{`["a"]`, `[null]`, `[null]`},
		{`["a", "b"]`, `["a", null]`, `["a", null]`},
		{`{"a":["b"]}`, `{"a": ["b", null]}`, `{"a":["b", null]}`},
		{`{"a":[]}`, `{"a": ["b", null, null, "a"]}`, `{"a":["b", null, null, "a"]}`},
	}

	for _, c := range cases {
		t.Log(c.original)
		act := mergePatch(t, c.original, c.patch)

		if !compareJSON(c.res, act) {
			t.Errorf("null values not preserved in array")
		}
	}
}

func TestMergePatchRecursesIntoObjects(t *testing.T) {
	doc := `{ "person": { "title": "hello", "age": 18 } }`
	pat := `{ "person": { "title": "goodbye" } }`

	res := mergePatch(t, doc, pat)

	exp := `{ "person": { "title": "goodbye", "age": 18 } }`

	if !compareJSON(exp, res) {
		t.Fatalf("Key was not replaced: %s", res)
	}
}

type nonObjectCases struct {
	doc, pat, res string
}

func TestMergePatchReplacesNonObjectsWholesale(t *testing.T) {
	a1 := `[1]`
	a2 := `[2]`
	o1 := `{ "a": 1 }`
	o2 := `{ "a": 2 }`
	o3 := `{ "a": 1, "b": 1 }`
	o4 := `{ "a": 2, "b": 1 }`

	cases := []nonObjectCases{
		{a1, a2, a2},
		{o1, a2, a2},
		{a1, o1, o1},
		{o3, o2, o4},
	}

	for _, c := range cases {
		act := mergePatch(t, c.doc, c.pat)

		if !compareJSON(c.res, act) {
			t.Errorf("whole object replacement failed")
		}
	}
}

func TestMergePatchReturnsErrorOnBadJSON(t *testing.T) {
	_, err := MergePatch([]byte(`[[[[`), []byte(`1`))

	if err == nil {
		t.Errorf("Did not return an error for bad json: %s", err)
	}

	_, err = MergePatch([]byte(`1`), []byte(`[[[[`))

	if err == nil {
		t.Errorf("Did not return an error for bad json: %s", err)
	}
}

func TestMergePatchReturnsEmptyArrayOnEmptyArray(t *testing.T) {
	doc := `{ "array": ["one", "two"] }`
	pat := `{ "array": [] }`

	exp := `{ "array": [] }`

	res, err := MergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Emtpy array did not return not return as empty array")
	}
}

var rfcTests = []struct {
	target   string
	patch    string
	expected string
}{
	// test cases from https://tools.ietf.org/html/rfc7386#appendix-A
	{target: `{"a":"b"}`, patch: `{"a":"c"}`, expected: `{"a":"c"}`},
	{target: `{"a":"b"}`, patch: `{"b":"c"}`, expected: `{"a":"b","b":"c"}`},
	{target: `{"a":"b"}`, patch: `{"a":null}`, expected: `{}`},
	{target: `{"a":"b","b":"c"}`, patch: `{"a":null}`, expected: `{"b":"c"}`},
	{target: `{"a":["b"]}`, patch: `{"a":"c"}`, expected: `{"a":"c"}`},
	{target: `{"a":"c"}`, patch: `{"a":["b"]}`, expected: `{"a":["b"]}`},
	{target: `{"a":{"b": "c"}}`, patch: `{"a": {"b": "d","c": null}}`, expected: `{"a":{"b":"d"}}`},
	{target: `{"a":[{"b":"c"}]}`, patch: `{"a":[1]}`, expected: `{"a":[1]}`},
	{target: `["a","b"]`, patch: `["c","d"]`, expected: `["c","d"]`},
	{target: `{"a":"b"}`, patch: `["c"]`, expected: `["c"]`},
	{target: `{"a":"foo"}`, patch: `null`, expected: `null`},
	{target: `{"a":"foo"}`, patch: `"bar"`, expected: `"bar"`},
	{target: `{"e":null}`, patch: `{"a":1}`, expected: `{"a":1,"e":null}`},
	{target: `[1,2]`, patch: `{"a":"b","c":null}`, expected: `{"a":"b"}`},
	{target: `{}`, patch: `{"a":{"bb":{"ccc":null}}}`, expected: `{"a":{"bb":{}}}`},
}

func TestMergePatchRFCCases(t *testing.T) {
	for i, c := range rfcTests {
		out := mergePatch(t, c.target, c.patch)

		if !compareJSON(out, c.expected) {
			t.Errorf("case[%d], patch '%s' did not apply properly to '%s'. expected:\n'%s'\ngot:\n'%s'", i, c.patch, c.target, c.expected, out)
		}
	}
}

func TestResembleJSONArray(t *testing.T) {
	testCases := []struct {
		input    []byte
		expected bool
	}{
		// Failure cases
		{input: []byte(``), expected: false},
		{input: []byte(`not an array`), expected: false},
		{input: []byte(`{"foo": "bar"}`), expected: false},
		{input: []byte(`{"fizz": ["buzz"]}`), expected: false},
		{input: []byte(`[bad suffix`), expected: false},
		{input: []byte(`bad prefix]`), expected: false},
		{input: []byte(`][`), expected: false},

		// Valid cases
		{input: []byte(`[]`), expected: true},
		{input: []byte(`["foo", "bar"]`), expected: true},
		{input: []byte(`[["foo", "bar"]]`), expected: true},
		{input: []byte(`[not valid syntax]`), expected: true},

		// Valid cases with whitespace
		{input: []byte(`      []`), expected: true},
		{input: []byte(`[]      `), expected: true},
		{input: []byte(`      []      `), expected: true},
		{input: []byte(`      [        ]      `), expected: true},
		{input: []byte("\t[]"), expected: true},
		{input: []byte("[]\n"), expected: true},
		{input: []byte("\n\t\r[]"), expected: true},
	}

	for _, test := range testCases {
		result := resemblesJSONArray(test.input)
		if result != test.expected {
			t.Errorf(
				`expected "%t" but received "%t" for case: "%s"`,
				test.expected,
				result,
				string(test.input),
			)
		}
	}
}

func TestCreateMergePatchReplaceKey(t *testing.T) {
	doc := `{ "title": "hello", "nested": {"one": 1, "two": 2} }`
	pat := `{ "title": "goodbye", "nested": {"one": 2, "two": 2}  }`

	exp := `{ "title": "goodbye", "nested": {"one": 2}  }`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Key was not replaced")
	}
}

func TestCreateMergePatchGetArray(t *testing.T) {
	doc := `{ "title": "hello", "array": ["one", "two"], "notmatch": [1, 2, 3] }`
	pat := `{ "title": "hello", "array": ["one", "two", "three"], "notmatch": [1, 2, 3]  }`

	exp := `{ "array": ["one", "two", "three"] }`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Array was not added")
	}
}

func TestCreateMergePatchGetObjArray(t *testing.T) {
	doc := `{ "title": "hello", "array": [{"banana": true}, {"evil": false}], "notmatch": [{"one":1}, {"two":2}, {"three":3}] }`
	pat := `{ "title": "hello", "array": [{"banana": false}, {"evil": true}], "notmatch": [{"one":1}, {"two":2}, {"three":3}] }`

	exp := `{  "array": [{"banana": false}, {"evil": true}] }`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Object array was not added")
	}
}

func TestCreateMergePatchDeleteKey(t *testing.T) {
	doc := `{ "title": "hello", "nested": {"one": 1, "two": 2} }`
	pat := `{ "title": "hello", "nested": {"one": 1}  }`

	exp := `{"nested":{"two":null}}`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	// We cannot use "compareJSON", since Equals does not report a difference if the value is null
	if exp != string(res) {
		t.Fatalf("Key was not removed")
	}
}

func TestCreateMergePatchEmptyArray(t *testing.T) {
	doc := `{ "array": null }`
	pat := `{ "array": [] }`

	exp := `{"array":[]}`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	// We cannot use "compareJSON", since Equals does not report a difference if the value is null
	if exp != string(res) {
		t.Fatalf("Key was not removed")
	}
}

func TestCreateMergePatchNil(t *testing.T) {
	doc := `{ "title": "hello", "nested": {"one": 1, "two": [{"one":null}, {"two":null}, {"three":null}]} }`
	pat := doc

	exp := `{}`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Object array was not added")
	}
}

func TestCreateMergePatchObjArray(t *testing.T) {
	doc := `{ "array": [ {"a": {"b": 2}}, {"a": {"b": 3}} ]}`
	exp := `{}`

	res, err := CreateMergePatch([]byte(doc), []byte(doc))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	// We cannot use "compareJSON", since Equals does not report a difference if the value is null
	if exp != string(res) {
		t.Fatalf("Array was not empty, was " + string(res))
	}
}

func TestCreateMergePatchSameOuterArray(t *testing.T) {
	doc := `[{"foo": "bar"}]`
	pat := doc
	exp := `[{}]`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Outer array was not unmodified")
	}
}

func TestCreateMergePatchModifiedOuterArray(t *testing.T) {
	doc := `[{"name": "John"}, {"name": "Will"}]`
	pat := `[{"name": "Jane"}, {"name": "Will"}]`
	exp := `[{"name": "Jane"}, {}]`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Expected %s but received %s", exp, res)
	}
}

func TestCreateMergePatchMismatchedOuterArray(t *testing.T) {
	doc := `[{"name": "John"}, {"name": "Will"}]`
	pat := `[{"name": "Jane"}]`

	_, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err == nil {
		t.Errorf("Expected error due to array length differences but received none")
	}
}

func TestCreateMergePatchMismatchedOuterTypes(t *testing.T) {
	doc := `[{"name": "John"}]`
	pat := `{"name": "Jane"}`

	_, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err == nil {
		t.Errorf("Expected error due to mismatched types but received none")
	}
}

func TestCreateMergePatchNoDifferences(t *testing.T) {
	doc := `{ "title": "hello", "nested": {"one": 1, "two": 2} }`
	pat := doc

	exp := `{}`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Fatalf("Key was not replaced")
	}
}

func TestCreateMergePatchComplexMatch(t *testing.T) {
	doc := `{"hello": "world","t": true ,"f": false, "n": null,"i": 123,"pi": 3.1416,"a": [1, 2, 3, 4], "nested": {"hello": "world","t": true ,"f": false, "n": null,"i": 123,"pi": 3.1416,"a": [1, 2, 3, 4]} }`
	empty := `{}`
	res, err := CreateMergePatch([]byte(doc), []byte(doc))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	// We cannot use "compareJSON", since Equals does not report a difference if the value is null
	if empty != string(res) {
		t.Fatalf("Did not get empty result, was:%s", string(res))
	}
}

func TestCreateMergePatchComplexAddAll(t *testing.T) {
	doc := `{"hello": "world","t": true ,"f": false, "n": null,"i": 123,"pi": 3.1416,"a": [1, 2, 3, 4], "nested": {"hello": "world","t": true ,"f": false, "n": null,"i": 123,"pi": 3.1416,"a": [1, 2, 3, 4]} }`
	empty := `{}`
	res, err := CreateMergePatch([]byte(empty), []byte(doc))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(doc, string(res)) {
		t.Fatalf("Did not get everything as, it was:\n%s", string(res))
	}
}

// createNestedMap created a series of nested map objects such that the number of
// objects is roughly 2^n (precisely, 2^(n+1)-1).
func createNestedMap(m map[string]interface{}, depth int, objectCount *int) {
	if depth == 0 {
		return
	}
	for i := 0; i < 2; i++ {
		nested := map[string]interface{}{}
		*objectCount += 1
		createNestedMap(nested, depth-1, objectCount)
		m[fmt.Sprintf("key-%v", i)] = nested
	}
}

func TestMatchesValue(t *testing.T) {
	testcases := []struct {
		name string
		a    interface{}
		b    interface{}
		want bool
	}{
		{
			name: "map empty",
			a:    map[string]interface{}{},
			b:    map[string]interface{}{},
			want: true,
		},
		{
			name: "map equal keys, equal non-nil value",
			a:    map[string]interface{}{"1": true},
			b:    map[string]interface{}{"1": true},
			want: true,
		},
		{
			name: "map equal keys, equal nil value",
			a:    map[string]interface{}{"1": nil},
			b:    map[string]interface{}{"1": nil},
			want: true,
		},

		{
			name: "map different value",
			a:    map[string]interface{}{"1": true},
			b:    map[string]interface{}{"1": false},
			want: false,
		},
		{
			name: "map different key, matching non-nil value",
			a:    map[string]interface{}{"1": true},
			b:    map[string]interface{}{"2": true},
			want: false,
		},
		{
			name: "map different key, matching nil value",
			a:    map[string]interface{}{"1": nil},
			b:    map[string]interface{}{"2": nil},
			want: false,
		},
		{
			name: "map different key, first nil value",
			a:    map[string]interface{}{"1": true},
			b:    map[string]interface{}{"2": nil},
			want: false,
		},
		{
			name: "map different key, second nil value",
			a:    map[string]interface{}{"1": nil},
			b:    map[string]interface{}{"2": true},
			want: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesValue(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func benchmarkMatchesValueWithDeeplyNestedFields(depth int, b *testing.B) {
	a := map[string]interface{}{}
	objCount := 1
	createNestedMap(a, depth, &objCount)
	b.ResetTimer()
	b.Run(fmt.Sprintf("objectCount=%v", objCount), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if !matchesValue(a, a) {
				b.Errorf("Should be equal")
			}
		}
	})
}

func BenchmarkMatchesValue1(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(1, b) }
func BenchmarkMatchesValue2(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(2, b) }
func BenchmarkMatchesValue3(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(3, b) }
func BenchmarkMatchesValue4(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(4, b) }
func BenchmarkMatchesValue5(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(5, b) }
func BenchmarkMatchesValue6(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(6, b) }
func BenchmarkMatchesValue7(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(7, b) }
func BenchmarkMatchesValue8(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(8, b) }
func BenchmarkMatchesValue9(b *testing.B)  { benchmarkMatchesValueWithDeeplyNestedFields(9, b) }
func BenchmarkMatchesValue10(b *testing.B) { benchmarkMatchesValueWithDeeplyNestedFields(10, b) }

func TestCreateMergePatchComplexRemoveAll(t *testing.T) {
	doc := `{"hello": "world","t": true ,"f": false, "n": null,"i": 123,"pi": 3.1416,"a": [1, 2, 3, 4], "nested": {"hello": "world","t": true ,"f": false, "n": null,"i": 123,"pi": 3.1416,"a": [1, 2, 3, 4]} }`
	exp := `{"a":null,"f":null,"hello":null,"i":null,"n":null,"nested":null,"pi":null,"t":null}`
	empty := `{}`
	res, err := CreateMergePatch([]byte(doc), []byte(empty))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if exp != string(res) {
		t.Fatalf("Did not get result, was:%s", string(res))
	}

	// FIXME: Crashes if using compareJSON like this:
	/*
		if !compareJSON(doc, string(res)) {
			t.Fatalf("Did not get everything as, it was:\n%s", string(res))
		}
	*/
}

func TestCreateMergePatchObjectWithInnerArray(t *testing.T) {
	stateString := `{
	  "OuterArray": [
	    {
		  "InnerArray": [
	        {
	          "StringAttr": "abc123"
	        }
	      ],
	      "StringAttr": "def456"
	    }
	  ]
	}`

	patch, err := CreateMergePatch([]byte(stateString), []byte(stateString))
	if err != nil {
		t.Fatal(err)
	}

	if string(patch) != "{}" {
		t.Fatalf("Patch should have been {} but was: %v", string(patch))
	}
}

func TestCreateMergePatchReplaceKeyNotEscape(t *testing.T) {
	doc := `{ "title": "hello", "nested": {"title/escaped": 1, "two": 2} }`
	pat := `{ "title": "goodbye", "nested": {"title/escaped": 2, "two": 2}  }`

	exp := `{ "title": "goodbye", "nested": {"title/escaped": 2}  }`

	res, err := CreateMergePatch([]byte(doc), []byte(pat))

	if err != nil {
		t.Errorf("Unexpected error: %s, %s", err, string(res))
	}

	if !compareJSON(exp, string(res)) {
		t.Log(string(res))
		t.Fatalf("Key was not replaced")
	}
}

func TestMergePatchReplaceKeyNotEscaping(t *testing.T) {
	doc := `{ "obj": { "title/escaped": "hello" } }`
	pat := `{ "obj": { "title/escaped": "goodbye" } }`
	exp := `{ "obj": { "title/escaped": "goodbye" } }`

	res := mergePatch(t, doc, pat)

	if !compareJSON(exp, res) {
		t.Fatalf("Key was not replaced")
	}
}

func TestMergeMergePatches(t *testing.T) {
	cases := []struct {
		demonstrates string
		p1           string
		p2           string
		exp          string
	}{
		{
			demonstrates: "simple patches are merged normally",
			p1:           `{"add1": 1}`,
			p2:           `{"add2": 2}`,
			exp:          `{"add1": 1, "add2": 2}`,
		},
		{
			demonstrates: "nulls are kept",
			p1:           `{"del1": null}`,
			p2:           `{"del2": null}`,
			exp:          `{"del1": null, "del2": null}`,
		},
		{
			demonstrates: "nulls are kept in complex objects",
			p1:           `{}`,
			p2:           `{"request":{"object":{"complex_object_array":["value1","value2","value3"],"complex_object_map":{"key1":"value1","key2":"value2","key3":"value3"},"simple_object_bool":false,"simple_object_float":-5.5,"simple_object_int":5,"simple_object_null":null,"simple_object_string":"example"}}}`,
			exp:          `{"request":{"object":{"complex_object_array":["value1","value2","value3"],"complex_object_map":{"key1":"value1","key2":"value2","key3":"value3"},"simple_object_bool":false,"simple_object_float":-5.5,"simple_object_int":5,"simple_object_null":null,"simple_object_string":"example"}}}`,
		},
		{
			demonstrates: "a key added then deleted is kept deleted",
			p1:           `{"add_then_delete": "atd"}`,
			p2:           `{"add_then_delete": null}`,
			exp:          `{"add_then_delete": null}`,
		},
		{
			demonstrates: "a key deleted then added is kept added",
			p1:           `{"delete_then_add": null}`,
			p2:           `{"delete_then_add": "dta"}`,
			exp:          `{"delete_then_add": "dta"}`,
		},
		{
			demonstrates: "object overrides array",
			p1:           `[]`,
			p2:           `{"del": null, "add": "a"}`,
			exp:          `{"del": null, "add": "a"}`,
		},
		{
			demonstrates: "array overrides object",
			p1:           `{"del": null, "add": "a"}`,
			p2:           `[]`,
			exp:          `[]`,
		},
	}

	for _, c := range cases {
		out, err := MergeMergePatches([]byte(c.p1), []byte(c.p2))

		if err != nil {
			panic(err)
		}

		if !compareJSON(c.exp, string(out)) {
			t.Logf("Error while trying to demonstrate: %v", c.demonstrates)
			t.Logf("Got %v", string(out))
			t.Logf("Expected %v", c.exp)
			t.Fatalf("Merged merge patch is incorrect")
		}
	}
}

func TestMergePatchWithOptions(t *testing.T) {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)

	v := struct {
		X string
	}{X: "&<>"}

	if err := enc.Encode(&v); err != nil {
		t.Fatal(err)
	}
	target := []byte(`{"key1": "val1", "key2": "val2"}`)

	opts := NewApplyOptions()
	opts.EscapeHTML = false

	modified, err := MergePatchWithOptions(b.Bytes(), target, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !compareJSON(string(modified), `{"X":"&<>","key1":"val1","key2":"val2"}`) {
		t.Fatalf("testMergePatchWithOptions fails for %s", string(modified))
	}
}
