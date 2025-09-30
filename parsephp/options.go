package parsephp

// Options defines configurable behavior for parsing.
// Future-friendly: You can expand fields without breaking ParseStr defaults.
//
// Separators: characters used to split pairs. Defaults to '&' and ';' (to mirror PHP's arg_separator.input).
// StrictDecode: if true, decoding errors (malformed percent-escapes) will be returned as errors.
//              if false, decoder is lenient: invalid escape sequences are kept as-is without failing the whole parse.
//
// Note: ParseStr uses DefaultOptions.
type Options struct {
    Separators   []rune
    StrictDecode bool
}

// DefaultOptions used by ParseStr.
var DefaultOptions = Options{
    Separators:   []rune{'&', ';'},
    StrictDecode: false,
}
