package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	// https://stackoverflow.com/a/74328802
	"github.com/go-playground/validator/v10"
	"github.com/nfx/go-htmltable"
)

// Status types
// https://stackoverflow.com/a/71934535
type SchemeStatus string

const (
	Permanent   SchemeStatus = "Permanent"
	Provisional SchemeStatus = "Provisional"
	Historical  SchemeStatus = "Historical"
)

type Scheme struct {
	UriScheme           string       `header:"URI Scheme"`
	Template            string       `header:"Template"`
	Description         string       `header:"Description"`
	Status              SchemeStatus `header:"Status" validate:"oneof=Permanent Provisional Historical"`
	WellKnownUriSupport string       `header:"Well-Known URI Support"`
	Reference           string       `header:"Reference"`
	Notes               string       `header:"Notes"`
}

// As well as [a-z], these characters are allowed in URI schemes
// https://github.com/JuliaWeb/URIs.jl/blob/dce395c3/src/URIs.jl#L91-L108
// TODO: handle user info and IPv6 hosts
var ADDITIONAL_ALLOWED_SCHEME_CHARS = []rune{'-', '+', '.'}
var SCHEME_PATTERN = schemePattern()
var CLEAN_SCHEME_PATTERN = cleanSchemePattern()

// Validate Scheme struct
// https://stackoverflow.com/a/71934231
func (s *Scheme) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}

// Construct scheme pattern to use in validation/cleaning step
func schemePattern() *regexp.Regexp {
	var allowedChars string
	for _, char := range ADDITIONAL_ALLOWED_SCHEME_CHARS {
		allowedChars += string(char)
	}
	pattern := fmt.Sprintf(`[\w%s]+`, regexp.QuoteMeta(allowedChars))
	return regexp.MustCompile(pattern)
}

// Schemes from IANA can contain additional information in parentheses
func cleanSchemePattern() *regexp.Regexp {
	pattern := fmt.Sprintf(`^(%s)(?:\s+\((.*)\))?$`, SCHEME_PATTERN)
	return regexp.MustCompile(pattern)
}

// Within s, replace characters at `positions' with the rune defined in `replacement`
//
// For example:
// ```go
// replaceAtPositions("hello", []int{1, 2}, rune('x')) == "hxxlo"
// ```
func replaceAtPositions(s string, positions []int, replacement rune) string {
	runes := []rune(s)

	for _, pos := range positions {
		if pos >= 0 && pos < len(runes) {
			runes[pos] = replacement
		}
	}

	return string(runes)
}

// The goal of defanging is to malform the URI such that it does not open if clicked
//
// However, as there is a *[re]fang* option in the Tomtils library, we need an algorithm
// to map invertibly fanged and defanged schemes.  Many libraries do not support schemes
// beyond http[s] [1, 2], as browsers do not support many different schemes.  However,
// it may be the case that different schemes are supported on different non-browser
// applications, so we *should* support defanging.
//
// There is also consideration to have enough information in a defanged stream such that
// it is invertible to its original scheme.
//
// [1]: https://stackoverflow.com/a/56150152
// [2]: https://github.com/ioc-fang/ioc_fanger
func defangScheme(scheme string) string {
	// TODO
	return ""
}

func toScreamingSnake(input string) string {
	// Regular expression to match camelCase words
	re := regexp.MustCompile("([a-z])([A-Z])")

	// Insert a space between camelCase words and replace spaces with underscores
	snake := re.ReplaceAllString(input, "${1}_${2}")
	snake = strings.ReplaceAll(snake, " ", "_")

	// Convert to upper case
	return strings.ToUpper(snake)
}

func constructPyList(strs []string, varName string) string {
	// Create a string that can be pasted into Python
	//
	// Maximum line length as per PEP-8:
	// https://peps.python.org/pep-0008#maximum-line-length
	maxLineLength := 79
	indentNumber := 4
	currentLineLength := 0
	var lines []string
	var currentLine strings.Builder
	for _, str := range strs {
		strStr := fmt.Sprintf("\"%s\",", str)

		// New line if the addition of the scheme will go over the maximum
		// line length as defined by PEP-8
		if currentLineLength+len(strStr) > maxLineLength {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLineLength = 0
		}

		// Add indent to each new line
		// https://stackoverflow.com/a/22979015
		//
		// Use spaces and indent of 4
		if currentLine.Len() == 0 {
			indent := strings.Repeat(" ", indentNumber)
			currentLine.WriteString(indent)
			currentLineLength = indentNumber
		}

		// Add space between elements of the list
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
			currentLineLength += 1
		}

		// Add the scheme to the current line
		currentLine.WriteString(strStr)
		currentLineLength += len(strStr)
	}

	// Add the final line to the list
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	// Join the output
	varName = toScreamingSnake(varName)
	return fmt.Sprintf("%s = [\n%s\n]", varName, strings.Join(lines, "\n"))
}

