package parsephp

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseStr parses a raw query string using DefaultOptions and returns a nested structure
// compatible with PHP's parse_str semantics (as closely as practical in Go):
// - Keys/values decoded using application/x-www-form-urlencoded rules
// - Bracket token semantics for arrays/maps (key[], key[0], key[sub], nested)
// - Duplicate semantics: last-wins for plain scalars; arrays append; mixed cases resolved per rules
// - Leaf values are strings; containers are map[string]any and []any; slices may contain nil
func ParseStr(query string) (map[string]any, error) {
	return ParseStrWithOptions(query, DefaultOptions)
}

// ParseStrWithOptions is like ParseStr but allows configuration via Options.
func ParseStrWithOptions(query string, opts Options) (map[string]any, error) {
	if len(opts.Separators) == 0 {
		opts.Separators = DefaultOptions.Separators
	}

	// Trim optional leading '?'
	if strings.HasPrefix(query, "?") {
		query = query[1:]
	}

	// Split pairs by the configured separators
	pairs := splitBySeparators(query, opts.Separators)
	root := make(map[string]any)

	for _, raw := range pairs {
		if raw == "" {
			// ignore completely empty pairs (e.g., leading/trailing separators or double separators)
			continue
		}
		// Split once on first '='; key without '=' => empty value
		k, v, hasEq := splitPair(raw)

		// Decode key/value
		dk, errK := decode(k, opts.StrictDecode)
		dv, errV := decode(v, opts.StrictDecode)
		// Lenient policy: if strict=false, decode returns no error and keeps invalid sequences
		if opts.StrictDecode {
			// If strict, any error bubbles up
			if errK != nil {
				return nil, fmt.Errorf("decode key error: %w", errK)
			}
			if errV != nil {
				return nil, fmt.Errorf("decode value error: %w", errV)
			}
		}
		dk = strings.TrimSpace(dk)
		dv = strings.TrimSpace(dv)

		if dk == "" {
			// ignore empty keys (robustness; PHP would create a variable with empty name, which is awkward in Go)
			continue
		}

		// Tokenize decoded key into base + bracket tokens
		seq := tokenizeKey(dk)
		if len(seq) == 0 {
			// Shouldn't happen; but guard anyway
			continue
		}
		base := seq[0]
		tokens := seq[1:]

		// Insert according to tokens
		if len(tokens) == 0 {
			// Plain scalar assignment, last wins policy
			root[base] = dv
			continue
		}

		insert(root, base, tokens, dv)
		_ = hasEq // only used to compute dv empty when no '='; dv already set
	}

	return root, nil
}

// splitPair splits a raw pair into key and value, only on the first '='.
// Returns key, value, and a boolean indicating if '=' existed.
func splitPair(s string) (string, string, bool) {
	if s == "" {
		return "", "", false
	}
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[:i], s[i+1:], true
	}
	return s, "", false
}

