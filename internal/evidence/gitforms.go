package evidence

import "strings"

// gitSubcommand splits a git argument list into its subcommand and the
// arguments after it, skipping only the plainly harmless global options
// --no-pager, -P and --no-optional-locks. -C is refused: it would select a
// repository other than the approved one. Any other
// leading option, or no subcommand, fails closed.
func gitSubcommand(args []string) (string, []string, bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--no-pager" || a == "-P" || a == "--no-optional-locks":
		case strings.HasPrefix(a, "-"):
			return "", nil, false
		default:
			return a, args[i+1:], a != ""
		}
	}
	return "", nil, false
}

// gitForm is one approved git check form: the options it may use (exact
// spellings only, so no abbreviation can stand for another option) and the
// options it must carry. Arguments that do not start with "-" are revisions
// or paths; everything after "--" is a path.
type gitForm struct {
	allowed  map[string]bool
	required []string
}

func gitOpts(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

// patch subcommands print patches, so they must say --no-ext-diff and
// --no-textconv: the recorded argv then shows that no configured diff
// helper can start.
var gitForms = map[string]gitForm{
	"status":     {allowed: gitOpts("-s", "--short", "-b", "--branch", "--porcelain", "--porcelain=v1", "--porcelain=v2", "-uno", "--untracked-files=no")},
	"diff":       {allowed: gitOpts("--no-ext-diff", "--no-textconv", "--stat", "--quiet", "--exit-code", "--name-only", "--name-status", "--numstat", "--shortstat", "--check", "--cached", "--staged", "--no-color", "--summary"), required: []string{"--no-ext-diff", "--no-textconv"}},
	"log":        {allowed: gitOpts("--no-ext-diff", "--no-textconv", "--oneline", "--no-color", "--name-only", "--name-status", "--stat"), required: []string{"--no-ext-diff", "--no-textconv"}},
	"show":       {allowed: gitOpts("--no-ext-diff", "--no-textconv", "--oneline", "--no-color", "--name-only", "--name-status", "--stat", "-s", "--quiet"), required: []string{"--no-ext-diff", "--no-textconv"}},
	"rev-parse":  {allowed: gitOpts("--git-dir", "--is-inside-work-tree", "--verify", "--abbrev-ref", "--short", "--quiet", "-q", "--show-toplevel", "--is-bare-repository", "--is-shallow-repository")},
	"ls-files":   {allowed: gitOpts("--cached", "-c", "--others", "-o", "--exclude-standard", "--modified", "-m", "--deleted", "-d", "--error-unmatch", "-z")},
	"show-ref":   {allowed: gitOpts("--verify", "--quiet", "-q", "--hash", "--head", "--heads", "--tags")},
	"merge-base": {allowed: gitOpts("--is-ancestor")},
	"cat-file":   {allowed: gitOpts("-e", "-t", "-s", "-p")},
	"version":    {allowed: gitOpts()},
}

// GitFormAllowed reports whether a git argument list is one of the approved
// check forms. Everything else, including aliases, mutating subcommands and
// options that start a program, is refused.
func GitFormAllowed(args []string) bool {
	sub, rest, ok := gitSubcommand(args)
	if !ok {
		return false
	}
	form, listed := gitForms[sub]
	if !listed {
		return false
	}
	seen := map[string]bool{}
	for _, a := range rest {
		if a == "--" {
			break
		}
		if a == "" {
			return false
		}
		if strings.HasPrefix(a, "-") {
			if !form.allowed[a] {
				return false
			}
			seen[a] = true
		}
	}
	for _, r := range form.required {
		if !seen[r] {
			return false
		}
	}
	return true
}