func constructPySchemeList(schemes []Scheme, varName string) string {
	var rawSchemes []string

	for _, scheme := range schemes {
		rawSchemes = append(rawSchemes, scheme.UriScheme)
	}

	return constructPyList(rawSchemes, varName)
}

func constructPyDict(keys []string, values []string, varName string) string {
	if len(keys) != len(values) {
		fmt.Printf("[ERROR] Keys and values must be the same length: keys length = %d, values length = %d", len(keys), len(values))
		os.Exit(1)
	}

	indentNumber := 4
	var lines []string

	// Each new key-value pair is on a new line
	// https://stackoverflow.com/a/18139301
	for i, key := range keys {
		indent := strings.Repeat(" ", indentNumber)
		lines = append(lines, fmt.Sprintf("%s\"%s\": \"%s\",", indent, key, values[i]))
	}

	return fmt.Sprintf("%s = {\n%s\n}", varName, strings.Join(lines, "\n"))
}

func constructPyDefangSchemeDict(schemes []Scheme, varName string) string {
	var rawSchemes []string
	var defangedSchemes []string

	for _, scheme := range schemes {
		rawSchemes = append(rawSchemes, scheme.UriScheme)
		defangedSchemes = append(defangedSchemes, defangScheme(scheme.UriScheme))
	}

	return constructPyDict(rawSchemes, defangedSchemes, varName)
}

// Mostly, the `URI Scheme` field is good, but there is a scheme called `shttp (OBSOLETE)`,
// which we need to clean up
func cleanScheme(scheme Scheme) Scheme {
	schemeRaw := scheme.UriScheme
	matches := CLEAN_SCHEME_PATTERN.FindStringSubmatch(schemeRaw)

	if matches == nil || len(matches) == 0 {
		fmt.Printf("[ERROR] Invalid scheme for \"%s\"", schemeRaw)
		os.Exit(1)
	}

	// Set the first match to the URI scheme
	// NOTE: we start counting from 1 because the first element is the entire match
	scheme.UriScheme = matches[1]

	// If the URI scheme holds additional information, add it to notes
	if len(matches) > 2 && matches[2] != "" {
		scheme.Notes = matches[2]
	}

	// Confirm we don't have any unhandled matching information
	if len(matches) > 3 {
		fmt.Printf("[ERROR] Unhandled matching groups in scheme regex for \"%s\"", schemeRaw)
		os.Exit(1)
	}

	// Ensure scheme is lowercase
	scheme.UriScheme = strings.ToLower(scheme.UriScheme)

	// Return the (potentially modified) scheme
	return scheme
}

// Importantly, confirm that a defanged scheme is not still a valid scheme
func schemeIsKnown(scheme string, knownSchemes []Scheme) bool {
	scheme = strings.ToLower(scheme)
	for _, knownScheme := range knownSchemes {
		if strings.ToLower(scheme) == knownScheme.UriScheme {
			return true
		}
	}
	return false
}

func main() {
	htmltable.Logger = func(_ context.Context, msg string, fields ...any) {
		fmt.Printf("[INFO] %s %v\n", msg, fields)
	}

	// Get URI Scheme table from IANA (based on RFC 7595)
	// https://stackoverflow.com/a/42289198
	url := "https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml"
	table, err := htmltable.NewSliceFromURL[Scheme](url)
	if err != nil {
		fmt.Printf("[ERROR] Could not get table by %s: %s", url, err)
		os.Exit(1)
	}

	// Collect URI schemes into a string list
	var schemes []Scheme
	for i := 0; i < len(table); i++ {
		scheme := table[i]
		err := scheme.Validate()
		if err != nil {
			fmt.Printf("[ERROR] Invalid Scheme struct: %s; Scheme: %v", err, scheme)
			os.Exit(1)
		}
		scheme = cleanScheme(scheme)
		if scheme.Status == Permanent {
			schemes = append(schemes, scheme)
		}
	}

	// Confirm that no defanged schemes are known!
	fmt.Println("[INFO] Checking that the defang algorithm does not produce any valid schemes")
	for _, scheme := range schemes {
		defangedScheme := defangScheme(scheme.UriScheme)
		if schemeIsKnown(defangedScheme, schemes) {
			fmt.Printf("[ERROR] Defanged scheme \"%s\" is still a valid scheme", defangedScheme)
			os.Exit(1)
		}
	}

	// Filter for permanent schemes
	var permanentSchemes []Scheme
	for _, scheme := range schemes {
		if scheme.Status == Permanent {
			permanentSchemes = append(permanentSchemes, scheme)
		}
	}

	// Format the output as a Python list
	pyStr := constructPySchemeList(permanentSchemes, "uriSchemes")
	fmt.Println(pyStr)

	pyDict := constructPyDefangSchemeDict(permanentSchemes, "uriSchemesDefangedMap")
	fmt.Println(pyDict)
}
