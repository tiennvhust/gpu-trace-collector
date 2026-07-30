#!/usr/bin/env bash
# STUDY-ONLY-FILE — this tool belongs to the learning branch only, and marking it
# also stops it from stripping itself (its awk patterns contain the markers).
# strip-study.sh — generate the public version of a file (or tree) from the
# learning version, by removing the study layer.
#
# This script exists so the two branches cannot drift. Write once, on `learning`,
# with the study material clearly marked; generate `main` mechanically.
#
# What it removes:
#   1. Teaching comments: any line whose first non-space content is `// »`
#      (or `# »` in YAML/Makefiles, `<!-- » ... -->` is not used).
#   2. Exercise blocks: everything from a line containing EXERCISE-BEGIN to a
#      line containing EXERCISE-END, inclusive.
#   3. Study-only markdown: from STUDY-ONLY-BEGIN to STUDY-ONLY-END, inclusive.
#   4. Whole study-only files: any file whose first 3 lines contain
#      STUDY-ONLY-FILE is skipped entirely.
#   5. Dangling comment separators: a `//`-only or `#`-only line left behind when
#      the teaching text after it was removed.
#   6. Runs of blank lines, collapsed to one.
#
# What it deliberately does NOT do: rewrite the surviving comments. A one-line
# `// TODO(weekN): ...` is written above every stub precisely so it survives the
# strip and the public version still reads as honest work-in-progress.
#
# Usage:
#   scripts/strip-study.sh FILE                 # write stripped file to stdout
#   scripts/strip-study.sh -o OUTDIR DIR        # strip a whole tree into OUTDIR
#   scripts/strip-study.sh --check FILE         # exit 1 if anything would change
#
# After stripping Go files, run `gofmt -w` on the output: removing lines can leave
# comment blocks that gofmt would reindent.

set -euo pipefail

usage() {
	sed -n '2,32p' "$0" | sed 's|^# \{0,1\}||'
	exit "${1:-1}"
}

is_study_only_file() {
	head -n 3 "$1" 2>/dev/null | grep -q 'STUDY-ONLY-FILE'
}

