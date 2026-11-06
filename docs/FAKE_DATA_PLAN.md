# Fake data generator implementation plan

Date: 2026-11-06. Status: proposed; no fake-data generator or dependency is implemented by this document.

## Decision

Use `github.com/brianvoe/gofakeit/v7`, initially pinning `v7.9.0` after a compatibility spike. The official latest-release URL resolved to that version during research. The project is MIT licensed, its module targets Go 1.22, and it supports instance-based Faker methods with custom math/rand/v2 sources. Verify the selected tag's APIs, license and module metadata during implementation; master documentation may differ from a pinned release.

Keep Go as the only server. Add no Node runtime or external fake-data service. Existing Static JSON, request variables, built-in generators, first-match conditions, response headers and preview remain compatible.

Sources checked on 2026-11-05:

- [Official release](https://github.com/brianvoe/gofakeit/releases/tag/v7.9.0)
- [Official repository and function reference](https://github.com/brianvoe/gofakeit)
- [Module requirements](https://github.com/brianvoe/gofakeit/blob/master/go.mod)
- [License](https://github.com/brianvoe/gofakeit/blob/master/LICENSE.txt)
- [Alternative considered: go-faker/faker](https://github.com/go-faker/faker), oriented around filling Go structs. Our renderer needs an explicit typed function registry and isolated per-expression sources, so gofakeit is the proposed fit.

## Delivery phases

| Phase | Deliverable | Completion criteria | Status |
| --- | --- | --- | --- |
| FKG01 | Dependency/API spike | Selected version compiles with the project toolchain; instance/source APIs and catalog verified; seeded concurrency behavior checked. | Todo |
| FKG02 | Allowlisted registry and adapter | Generator names, argument schemas, result types and limits centralized; library isolated behind a use-case port. | Todo |
| FKG03 | Dynamic renderer integration | `generate.*` expressions validated and rendered with typed values; stable seed and cache behavior; Static JSON unchanged. | Todo |
| FKG04 | UI catalog and preview | Fake data source, searchable categories, bounded argument forms, sample preview and body/header insertion in the shared response component. | Todo |
| FKG05 | Verification and documentation | Unit/race/HTTP/persistence checks and Node/Table browser QA; usage guide/catalog, README, roadmap and changelog updated. | Todo |
| FKG06 | Named references and coherent profiles | Separate customer/seller references; consistent profile fields and derived email; repeated references share one profile. | Todo |
| FKG07 | Locale providers | Explicit supported-provider matrix; optional curated Indonesian datasets and coherent geographical records. | Todo |
| FKG08 | Arrays/datasets | Bounded list generation and per-item seed scope; designed separately from pagination/stateful storage. | Todo |

The first increment contains FKG01–FKG05. Do not advertise profiles, localization, lists or pagination as available until their phases pass validation.

## Initial catalog

Names below are the proposed public application API, not direct library function names. Each is an explicit adapter entry; do not expose arbitrary library function lookup or its general template engine.

| Category | Proposed expression name | JSON result | Provider / behavior |
| --- | --- | --- | --- |
| Person | `generate.person.first_name` | string | `FirstName()` |
| Person | `generate.person.last_name` | string | `LastName()` |
| Person | `generate.person.full_name` | string | `Name()` |
| Internet | `generate.internet.username` | string | `Username()` |
| Internet | `generate.internet.email` | string | Application wrapper: normalized generated username plus `@example.com`. |
| Internet | `generate.internet.url` | string | Application wrapper: generated slug on `https://example.com/`; not a live target. |
| Internet | `generate.internet.domain` | string | Application wrapper: generated label under `.example`. |
| Phone | `generate.phone.number` | string | `PhoneFormatted()`; provider format, no Indonesian/E.164 guarantee. |
| Address | `generate.address.street` | string | `Street()` |
| Address | `generate.address.city` | string | `City()` |
| Address | `generate.address.state` | string | `State()` |
| Address | `generate.address.postal_code` | string | `Zip()` |
| Address | `generate.address.country` | string | `Country()` |
| Address | `generate.address.country_code` | string | `CountryAbr()` |
| Company | `generate.company.name` | string | `Company()` |
| Company | `generate.company.job_title` | string | `JobTitle()` |
| Commerce | `generate.commerce.product_name` | string | `ProductName()` |
| Commerce | `generate.commerce.category` | string | `ProductCategory()` |
| Commerce | `generate.commerce.description` | string | `ProductDescription()` |
| Commerce | `generate.commerce.price` | number | `Price(min,max)` behind explicit application bounds and rounding. |
| Commerce | `generate.commerce.currency` | string | `CurrencyShort()`; not automatically tied to price or geography. |
| Text | `generate.text.word` | string | `LoremIpsumWord()` |
| Text | `generate.text.sentence` | string | `Sentence()`; wrapper checks output bounds. |
| Text | `generate.text.paragraph` | string | Bounded application composition from a small number of generated sentences. |
| Date | `generate.date.between` | string | `DateRange(start,end)`, explicit absolute bounds, UTC RFC3339 output. |

All generators use the shared `generate.*` namespace; no separate `fake.*` prefix or alias is planned.

Initial catalog: 25 additional generators. UUID, current timestamp, integer and choice remain available through existing `generate.*`; do not duplicate or change them.

Independent name/email/address expressions do not promise a consistent person or geographical address. The UI must state this. Use FKG06/FKG07 for linked profiles and location consistency. Provider datasets are not claimed to be complete English-US or Indonesian locale implementations.

## Syntax and arguments

Proposed no-argument expressions:

```json
{
  "name": "{{ generate.person.full_name }}",
  "email": "{{ generate.internet.email }}",
  "company": "{{ generate.company.name }}"
}
```

Parameterized expressions use a JSON options object:

```json
{
  "price": "{{ generate.commerce.price: {\"min\":10,\"max\":500,\"decimals\":2} }}",
  "created_at": "{{ generate.date.between: {\"from\":\"2025-01-01T00:00:00Z\",\"to\":\"2025-12-31T23:59:59Z\"} }}"
}
```

Proposed policies:

- Price defaults to min 1, max 1000, decimals 2. Bounds must be finite, nonnegative, ordered and at most 1,000,000,000. Decimal precision must be an integer from 0 to 4. Round deterministically within the accepted bounds; reject invalid options. This is a JSON number for testing, not a money/accounting engine.
- Date requires both RFC3339 bounds, from <= to, with a maximum 100-year span. It must not derive an implicit bound from the current clock; a fixed seed and unchanged options must remain reproducible.
- No-argument generators reject options. Parameterized generators reject unknown keys and wrong types.
- Keep the current 200-expression, nesting, response-size, header and missing-variable policies. Add a per-generated-string cap of 4096 UTF-8 bytes; handle limits with descriptive errors, not silent truncation that could corrupt output formats.
- Fallback defaults remain request-variable syntax; fake generators reject `| default:`.
- Whole-value bindings preserve JSON types; sentence/header bindings produce text. Generated data are not recursively parsed as templates.

The proposed syntax is not runnable in the current app until FKG03 is implemented.

## Architecture

1. **Domain:** extend the existing `generate.*` expression namespace with categorized names and typed argument validation. Keep third-party imports out of domain code. Generator descriptors define public name, category, result type and accepted options.
2. **Use case:** introduce a `FakeDataProvider` port and render context containing seed bytes, canonical expression identity, generator schema version and per-render cache. Extend request evaluation without global mutable Faker state.
3. **Adapter:** implement the port with gofakeit. Instantiate Faker with the existing deterministic per-expression ChaCha8 seed strategy; no global `Seed()` calls. Wrapper generators implement reserved example domains, formatting and application limits.
4. **HTTP/composition:** inject the adapter into real execution and Playground/response-preview execution. Both paths must use the same renderer; avoid a second browser fake-data implementation. Expose only generator metadata needed by the UI.
5. **UI:** add Fake data to Insert variable, category/search, generator, applicable options, result-type hint, sample generation, and existing destination controls. Reuse the templ response component for Node/Table/default response editing.
6. **Persistence:** store expressions in existing body/header configuration and reuse response seed. Pin a generator schema version for fake-enabled configurations so library upgrades cannot silently change the seeded contract. Missing version maps to the first supported schema after rollout; reject unknown versions. No separate dataset table is needed for scalar generators.

## Seed and reference contract

- Reuse existing response-seed behavior: absent seed gets fresh random seed bytes per render; a fixed seed yields deterministic fake values for the same generator schema, library version and options.
- Derive each source from response seed + namespace + canonical name + validated/defaulted options. JSON key order must not affect results.
- Cache equivalent expressions per render, shared between body and headers. The same full-name expression yields the same name in that render; scalar generators do not guarantee uniqueness.
- Do not guarantee unchanged outputs across a dependency/dataset upgrade. Pin the library and require a reviewed schema-version/migration policy before changing seeded results.
- FKG06 proposes `{"ref":"customer"}` and `{"ref":"seller"}` option scopes. A profile is generated once per reference; its first/last/full name and derived email agree. References are local to the response, not persistent user records. Final reference syntax must be specified before that phase is implemented.
- FKG08 must add item-index seed scope so identical expressions can generate different items while remaining reproducible. Do not reinterpret existing scalar cache behavior silently.

## Locale and data boundaries

No verified general locale-switching contract is assumed for the selected library. Do not offer an `id_ID` toggle based only on language labels. For Indonesian support, evaluate curated, licensed datasets and formatters for names, telephone examples, provinces/cities/postal codes and address relationships; document coverage explicitly.

Do not expose identity documents, payment-card/CVV, bank-account, private-key, password or authentication-token generators in the initial catalog. No requests to remote services, real contact validation, external image downloads, or executable user functions are needed.

## Verification and acceptance

- Existing Static JSON and `generate.*` results/semantics remain compatible.
- Every catalog entry has validation, result-type and output-bound tests.
- Same seed/name/options reproduces values across key ordering, body/header placement, concurrent requests and process restart on the pinned version.
- Missing seed does not inadvertently use a shared global source. No uniqueness promise is made for finite datasets.
- Arguments with malformed JSON, unknown keys, unsupported names and oversized strings fail clearly.
- Arbitrary methods, library templates, script evaluation, newline header injection and recursive expansion remain unavailable.
- HTTP tests cover real `/public/...` execution and Playground parity; repository tests cover any schema-version field round trip.
- Browser checks cover Fake data insertion, sample/response preview, saving/reloading, Node/Table and Static mode isolation in Light/Night.
- Update docs/DYNAMIC_JSON.md with runnable examples only after implementation; this plan stays explicitly proposed until then.

