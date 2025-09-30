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

func TestAssociativeSiblingsUnderSameBase(t *testing.T) {
	got, err := ParseStr("a[b][c]=d&a[d]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": map[string]any{"c": "d"}, "d": "c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestHybridAppendUnderMap(t *testing.T) {
	got, err := ParseStr("a[b][c]=d&a[][d]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": map[string]any{"c": "d"}, "0": map[string]any{"d": "c"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAutoIndexProgressionInMap(t *testing.T) {
	got, err := ParseStr("a[b]=x&a[]=y&a[]=z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": "x", "0": "y", "1": "z"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestMixedNumericAssociativeThenAppendHybrid(t *testing.T) {
	got, err := ParseStr("a[0]=x&a[b]=y&a[]=z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"0": "x", "b": "y", "1": "z"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestMalformed_StrayCloseBracketIgnored(t *testing.T) {
	got, err := ParseStr("a[b][c]=d&a[d]]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": map[string]any{"c": "d"}, "d": "c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestMalformed_UnmatchedOpenBracketConvertedToUnderscore(t *testing.T) {
	got, err := ParseStr("a[b][c]=d&a[d]=c&a[=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": map[string]any{"c": "d"}, "d": "c"}, "a_": "1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestUnmatchedOpenBracketInMiddle(t *testing.T) {
	got, err := ParseStr("p[q=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"p_q": "1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestStrayCloseBracketInBaseLiteralKept(t *testing.T) {
	got, err := ParseStr("b]=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"b]": "1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestExtraCloseBracketAfterMatchedTokenIgnored(t *testing.T) {
	got, err := ParseStr("a[b]]=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": "1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