strip_file() {
	awk '
		# Exercise and study-only blocks: drop everything between the markers.
		/EXERCISE-BEGIN/  { skip = 1 }
		/STUDY-ONLY-BEGIN/ { skip = 1 }
		/EXERCISE-END/    { skip = 0; next }
		/STUDY-ONLY-END/  { skip = 0; next }
		skip { next }

		# Teaching comments, in Go/C-style and hash-style files alike.
		/^[[:space:]]*\/\/ »/ { next }
		/^[[:space:]]*# »/    { next }
		/^[[:space:]]*\/\/ *»$/ { next }
		/^[[:space:]]*# *»$/    { next }

		{ print }
	' "$1" | awk '
		# Second pass: drop comment-separator lines that are now dangling — a bare
		# `//` or `#` whose next SURVIVING line is not itself a comment. This is what
		# keeps a doc comment from ending in a stray `//` after its teaching
		# paragraph was removed.
		#
		# Done backwards, so a run of several separators collapses in one go: once
		# the last `//` is marked dead, the one above it sees the non-comment line
		# beyond it and is marked too.
		{ lines[NR] = $0 }
		END {
			for (i = NR; i >= 1; i--) {
				line = lines[i]
				if (line ~ /^[[:space:]]*\/\/$/ || line ~ /^[[:space:]]*#$/) {
					# nextKept was set by the iteration for i+1.
					if (nextKept !~ /^[[:space:]]*\/\// && nextKept !~ /^[[:space:]]*#/) {
						dead[i] = 1
						continue        # nextKept unchanged: this line is gone
					}
				}
				nextKept = line
			}
			for (i = 1; i <= NR; i++) {
				if (dead[i]) continue
				line = lines[i]
				# Collapse runs of blank lines to one.
				if (line ~ /^[[:space:]]*$/) {
					if (blank) continue
					blank = 1
				} else {
					blank = 0
				}
				# Collapse consecutive bare comment separators inside one comment
				# block: removing a teaching paragraph from the middle of a doc
				# comment leaves `//` above and below it.
				bare = (line ~ /^[[:space:]]*\/\/$/ || line ~ /^[[:space:]]*#$/)
				if (bare && lastBare) continue
				lastBare = bare
				print line
			}
		}
	' | sed -E '
		# Third pass: inline study-only PHRASES, marked «like this». Used for
		# cross-references to exercise numbers inside comments and prose that must
		# survive the strip — "see prio3« EXERCISE 30»" reads correctly in both
		# versions. Anything between the guillemets goes.
		#
		# CONVENTION: a marker must open and close on ONE line (this is a per-line
		# substitution), and it absorbs its own leading space and punctuation so no
		# double space or orphaned dash is left behind. `--check` reports any marker
		# that is left unbalanced.
		s/«[^»]*»//g

		# Exercise identifiers inside string literals become plain TODOs, so the
		# public stubs still say what is missing.
		s/"EXERCISE [0-9]+: /"TODO: /g
		s/"EXERCISE [0-9]+"/"not implemented"/g

		# A trailing teaching comment on a line of code keeps its content and loses
		# its marker: `x = 1 // » why` becomes `x = 1 // why`.
		s/([^[:space:]])[[:space:]]*\/\/ » /\1 \/\/ /
		s/([^[:space:]])[[:space:]]*# » /\1 # /

		# Restore the full stop when a marker swallowed the end of the sentence,
		# e.g. "TODO(week1): implement« — see EXERCISE 1 in LaplaceScale.»".
		s/(TODO\(week[0-9]+[^)]*\): [a-z][A-Za-z ]*[a-z])$/\1./

		# Only trailing whitespace is tidied. Do NOT squeeze runs of spaces: Go
		# indentation is tabs but gofmt aligns struct fields and const blocks with
		# spaces, and markdown tables depend on their padding. The «» convention
		# absorbs its own leading space and punctuation instead — write
		# "see prio3« EXERCISE 30»", not "see prio3 «EXERCISE 30»".
		s/[[:space:]]+$//
	'
}

outdir=""
check=0
args=()
while [[ $# -gt 0 ]]; do
	case "$1" in
		-o) outdir="$2"; shift 2 ;;
		--check) check=1; shift ;;
		-h|--help) usage 0 ;;
		*) args+=("$1"); shift ;;
	esac
done
[[ ${#args[@]} -eq 1 ]] || usage

target="${args[0]}"

if [[ -f "$target" ]]; then
	if [[ $check -eq 1 ]]; then
		# Unbalanced inline markers would leak study text into the public version.
		if grep -n '«[^»]*$\|^[^«]*»' "$target" | grep -q .; then
			echo "$target: inline « » marker spans lines or is unbalanced" >&2
			grep -n '«[^»]*$\|^[^«]*»' "$target" >&2
			exit 1
		fi
		if diff -q <(strip_file "$target") "$target" >/dev/null; then
			echo "no study content in $target"
		else
			echo "study content present in $target"
			exit 1
		fi
	else
		strip_file "$target"
	fi
	exit 0
fi

[[ -d "$target" ]] || { echo "no such file or directory: $target" >&2; exit 1; }
[[ -n "$outdir" ]] || { echo "stripping a directory requires -o OUTDIR" >&2; exit 1; }

# Only text we actually annotate. Binary and generated files are copied verbatim.
find "$target" \
	-type d \( -name .git -o -name bin -o -name node_modules \) -prune -o \
	-type f -print |
while read -r f; do
	rel="${f#"$target"/}"
	dest="$outdir/$rel"
	mkdir -p "$(dirname "$dest")"

	if is_study_only_file "$f"; then
		echo "skip (study-only file): $rel" >&2
		continue
	fi

	case "$f" in
		*.go|*.md|*.yaml|*.yml|*.sh|Makefile|Dockerfile|*/Makefile|*/Dockerfile)
			strip_file "$f" > "$dest"
			;;
		*)
			cp "$f" "$dest"
			;;
	esac
done

echo "stripped $target into $outdir" >&2
echo "now run: gofmt -l -w $outdir" >&2
