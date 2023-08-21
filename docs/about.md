

## Formatting
- [Markdown syntax](https://www.markdownguide.org/basic-syntax/)
- [Material for mkdocs](https://squidfunk.github.io/mkdocs-material/reference/abbreviations/)

## Contents
If you want the contents of a `.md` file appear on the right hand-side menu, add `##` sections.

## Working from another project
The project git repo must be supported by this project.

Its documentation must be inside `docs` folder and contain `.md` markdown files.

Optionally project can define `nav.yml` navigation file which will be included during this docs build and
visible on the left hand-side menu.

Sample `nav.yml`:
```yaml
Backend:
  - Overview: index.md
  - Cart: cart.md
```
