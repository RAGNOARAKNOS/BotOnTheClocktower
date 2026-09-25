# Copilot instructions

## Language: British English

Write all prose in British English. This covers code comments, doc comments, Markdown documentation and commit messages.

- Use `-ise` / `-isation`, not `-ize` / `-ization`: initialise, serialise, organise, recognise, normalisation.
- Use `-our`: behaviour, colour, favour, honour.
- Use `-re`: centre, metre.
- Double the final `l` before a suffix: cancelled, travelling, modelled, labelled.
- Other common forms: licence (noun) / license (verb), practice (noun) / practise (verb), defence, catalogue, grey, programme (a schedule; "program" for software).

**Don't change code to match.** Keep American spelling where it's part of code, not prose:

- Identifiers, and names from Go's standard library or third-party packages (e.g. `Color`, `Initialize`, `discordgo` types and methods)
- String literals, config keys, JSON fields, command names, and Discord role or channel names
- Quoted text from external sources, such as API documentation or error messages

## Commit messages

Commit messages follow [.github/commit-message-instructions.md](commit-message-instructions.md).
