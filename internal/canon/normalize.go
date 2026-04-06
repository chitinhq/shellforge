package canon

import (
	"strings"
)

// normalizeFlag converts a flag name to its canonical long form.
func normalizeFlag(tool, action, flag string) string {
	// Try tool.action:flag first (most specific).
	if action != "" {
		key := tool + "." + action + ":-" + flag
		if long, ok := flagAliases[key]; ok {
			return long
		}
		key = tool + "." + action + ":--" + flag
		if long, ok := flagAliases[key]; ok {
			return long
		}
	}

	// Try tool:flag.
	key := tool + ":-" + flag
	if long, ok := flagAliases[key]; ok {
		return long
	}
	key = tool + ":--" + flag
	if long, ok := flagAliases[key]; ok {
		return long
	}

	// Return as-is (already canonical or unknown).
	return flag
}

// flagTakesValue returns true if the given flag is known to require a value argument.
func flagTakesValue(tool, action, flag string) bool {
	valuedFlags := map[string]bool{
		// grep
		"grep:after-context":  true,
		"grep:before-context": true,
		"grep:context":        true,
		"grep:regexp":         true,
		"grep:A":              true,
		"grep:B":              true,
		"grep:C":              true,
		"grep:e":              true,
		"grep:max-count":      true,
		"grep:m":              true,

		// git log
		"git.log:max-count":  true,
		"git.log:n":          true,
		"git.log:format":     true,
		"git.log:pretty":     true,
		"git.log:since":      true,
		"git.log:until":      true,
		"git.log:author":     true,
		"git.log:grep":       true,

		// git commit
		"git.commit:m":       true,
		"git.commit:message": true,
		"git.commit:author":  true,

		// git diff
		"git.diff:stat": false,

		// read (head/tail)
		"read:lines": true,
		"read:n":     true,

		// curl
		"curl:output":  true,
		"curl:o":       true,
		"curl:request": true,
		"curl:X":       true,
		"curl:header":  true,
		"curl:H":       true,
		"curl:data":    true,
		"curl:d":       true,

		// find
		"find:name":     true,
		"find:type":     true,
		"find:maxdepth": true,

		// docker
		"docker:name":    true,
		"docker:format":  true,
		"docker:f":       true,
	}

	// Check with action specificity first.
	if action != "" {
		if v, ok := valuedFlags[tool+"."+action+":"+flag]; ok {
			return v
		}
	}
	if v, ok := valuedFlags[tool+":"+flag]; ok {
		return v
	}

	return false
}

// normalizeToolSpecific applies tool-aware normalizations that go beyond flag aliases.
func normalizeToolSpecific(tool, rawCmd, action string, flags map[string]string, args *[]string) {
	switch tool {
	case "read":
		normalizeRead(rawCmd, flags, args)
	case "grep":
		normalizeGrep(rawCmd, flags)
	case "git":
		normalizeGit(action, flags)
	}
}

// normalizeRead merges cat/head/tail into a unified "read" representation.
func normalizeRead(rawCmd string, flags map[string]string, args *[]string) {
	switch rawCmd {
	case "head":
		// head -N file → read --lines=N file (from start)
		if n, ok := flags["lines"]; ok {
			flags["head-lines"] = n
			delete(flags, "lines")
		} else if n, ok := flags["n"]; ok {
			flags["head-lines"] = n
			delete(flags, "n")
		}
	case "tail":
		// tail -N file → read --tail-lines=N file
		if n, ok := flags["lines"]; ok {
			flags["tail-lines"] = n
			delete(flags, "lines")
		} else if n, ok := flags["n"]; ok {
			flags["tail-lines"] = n
			delete(flags, "n")
		}
	}
	// cat with no special flags is just "read" — no extra normalization needed.
}

// normalizeGrep merges rg/ag/ack/grep into a unified representation.
// Canonical grep is always recursive — the -r flag is informational noise.
func normalizeGrep(rawCmd string, flags map[string]string) {
	// All grep variants: strip recursive flag since canonical grep is
	// inherently "search in files/dirs".
	delete(flags, "recursive")
	delete(flags, "r")
}

// normalizeGit applies git-specific flag normalizations.
func normalizeGit(action string, flags map[string]string) {
	if action == "log" {
		// --oneline is shorthand for --format=oneline --abbrev-commit
		if _, ok := flags["oneline"]; ok {
			flags["format"] = "oneline"
			delete(flags, "oneline")
			delete(flags, "abbrev-commit") // implied by oneline
		}
		// --pretty=X is equivalent to --format=X
		if v, ok := flags["pretty"]; ok {
			flags["format"] = v
			delete(flags, "pretty")
		}
		// Normalize format=oneline (from alias table)
		if v, ok := flags["format=oneline"]; ok && v == "" {
			flags["format"] = "oneline"
			delete(flags, "format=oneline")
		}
	}
}

// maskSensitive replaces values that look like secrets with "[MASKED]".
func maskSensitive(val string) string {
	upper := strings.ToUpper(val)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(upper, pattern) {
			return "[MASKED]"
		}
	}
	// Also mask things that look like API keys (long hex/base64 strings).
	if len(val) > 30 && !strings.Contains(val, " ") && !strings.Contains(val, "/") {
		allAlnum := true
		for _, ch := range val {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
				allAlnum = false
				break
			}
		}
		if allAlnum {
			return "[MASKED]"
		}
	}
	return val
}
