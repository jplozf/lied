package edit

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"regexp"
)

// ****************************************************************************
// TYPES
// ****************************************************************************
type Func struct {
	line int
	name string
}

// ****************************************************************************
// GetFuncs()
// ****************************************************************************
func GetFuncs(source string, language string) []Func {
	var funcs []Func
	// The regex pattern to extract function names
	// The function name is in the first capturing group

	regexPatternGolang := `func(?:\s+\([^)]+\))?\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`
	regexPatternPython := `def\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`
	regexPatternC_CPP := `\b(?:(?:[a-zA-Z_][a-zA-Z0-9_]*\s*(?:\*|&)?\s+)|(?:virtual|inline|explicit|static|friend|extern)\s+)?(?:[a-zA-Z_][a-zA-Z0-9_]*::)*([a-zA-Z_~][a-zA-Z0-9_]*)\s*\([^)]*\)\s*(?:const|override|final|noexcept)?\s*\{`
	regexPatternJava := `\b(?:public|private|protected|static|final|abstract|synchronized|native|strictfp|default)?\s*(?:<[a-zA-Z0-9_,\s]+>\s*)?(?:[a-zA-Z_][a-zA-Z0-9_<>,[\]\.]+\s+)+([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)`
	regexPatternRust := `\bfn\s+(?:[a-zA-Z_][a-zA-Z0-9_]*::)*([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)`
	regexPatternBash := `(?:function\s+)?([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:\(\))?\s*\{`
	regexPatternJavaScript := `(?:function\s+([a-zA-Z_][a-zA-Z0-9_]*)|(?:const|let|var)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(?:function|\(?[^)]*\)?\s*=>)|([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\))`
	var zeRegex string

	cppKeywords := map[string]bool{
		"if": true, "for": true, "while": true, "switch": true, "catch": true, "connect": true,
		"new": true, "delete": true, "return": true, "goto": true, "sizeof": true, "typeid": true,
		"alignof": true, "decltype": true, "noexcept": true, "static_assert": true, "thread_local": true,
		"const_cast": true, "dynamic_cast": true, "reinterpret_cast": true, "static_cast": true,
	}

	switch language {
	case "c++", "c":
		zeRegex = regexPatternC_CPP // OK
	case "java":
		zeRegex = regexPatternJava // OK
	case "shell":
		zeRegex = regexPatternBash // OK
	case "rust":
		zeRegex = regexPatternRust // OK
	case "go":
		zeRegex = regexPatternGolang // OK
	case "python":
		zeRegex = regexPatternPython // OK
	case "javascript", "js", "typescript", "ts":
		zeRegex = regexPatternJavaScript
	}
	// Compile the regex
	re := regexp.MustCompile(zeRegex)

	// Find all matches in the entire source string
	matches := re.FindAllSubmatchIndex([]byte(source), -1)

	for _, match := range matches {
		// Iterate through capturing groups to find the function name
		for i := 2; i < len(match); i += 2 { // Start from index 2 for the first capturing group
			if match[i] != -1 { // Check if the capturing group matched
				funcName := source[match[i]:match[i+1]]

				// Filter out C++ keywords if the language is C++
				if (language == "c++" || language == "c") && cppKeywords[funcName] {
					continue // Skip this match if it's a C++ keyword
				}

				// Determine the line number by counting newlines manually
				lineNumber := 1
				for charIndex := 0; charIndex < match[0]; charIndex++ {
					if source[charIndex] == '\n' {
						lineNumber++
					}
				}

				funcs = append(funcs, Func{lineNumber, funcName})
				break // Found the function name, move to the next match
			}
		}
	}
	return funcs
}
