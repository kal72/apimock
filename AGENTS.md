# Mocklab project instructions

- Prefix every shell command with `rtk`; use `rtk proxy` when raw output is required.
- Follow the clean architecture described in README.md. Go is the only application server; npm/Vite are used only for dependencies and asset builds.
- Read CONTRIBUTING.md before changing the project.
- Write all project documentation and code comments in English, including release notes, architecture notes, and comments added during tests or refactoring. The web UI and user-facing runtime messages must also use English. Preserve user-authored workspace names, endpoint labels, and response content.
- Every feature, bug fix, and refactor must update the Unreleased section of CHANGELOG.md in the same change. Explain its impact and any breaking changes or migrations.
- Update ROADMAP.md and its last-updated date when related scope or status changes. Use Partial for incomplete work; do not mark browser QA or releases complete without verification.
- Update README.md when setup, configuration, usage, or limitations change.
- Do not invent release versions, release dates, PR links, or test results. Do not create tags or publish GitHub Releases unless requested.
