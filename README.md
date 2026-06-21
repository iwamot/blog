# blog.iwamot.com

Source for [blog.iwamot.com](https://blog.iwamot.com/), built with [Hugo](https://gohugo.io/) and the [Hextra](https://github.com/imfing/hextra) theme and deployed to GitHub Pages.

## Local preview

```shell
hugo mod tidy   # first run / after updating the theme
hugo server -D  # http://localhost:1313/  (-D shows drafts)
```

## Writing an article

Articles are [page bundles](https://gohugo.io/content-management/page-bundles/) under `content/articles/`. Each one gets its own directory holding `index.md` and any images it references:

```
content/articles/<slug>/
├── index.md
└── diagram.png   # an image referenced from the article
```

### 1. Write `index.md`

```markdown
---
title: "Article title"
date: 2026-06-20
images: ["ogp.png"] # the OGP card; always this value
draft: true

# optional fields
tags: ["Hugo", "OGP"]
description: "A new article"
lastmod: 2026-06-21
---

Body.

![Diagram](diagram.png)
```

To include an image, drop the file next to `index.md` and reference it by filename (`![alt](diagram.png)`). Hugo publishes it at `/articles/<slug>/<filename>`.

### 2. Run `./validate.sh`

Runs the lint checks and, along the way, (re)generates each article's `ogp.png` — so a forgotten or stale card is built for you here.

### 3. Publish

Remove `draft: true` (or set it to `false`) and push to `main`. A `draft: true` article is shown only locally (`hugo server -D`) and is excluded from the production build.

## Deployment

Pushing to `main` triggers [`.github/workflows/pages.yaml`](.github/workflows/pages.yaml), which builds the site with Hugo and deploys it to GitHub Pages (custom domain `blog.iwamot.com`).

## Theme updates

The Hextra theme is a Hugo Module pinned in [`go.mod`](go.mod). Renovate watches it and opens a PR when a new version ships; merging that PR to `main` runs the Pages workflow and redeploys. No manual `hugo mod get` is needed.
