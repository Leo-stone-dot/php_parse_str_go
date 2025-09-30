package parsephp

import (
	"reflect"
	"testing"
)

// 1) 分隔符与键值对
func TestAudit_Separators_EmptySegmentsAndTrailing(t *testing.T) {
	cases := []struct {
		in   string
		out  map[string]any
		name string
	}{
		{"a=1&&b=2", map[string]any{"a": "1", "b": "2"}, "double_ampersand"},
		{"a=1;;b=2", map[string]any{"a": "1", "b": "2"}, "double_semicolon"},
		{"a=1&b=2&", map[string]any{"a": "1", "b": "2"}, "trailing_ampersand"},
		{"a=1;b=2;", map[string]any{"a": "1", "b": "2"}, "trailing_semicolon"},
	}
	for _, c := range cases {
		got, err := ParseStr(c.in)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if !reflect.DeepEqual(got, c.out) {
			t.Fatalf("%s: got %#v, want %#v", c.name, got, c.out)
		}
	}
}

func TestAudit_Separators_SemicolonAndMixed(t *testing.T) {
	got, err := ParseStr(";x=1;y=2&a=3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"x": "1", "y": "2", "a": "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Separators_LeadingQuestion(t *testing.T) {
	got, err := ParseStr("?x=1&y=2&")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"x": "1", "y": "2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_PairWithoutEqual_ScalarAndBracket(t *testing.T) {
	got1, err1 := ParseStr("flag")
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	want1 := map[string]any{"flag": ""}
	if !reflect.DeepEqual(got1, want1) {
		t.Fatalf("got %#v, want %#v", got1, want1)
	}

	got2, err2 := ParseStr("a[b]")
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	want2 := map[string]any{"a": map[string]any{"b": ""}}
	if !reflect.DeepEqual(got2, want2) {
		t.Fatalf("got %#v, want %#v", got2, want2)
	}
}

// 2) 解码与鲁棒性
func TestAudit_Decoding_PlusPercent_Lenient(t *testing.T) {
	got, err := ParseStr("q=%2B+%2520")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"q": "+ %20"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Decoding_MalformedEscape_Lenient(t *testing.T) {
	got, err := ParseStr("bad=%ZZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"bad": "%ZZ"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Decoding_UnicodeKeysValues(t *testing.T) {
	got, err := ParseStr("城市=北京&k=%E4%B8%AD%E6%96%87")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"城市": "北京", "k": "中文"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Decoding_TrimSpacesAround(t *testing.T) {
	got, err := ParseStr("   k   =   v   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"k": "v"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// 3) 括号语法要点
func TestAudit_Bracket_ArraysAppend(t *testing.T) {
	got, err := ParseStr("a[]=b&a[]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{"b", "c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Bracket_NumericHoles(t *testing.T) {
	got, err := ParseStr("a[0]=b&a[2]=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{"b", nil, "c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Bracket_AssociativeNesting(t *testing.T) {
	got, err := ParseStr("a[b][c]=d&a[b][e]=f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": map[string]any{"c": "d", "e": "f"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Bracket_AppendContainerInference(t *testing.T) {
	got, err := ParseStr("a[][b]=c&a[][b]=d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{map[string]any{"b": "c"}, map[string]any{"b": "d"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Bracket_ScalarToArrayUpgrade(t *testing.T) {
	got, err := ParseStr("a=1&a[]=2&a[]=3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": []any{"1", "2", "3"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Bracket_NumericTokenUnderMapHybrid(t *testing.T) {
	got, err := ParseStr("a[b]=x&a[0]=y&a[]=z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": "x", "0": "y", "1": "z"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// 4) 混合 map/slice 语义（同一 base）
func TestAudit_Mixed_BaseMapThenAppend(t *testing.T) {
	got, err := ParseStr("a[b]=x&a[]=y&a[]=z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": "x", "0": "y", "1": "z"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Mixed_BaseSliceThenIndexThenAssoc(t *testing.T) {
	got, err := ParseStr("a[]=x&a[2]=y&a[b]=z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"0": "x", "1": nil, "2": "y", "b": "z"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// 5) 括号畸形细则
func TestAudit_Malformed_UnmatchedOpenBracketAtDeeperLevel(t *testing.T) {
	got, err := ParseStr("a[b][=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a_": map[string]any{"b": "1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Malformed_ExtraCloseBracketOnlyAfterMatched(t *testing.T) {
	got, err := ParseStr("a[b]][][c]=x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": []any{map[string]any{"c": "x"}}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// 6) 重复与覆盖
func TestAudit_Repeat_ScalarLastWins(t *testing.T) {
	got, err := ParseStr("a=b&a=c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Repeat_AssociativeLeafLastWins(t *testing.T) {
	got, err := ParseStr("a[b]=x&a[b]=y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestAudit_Repeat_ArrayEstablishedThenPlainScalar_LastWinsPHP(t *testing.T) {
	// 选择对齐 PHP：后到的纯标量覆盖数组
	got, err := ParseStr("a[]=x&a=Y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": "Y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// 7) 极端索引与深度
func TestAudit_Extreme_LargeNumericIndex_NoPanic(t *testing.T) {
	got, err := ParseStr("a[1000]=x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 期望为切片扩容；只断言关键位置与长度，不比较整片结构
	m := got["a"].([]any)
	if len(m) != 1001 {
		t.Fatalf("len=%d, want=1001", len(m))
	}
	if m[1000] != "x" {
		t.Fatalf("index 1000 value=%#v, want 'x'", m[1000])
	}
}

func TestAudit_Extreme_DeepNestingStability(t *testing.T) {
	got, err := ParseStr("a[b][c][d][e][f]=x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": map[string]any{"e": map[string]any{"f": "x"}}}}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// 8) token 内的边缘解码（编码的括号在 token 内按字面处理）
func TestAudit_TokenInnerDecoding_EncodedBracketsLiteral(t *testing.T) {
	got1, err1 := ParseStr("a[%5D]=x")
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	want1 := map[string]any{"a": map[string]any{"]": "x"}}
	if !reflect.DeepEqual(got1, want1) {
		t.Fatalf("got %#v, want %#v", got1, want1)
	}

	got2, err2 := ParseStr("a[%5B]=y")
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	want2 := map[string]any{"a": map[string]any{"[": "y"}}
	if !reflect.DeepEqual(got2, want2) {
		t.Fatalf("got %#v, want %#v", got2, want2)
	}
}