// splitBySeparators splits s by any rune in seps. Empty segments are preserved (caller may ignore).
func splitBySeparators(s string, seps []rune) []string {
	if s == "" {
		return []string{}
	}
	// Build a set for quick lookup
	sepSet := make(map[rune]struct{}, len(seps))
	for _, r := range seps {
		sepSet[r] = struct{}{}
	}
	var out []string
	var b strings.Builder
	for _, r := range s {
		if _, isSep := sepSet[r]; isSep {
			out = append(out, b.String())
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	out = append(out, b.String())
	return out
}

// decode applies application/x-www-form-urlencoded rules.
// When strict=false, returns lenient decoding: invalid percent sequences are left as literal characters.
// When strict=true, errors from url.QueryUnescape are returned.
func decode(s string, strict bool) (string, error) {
	if !strict {
		// Try fast path
		d, err := url.QueryUnescape(s)
		if err == nil {
			return d, nil
		}
		// Fallback to lenient decoder
		return lenientDecode(s), nil
	}
	d, err := url.QueryUnescape(s)
	if err != nil {
		return "", err
	}
	return d, nil
}

// lenientDecode performs application/x-www-form-urlencoded decoding without failing on malformed escapes.
// '+' -> space; valid %XX hex are decoded; invalid '%' sequences are kept literally.
func lenientDecode(s string) string {
	// Replace '+' first (per x-www-form-urlencoded semantics)
	// We do this manually in the loop, to avoid double pass.
	var out []byte
	b := []byte(s)
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch c {
		case '+':
			out = append(out, ' ')
		case '%':
			if i+2 < len(b) && isHex(b[i+1]) && isHex(b[i+2]) {
				hx := string(b[i+1 : i+3])
				v, err := strconv.ParseUint(hx, 16, 8)
				if err != nil {
					// Shouldn't happen because we checked isHex, but keep literal just in case
					out = append(out, '%')
				} else {
					out = append(out, byte(v))
					i += 2
				}
			} else {
				// invalid percent; keep literal '%'
				out = append(out, '%')
				// do not consume following bytes; they will be appended normally
			}
		default:
			out = append(out, c)
		}
	}
	return string(out)
}

func isHex(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// tokenizeKey splits a decoded key into base + bracket tokens, matching the clarified PHP boundary behaviors.
// Rules:
// - Only matched bracket pairs "[...]" become tokens.
// - Unmatched '[' is converted into an underscore '_' in the base, and the remaining characters are scanned as base.
// - Unmatched ']' is ignored (dropped).
func tokenizeKey(s string) []string {
	if s == "" {
		return nil
	}
	var baseB strings.Builder
	var tokens []string
	for i := 0; i < len(s); {
		c := s[i]
		if c == '[' {
			// search for the next ']'
			j := i + 1
			for j < len(s) && s[j] != ']' {
				j++
			}
			if j < len(s) {
				// matched bracket pair -> emit token
				tokens = append(tokens, s[i+1:j])
				i = j + 1
				// skip any immediate extra consecutive ']' characters after a matched pair
				for i < len(s) && s[i] == ']' {
					i++
				}
				continue
			}
			// unmatched '[' -> convert to '_' and continue scanning the rest as part of base (treat future ']' literally)
			baseB.WriteByte('_')
			i++
			continue
		}
		// outside of bracket-token parsing, a stray ']' is treated as a literal and kept in the base key
		baseB.WriteByte(c)
		i++
	}
	if len(tokens) == 0 {
		return []string{baseB.String()}
	}
	res := make([]string, 1, 1+len(tokens))
	res[0] = baseB.String()
	res = append(res, tokens...)
	return res
}

// insert updates root[base] following bracket tokens, creating containers as needed per rules.
// Containers:
// - Numeric tokens => ensure slice and set at index (expanding with nils)
// - Empty token "" => append; for non-leaf, append a new container decided by the next token
// - Non-numeric tokens => ensure map and set by key; last-wins for duplicate leaves
//
// Mixed scalar/array/map resolution:
// - If base doesn't exist, choose container by first token: "" or numeric => slice; non-numeric => map
// - If base exists as string and first token is ""/numeric => convert to slice with the prior scalar as first element
// - If base exists as string and first token is non-numeric => convert to map; prior scalar is discarded (mirrors PHP behavior for key[sub])
// - If base exists as slice/map, keep existing container type
func insert(root map[string]any, base string, tokens []string, value string) {
	if len(tokens) == 0 {
		root[base] = value
		return
	}

	// Ensure base container type per first token
	current, exists := root[base]
	first := tokens[0]

	// setter writes the updated container header back to the parent location
	setBase := func(updated any) {
		root[base] = updated
	}

	if !exists {
		if first == "" || isNumeric(first) {
			current = []any{}
		} else {
			current = make(map[string]any)
		}
		setBase(current)
	} else {
		switch c := current.(type) {
		case string:
			if first == "" || isNumeric(first) {
				// Convert to slice, put prior scalar as first element
				current = []any{c}
				setBase(current)
			} else {
				// Convert to map, drop prior scalar (PHP: a=1 then a[b]=2 => a becomes array with b)
				current = make(map[string]any)
				setBase(current)
			}
		case []any:
			current = c // keep slice
		case map[string]any:
			current = c // keep map
		default:
			// unexpected type; replace according to first token for robustness
			if first == "" || isNumeric(first) {
				current = []any{}
			} else {
				current = make(map[string]any)
			}
			setBase(current)
		}
	}

	// Traverse tokens
	cur := current
	setCur := setBase // initially, mutations at this level should update root[base]

	for idx, tok := range tokens {
		isLeaf := idx == len(tokens)-1
		nextTok := ""
		if !isLeaf {
			nextTok = tokens[idx+1]
		}

		if tok == "" { // append semantics
			// Hybrid behavior: if current is a slice, append to slice; if current is a map, append under next numeric string key.
			switch c := cur.(type) {
			case []any:
				// Slice append semantics (unchanged)
				sl := c
				if isLeaf {
					sl = append(sl, value)
					setCur(sl)
					return
				}
				var child any
				if nextTok == "" || isNumeric(nextTok) {
					child = []any{}
				} else {
					child = make(map[string]any)
				}
				sl = append(sl, child)
				setCur(sl)
				idxInParent := len(sl) - 1
				cur = child
				setCur = func(updated any) { sl[idxInParent] = updated }
				continue
			case map[string]any:
				// Map hybrid append: choose next auto index key and set child/value under that string key
				mp := c
				key := strconv.Itoa(nextAutoIndex(mp))
				if isLeaf {
					mp[key] = value
					setCur(mp)
					return
				}
				var child any
				if nextTok == "" || isNumeric(nextTok) {
					child = []any{}
				} else {
					child = make(map[string]any)
				}
				mp[key] = child
				setCur(mp)
				cur = child
				setCur = func(updated any) { mp[key] = updated }
				continue
			default:
				// Unknown or nil: default to slice semantics for robustness
				sl := ensureSlice(cur)
				setCur(sl)
				if isLeaf {
					sl = append(sl, value)
					setCur(sl)
					return
				}
				var child any
				if nextTok == "" || isNumeric(nextTok) {
					child = []any{}
				} else {
					child = make(map[string]any)
				}
				sl = append(sl, child)
				setCur(sl)
				idxInParent := len(sl) - 1
				cur = child
				setCur = func(updated any) { sl[idxInParent] = updated }
				continue
			}
		}

		if isNumeric(tok) { // numeric index
			// If current is a map, treat numeric token as a string key under the map (hybrid semantics).
			switch c := cur.(type) {
			case map[string]any:
				mp := ensureMap(c)
				setCur(mp)
				if isLeaf {
					mp[tok] = value
					setCur(mp)
					return
				}
				child, ok := mp[tok]
				if !ok || child == nil {
					if nextTok == "" || isNumeric(nextTok) {
						child = []any{}
					} else {
						child = make(map[string]any)
					}
					mp[tok] = child
				} else {
					switch child.(type) {
					case []any, map[string]any:
						// OK
					case string:
						if nextTok == "" || isNumeric(nextTok) {
							child = []any{}
						} else {
							child = make(map[string]any)
						}
						mp[tok] = child
					default:
						if nextTok == "" || isNumeric(nextTok) {
							child = []any{}
						} else {
							child = make(map[string]any)
						}
						mp[tok] = child
					}
				}
				cur = child
				key := tok
				setCur = func(updated any) { mp[key] = updated }
				continue
			default:
				// Default behavior: ensure slice and set by numeric index
				sl := ensureSlice(cur)
				setCur(sl)
				n, _ := strconv.Atoi(tok) // safe due to isNumeric
				sl = growSlice(sl, n)
				setCur(sl)
				if isLeaf {
					sl[n] = value
					setCur(sl)
					return
				}
				child := sl[n]
				if child == nil {
					if nextTok == "" || isNumeric(nextTok) {
						child = []any{}
					} else {
						child = make(map[string]any)
					}
					sl[n] = child
					setCur(sl)
				} else {
					switch child.(type) {
					case []any, map[string]any:
						// OK
					case string:
						if nextTok == "" || isNumeric(nextTok) {
							child = []any{}
						} else {
							child = make(map[string]any)
						}
						sl[n] = child
						setCur(sl)
					default:
						if nextTok == "" || isNumeric(nextTok) {
							child = []any{}
						} else {
							child = make(map[string]any)
						}
						sl[n] = child
						setCur(sl)
					}
				}
				cur = child
				setCur = func(updated any) { sl[n] = updated }
				continue
			}
		}

		// non-numeric associative key => map
		mp := ensureMap(cur)
		setCur(mp)

		if isLeaf {
			mp[tok] = value
			// Map header doesn't need reassign (reference type), but keep consistent
			setCur(mp)
			return
		}

		child, ok := mp[tok]
		if !ok || child == nil {
			if nextTok == "" || isNumeric(nextTok) {
				child = []any{}
			} else {
				child = make(map[string]any)
			}
			mp[tok] = child
		} else {
			switch child.(type) {
			case []any, map[string]any:
				// OK
			case string:
				if nextTok == "" || isNumeric(nextTok) {
					child = []any{}
				} else {
					child = make(map[string]any)
				}
				mp[tok] = child
			default:
				if nextTok == "" || isNumeric(nextTok) {
					child = []any{}
				} else {
					child = make(map[string]any)
				}
				mp[tok] = child
			}
		}
		// Descend
		key := tok
		cur = child
		setCur = func(updated any) { mp[key] = updated }
	}
}

// ensureSlice coerces container to []any. If it is a map, we replace it (robust resolution per tokens).
func ensureSlice(container any) []any {
	if container == nil {
		return []any{}
	}
	if sl, ok := container.([]any); ok {
		return sl
	}
	// If a map is here but tokens require a slice, we replace it
	return []any{}
}

// ensureMap coerces container to map[string]any. If it is a slice, we replace it (robust resolution per tokens).
func ensureMap(container any) map[string]any {
	if container == nil {
		return make(map[string]any)
	}
	switch v := container.(type) {
	case map[string]any:
		return v
	case []any:
		// convert slice elements to string-indexed map entries to preserve content
		m := make(map[string]any, len(v))
		for i, elem := range v {
			m[strconv.Itoa(i)] = elem
		}
		return m
	default:
		// string or other types -> start fresh map
		return make(map[string]any)
	}
}

// growSlice ensures sl has length > idx, expanding with nils.
func growSlice(sl []any, idx int) []any {
	if idx < 0 {
		return sl
	}
	// Expand as needed
	for len(sl) <= idx {
		sl = append(sl, nil)
	}
	return sl
}

// isNumeric reports whether the token is an unsigned integer consisting of digits only.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// nextAutoIndex scans a map's keys and returns the next automatic numeric index
// (max existing numeric key + 1), or 0 if none exist. Numeric keys are strings of digits only.
func nextAutoIndex(m map[string]any) int {
	max := -1
	for k := range m {
		if isNumeric(k) {
			if n, err := strconv.Atoi(k); err == nil {
				if n > max {
					max = n
				}
			}
		}
	}
	return max + 1
}

// writeBack is a no-op shim that returns the up-to-date container.
// In this implementation we mutate the container directly and return it,
// keeping the code intentional and simple.
func writeBack(child any, parent any) any {
	// This function exists to document the intended flow and future-proof refactors where
	// we may need to handle parent references explicitly.
	return child
}

// Errors for potential future expansion
var (
	ErrInvalidPercent = errors.New("invalid percent-escape")
)
