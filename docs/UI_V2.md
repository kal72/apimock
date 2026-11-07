# V2 design preview

V2 is an isolated, static Graphite Blue + Cloudline design sandbox. It is served by the existing Go server and requires the same management login as the current application.

Start the application normally and open `/app/v2`. The current `/app` editor remains available.

| Route | Template |
| --- | --- |
| `/app/v2/endpoints` | Node canvas, Table alternative, condition inspector and sample playground |
| `/app/v2/environments` | Environment selection, variable list and edit form |
| `/app/v2/mocks` | Workspace summary and mock catalog |
| `/app/v2/logs` | Request list, execution trace and formatted response |
| `/app/v2/settings` | General, Members and Mock defaults tabs |

## Preview behavior

- All workspace names, endpoints, members, counts and requests are fictional fixtures embedded in the templates.
- No application API, repository, session mutation, mock execution, or persistence is called by the preview UI. Login protection is inherited from management routes.
- Light/Night switching, Node/Table switching, bounded canvas zoom (50–150%), inspector dismissal, environment selection, settings tabs and a fixed playground result work locally in the page.
- Search, creation, copying and saving controls are visual placeholders with explanatory feedback where appropriate. Deletion is disabled. The sample form does not save edits. Navigation resets preview state.
- Node dragging, connection editing, real filtering, request replay and live logs are not implemented. The canvas can scroll to reveal nodes at smaller widths.
- Environments, log retention, mock enable/disable and other proposed capabilities shown in these designs do not imply backend support.

## Header design

The shared header follows the first Graphite Blue mockup: a compact breadcrumb row with an illustrative Saved status, Light/Night switch, account popover and sky decoration at the far right. The endpoint list sits inside the shared sidebar to avoid a competing navigation column. Endpoints uses a second toolbar for its name, method/path, base URL, URL action and Node/Table switch. The V2/sample-data label remains in the brand/footer. Theme and account interactions are local preview state.

## Components and migration

`internal/web/v2.templ` contains `V2Page`, `V2Sidebar`, `V2Header`, `V2Title`, `V2Node` and page-specific components. `web/src/v2.css` owns scoped `.v2` tokens, surfaces and responsive rules; it does not change current editor selectors. Assets are built by Vite and embedded in the existing Go binary.

To migrate later, introduce presentation view models for each page, replace fixtures with authorized application data, bind actions to existing use cases, and test permissions, validation and persistence. Keep node evaluation semantics and flow storage separate from this visual layer. Establish actual product scope before implementing the proposed Environments/Logs capabilities.

These are runnable HTML/templ mockups rather than exact bitmap reproductions. Mobile and comprehensive accessibility QA remain incomplete.

## Verification

`make build` and `make check` passed (67 JavaScript tests, Go race tests and vet; PostgreSQL-specific integration tests were not enabled for this template-only change). The Compose build and startup passed. Desktop browser checks covered all five routes, Light/Night switching, condition inspector open/close, Node/Table switching, sample execution, environment selection and Settings tabs. Unauthenticated HTTP checks returned login redirects for all five pages. Screenshots: [Light](img/v2-node-light.jpg) and [Night](img/v2-node-night.jpg).

Header refinement verification: standalone and Compose builds passed, with desktop browser checks for the compact Light/Night header, unified endpoint sidebar and account popover open/close. `make check` passed during the header revision. Latest screenshots: [Header Light](img/v2-header-light.jpg) and [Header Night](img/v2-header-night.jpg). Full mobile/accessibility coverage remains incomplete.
