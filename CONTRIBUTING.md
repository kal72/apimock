# Contribution and release guide

## Every change

1. Follow the project architecture and verify the affected behavior.
2. **Every feature, bug fix, and refactor must update CHANGELOG.md under Unreleased in the same change or PR.** Do not defer entries until release preparation.
3. Update ROADMAP.md when scope, status, limitations, or follow-up work changes. Add a row for new work and update the document date.
4. Update README.md when setup, configuration, usage, or application limitations change.
5. Describe completed checks and any verification gaps in the PR description.
6. **Write documentation and code comments in English.** This includes README content, roadmap entries, changelog entries, release notes, architecture notes, and comments in implementation and test code. Web UI labels, accessibility text, and user-facing runtime messages must also use English; preserve user-authored content.

Record significant documentation changes as well. Describe their impact on users and maintainers; avoid listing every modified file or mechanical edit.

## Writing changelog entries

Add concise bullets under the appropriate category within `## [Unreleased]`:

| Category | Purpose |
| --- | --- |
| Added | New features or capabilities. |
| Changed | Changes to behavior, defaults, APIs, or configuration. |
| Fixed | Bug fixes; explain the trigger and corrected behavior. |
| Refactored | Internal restructuring without intentional behavior changes. |
| Deprecated | Features still available but scheduled for retirement. |
| Removed | Removed features, APIs, or configuration. |
| Security | Security improvements or changes. |

If a change spans multiple categories, describe distinct aspects without repeating the same entry. Include a **Breaking change** notice and migration steps for incompatible changes. Document database migrations, environment variable changes, and flow format changes when applicable. Empty categories may be omitted.

Do not invent version numbers, release dates, PR links, or test results. Unfinished features belong in the roadmap; changelog entries describe available capabilities and their limitations.

Example entries for future changes (these are not implemented changes):

```markdown
### Fixed

- Preserve header changes when switching from Table to Node mode.

### Refactored

- Separate PostgreSQL error mapping from the repository without changing the management API contract.
```

## Verification

For Go changes, run `make check`. Set `TEST_DATABASE_URL` when changes affect persistence or migrations. For UI changes, run `make build` and inspect affected browser interactions. For container changes, verify both Docker builds and runtime behavior. Markdown-only changes require content, local-link, and status-consistency checks.

## Preparing a GitHub release

1. Review Unreleased, roadmap status, test results, and open limitations. Do not mark unfinished QA complete.
2. Choose a version based on actual changes: major for breaking changes, minor for compatible features, and patch for compatible fixes. During 0.x development, still explain every breaking change and migration step.
3. Move Unreleased entries into `## [X.Y.Z] - YYYY-MM-DD` using the actual release date. Leave an Unreleased section for subsequent work.
4. Preserve earlier release history. Add comparison or tag links only when the actual repository URL and tags are available.
5. Once release changes are approved, create the `vX.Y.Z` tag and GitHub Release within the authorized scope. Use the corresponding changelog section for release notes, including breaking changes, migrations, and relevant limitations.
6. Attach binaries or images only when the build and distribution pipeline is available and verified.

This document does not automatically create commits, tags, or GitHub Releases.
