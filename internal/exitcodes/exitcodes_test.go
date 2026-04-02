package exitcodes

import "testing"

func TestConstantValues(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"Success", Success, 0},
		{"InvalidUsage", InvalidUsage, 2},
		{"RepoValidation", RepoValidation, 3},
		{"DirtyWorktree", DirtyWorktree, 4},
		{"NoMatchesFound", NoMatchesFound, 5},
		{"RewriteExecution", RewriteExecution, 6},
		{"UserDeclined", UserDeclined, 7},
		{"VerificationFailed", VerificationFailed, 8},
		{"DependencyMissing", DependencyMissing, 9},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}
