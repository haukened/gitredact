package rewriter

// DeletePath rewrites all commits reachable from branches (and optionally
// tags) by removing every tree entry whose repo-relative path equals target.
func DeletePath(root, target string, includeTags, silent bool) error {
	return run(root, includeTags, Options{
		Silent:     silent,
		PathFilter: func(p string) bool { return p != target },
	})
}
