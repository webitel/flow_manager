package tree

import (
	"encoding/binary"
	"encoding/json"
	"hash/fnv"
	"sort"
)

// hashSchema computes a deterministic uint64 hash of the schema.
//
// Strategy: re-serialise the schema with sorted keys (via normalise), then
// feed the bytes through FNV-1a 64-bit. This is fast and good enough for
// version pinning — collision probability is negligible for the schema counts
// we deal with.
func hashSchema(schema Schema) uint64 {
	b, err := json.Marshal(normalise(schema))
	if err != nil {
		// Should never happen with a valid schema; return 0 as a sentinel.
		return 0
	}
	h := fnv.New64a()
	_ = binary.Write(h, binary.LittleEndian, uint64(len(b)))
	h.Write(b)
	return h.Sum64()
}

// normalise deep-converts a schema into a representation whose JSON encoding
// is deterministic (object keys sorted, whitespace stripped by json.Marshal).
func normalise(v any) any {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys)*2)
		// Encode as [[key, value], ...] to preserve sorted order through
		// json.Marshal (Go maps have non-deterministic iteration).
		pairs := make([][2]any, len(keys))
		for i, k := range keys {
			pairs[i] = [2]any{k, normalise(val[k])}
		}
		_ = out
		return pairs
	case []any:
		out := make([]any, len(val))
		for i, el := range val {
			out[i] = normalise(el)
		}
		return out
	default:
		return val
	}
}
