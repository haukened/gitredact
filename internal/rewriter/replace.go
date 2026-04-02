package rewriter

import "bytes"

// Replace rewrites all commits reachable from branches (and optionally tags)
// by replacing every occurrence of from with to in blob content.
func Replace(root, from, to string, includeTags, silent bool) error {
	fromBytes := []byte(from)
	toBytes := []byte(to)
	return run(root, includeTags, Options{
		Silent: silent,
		BlobMapper: func(_ string, content []byte) ([]byte, bool) {
			out := bytes.ReplaceAll(content, fromBytes, toBytes)
			return out, !bytes.Equal(out, content)
		},
	})
}
