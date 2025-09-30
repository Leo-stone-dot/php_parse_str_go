package parsephp

import (
	"reflect"
	"testing"
)

func TestPlainDuplicateLastWins(t *testing.T) {
	got, err := ParseStr("a=b&a=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSimpleArrayAppend(t *testing.T) {
	got, err := ParseStr("a[]=b&a[]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{"b", "c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestArrayNumericIndexWithGaps(t *testing.T) {
	got, err := ParseStr("a[0]=b&a[2]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{"b", nil, "c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestNestedAssociative(t *testing.T) {
	got, err := ParseStr("a[b][c]=d&a[b][e]=f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": map[string]any{"c": "d", "e": "f"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestNestedAppendContainers(t *testing.T) {
	got, err := ParseStr("a[][b]=c&a[][b]=d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{map[string]any{"b": "c"}, map[string]any{"b": "d"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestScalarThenArray(t *testing.T) {
	got, err := ParseStr("a=1&a[]=2&a[]=3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{"1", "2", "3"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSemicolonSeparators(t *testing.T) {
	got, err := ParseStr(";a=b;c=d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": "b", "c": "d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestDecodingDemonstration(t *testing.T) {
	got, err := ParseStr("q=%2B+%2520")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"q": "+ %20"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestKeyWithoutEqual(t *testing.T) {
	got, err := ParseStr("flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"flag": ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLeadingQuestionMark(t *testing.T) {
	got, err := ParseStr("?x=1&y=2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"x": "1", "y": "2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestNumericNestedIndexing(t *testing.T) {
	got, err := ParseStr("a[0][1]=x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{[]any{nil, "x"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestRepeatAssociativeKeyLastWins(t *testing.T) {
	got, err := ParseStr("a[b]=x&a[b]=y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
func TestOverwrite(t *testing.T) {
	got, err := ParseStr("f[][]=m&f[][]=n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"f": []any{[]any{"m"}, []any{"n"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestEmpty(t *testing.T) {
	got, err := ParseStr("f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"f": ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestMix(t *testing.T) {
	got, err := ParseStr("a[b][c]=d&a[d]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"d": "c", "b": map[string]any{"c": "d"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
