package canon

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// Digest computes a deterministic hash of a Command's canonical form.
// Two semantically equivalent commands produce the same digest.
func Digest(cmd Command) string {
	h := sha256.New()

	h.Write([]byte(cmd.Tool))
	h.Write([]byte("|"))
	h.Write([]byte(cmd.Action))
	h.Write([]byte("|"))

	// Sorted flags for deterministic output.
	keys := make([]string, 0, len(cmd.Flags))
	for k := range cmd.Flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte("="))
		h.Write([]byte(cmd.Flags[k]))
		h.Write([]byte(";"))
	}

	h.Write([]byte("|"))
	h.Write([]byte(strings.Join(cmd.Args, "\x00")))

	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// PipelineDigest computes a digest for an entire pipeline.
func PipelineDigest(p Pipeline) string {
	h := sha256.New()
	for _, seg := range p.Segments {
		h.Write([]byte(string(seg.Op)))
		h.Write([]byte(":"))
		h.Write([]byte(seg.Command.Digest))
		h.Write([]byte("|"))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
