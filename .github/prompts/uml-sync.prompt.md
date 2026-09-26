---
description: Bring the UML diagrams in docs/uml/ up to date with the Go code
agent: agent
---

<!-- Keep in step with .claude/commands/uml-sync.md (the Claude Code version of this prompt). -->

Update the Mermaid UML diagrams in `docs/uml/` so they match the current Go code. Follow the notation and conventions in `docs/uml/README.md`.

Base ref to compare against (optional): ${input:base:git ref, e.g. main; leave empty to use the Last checked commits}

1. **Choose the base ref.** Use the base ref above if given. Otherwise use the oldest commit named on the `Last checked against code:` lines of `docs/uml/*.md`, or `main` if none has one.
2. **Find the changed code.** Run `git diff --name-only <base> -- '*.go' ':!*_test.go'`. This compares with the working tree, so uncommitted changes are included. If nothing changed, say the diagrams are current and stop.
3. **List what changed.** For each changed file, read `git diff <base> -- <file>` and list the functions added, removed, renamed or changed. Include package-level values that drive a flow, such as `villageCodeLookup` or `teamWordNames`.
4. **Find the affected diagrams.** Look each item up in the *Source map* in `docs/uml/README.md`. Read every affected diagram file and the current code of every function it covers.
5. **Update the diagrams.** Change only what the code change affects: new or removed branches, guards, Discord calls, state changes and reply text. A change that doesn't alter a flow (a renamed local variable, debug output) needs no diagram change.
6. **Deal with code missing from the source map.** A new command, or a function that is now part of a command's flow, gets a diagram (in the file that fits, or in a new file added to the *Diagrams* table) and a source-map row. A trivial helper goes on the *Not diagrammed* list. If a function was removed or renamed, update or remove its rows.
7. **Update `Last checked against code:`** in every diagram file you reviewed: today's date and `git rev-parse --short HEAD`.
8. **Validate.** Render each file you changed. Either:
   - `docker run --rm -v "$PWD/docs/uml:/data" minlag/mermaid-cli -i /data/<file>.md -o /tmp/<file>.md` (on Git Bash for Windows, put `MSYS_NO_PATHCONV=1` in front), or
   - `npx -y @mermaid-js/mermaid-cli -i docs/uml/<file>.md -o <scratch dir>/<file>.md`.

   Fix any parse errors. Common causes: a node ID called `end`, double quotes or `<`/`>` inside a label, or a `;` in a sequence-diagram message.
9. **Keep the other docs in step.** If a command was added or removed, check that the README's Commands table, `CLAUDE.md` and `.github/copilot-instructions.md` agree.
10. **Report.** Say which diagrams changed and why, which rows were added to the source map, and anything you weren't sure about.
