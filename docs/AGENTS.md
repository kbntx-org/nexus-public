# Documentation Guidelines

## Philosophy

Docs live alongside the code and cover only what the code and the upstream docs don't: why we chose
something, how it is used in this repo, our conventions and gotchas, and the tasks a reader does.
Never restate upstream documentation; link to it.

The site is public: no admin-only tooling, and no hostnames (see
[Avoid Over-Specific Details](#avoid-over-specific-details)).

## Writing Principles

- **Useful and concise.** Every paragraph must help a reader do or understand something. Concise
  means fewer words, not fewer facts: cut tutorials, restatements of upstream docs, repetition and
  anything the code already answers, but keep every decision and gotcha.
- **Keep it true.** Check every statement against the code. When one is wrong, replace it with the
  correct fact rather than deleting it; when you're unsure, ask.
- **Justify decisions.** State what we chose, the alternative we rejected and why, in plain
  language.
- **Easy to read.** Short paragraphs, the outcome before the mechanics, and nothing said twice on a
  page or across pages: link to where it is explained instead.
- **Illustrate.** When a flow or an architecture takes more than a few sentences, draw it and keep
  only the prose the diagram can't carry. See [Diagrams](#diagrams).

## Structure

```text
docs/src/
  getting-started/       # what Nexus is and where to start
  platform/
    .nav.yml             # sidebar: tools grouped by domain (Compute, Network & access, ...)
    <tool>/
      .nav.yml           # title, then Design, How to and any extra pages
      design.md
      how-to.md
      <procedure>.md     # optional, see Longer procedures
```

The sidebar is grouped by **domain**; each entry is a **tool** (ArgoCD, Traefik, Vault). A small
companion doesn't get its own folder: it goes on the page of the tool a reader would open
(cert-manager and external-dns live on Traefik). Add a domain group only when a tool fits none of
the existing ones.

Each tool has two pages, one per reader question, plus optional
[procedure pages](#longer-procedures):

### Design page

`title: <Tool> design`. What it is and why we use it, in one or two sentences, then:

- `## How it's set up`: how it fits into the platform, usually a diagram with numbered steps under
  it.
- `## Decisions and gotchas`: bullets, each opening with the bold decision or gotcha, then the why.
- `## References`.

### How to page

`title: <Tool>`. One or two sentences on what the tool does for the reader and where its UI is, then
**tasks only**, each a `## <Verb> ...` heading (Add an app, Roll back, Debug a route) with the
commands or YAML to copy. Tasks every engineer does come first; platform-only tasks come last,
marked `(platform)`, or the intro says the whole page is platform work. End with
`How it works: [<Tool> design](design.md).` and `## References`.

A how-to explains no mechanism: it links to the section of the design page that does.

### Longer procedures

A procedure too long for a how-to section (restoring a database) gets its own page in the tool's
folder, listed in its `.nav.yml` after How to, and linked from the how-to. It covers when to use it,
prerequisites (access such as WARP), numbered steps with copy-pasteable commands using placeholders,
how to verify, and gotchas. One-off setup gets no page: put it in the PR description.

## Writing a Doc

### Diagrams

Diagrams are **SVG only, never Mermaid**: a hand-written SVG lays out zones, boundaries and mixed
line styles that Mermaid can't, and reads the same in both themes.

- Hand-write it and keep it tiny (a few KB at most): no editor metadata, no embedded fonts or raster
  images, a `viewBox` instead of fixed width/height, shapes and `<text>` only.
- Color it only with `currentColor` and Material CSS variables (`--md-default-fg-color--light`,
  `--md-accent-fg-color`, ...) in one `<style>` block, so it follows the light and dark themes.
- Put it in `docs/src/assets/diagrams/` and inline it with a snippet wrapped in a `div`:

  ```markdown
  <div class="nexus-diagram">
  --8<-- "src/assets/diagrams/<name>.svg"
  </div>
  ```

  An `<img>` can't inherit the page's theme colors, and without the `div` Markdown parses the SVG's
  text: a `*` or `_` in a label becomes emphasis and breaks the diagram. Prefix its class names and
  ids with the diagram name, since inlined SVGs share the page's CSS.

- Match the style of the existing diagrams in that folder, and optimize it with
  [SVGO](https://svgo.dev/) before committing:
  `pnpm dlx svgo --config docs/svgo.config.mjs docs/src/assets/diagrams/<name>.svg`. The config
  keeps the `<style>` block, class names and ids the default preset would inline or rename.

### Avoid Over-Specific Details

Do not hard-code specifics that can be found in the repository itself:

- **No version numbers**: exact versions live in `Chart.yaml`, `values.yaml`, `mise.toml`, etc.
- **No counts**: replicas, nodes or workers change; the code is the source of truth.
- **No exact CIDRs or IPs**: they belong in the OpenTofu configs.
- **No public hostnames** (`argocd.kbntx.com`, `vault.kbntx.com`, ...). Refer to services by name
  ("the ArgoCD UI") and link to the template that owns the hostname. In code samples, use
  placeholders (`my-app.example.com`).
- **No enumerations of what currently exists**: never list every app, chart or component of a kind.
  Describe the category instead ("one image per deploy target"); a single named example is fine.

These details age badly. Link to the file instead.

### Links

- **Inline-link a tool's official docs on its first mention** on its design page; the how-to links
  its design page instead.
- **Links outside `docs/src/` open in a new tab**, GitHub repo links included. Use a plain HTML
  anchor, not `attr_list` syntax, which oxfmt breaks when it wraps the line; wrap file paths and
  commands in `<code>`:

  ```markdown
  <a href="https://argo-cd.readthedocs.io/" target="_blank" rel="noopener">ArgoCD</a>
  <a href="https://github.com/kbntx-org/nexus/blob/main/Tiltfile" target="_blank" rel="noopener"><code>Tiltfile</code></a>
  ```

- **Links between doc pages stay plain Markdown** (`[Vault](../vault/how-to.md)`).

### References Section

**Every page ends with a `## References` section**: repo links (files, folders, charts, workflows) a
reader might open next, one bullet each with a short trailing description. Upstream docs worth
keeping at hand, when the prose hasn't already linked them, come after an `**External**:` line, one
bullet each.
