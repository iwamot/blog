# blog.iwamot.com

Source for [blog.iwamot.com](https://blog.iwamot.com/), built with [Hugo](https://gohugo.io/) and the [Hextra](https://github.com/imfing/hextra) theme and deployed to GitHub Pages.

## Local preview

```shell
hugo mod tidy   # first run / after updating the theme
hugo server -D  # http://localhost:1313/  (-D shows drafts)
```

## Writing an article

Add a Markdown file under `content/articles/`:

```markdown
---
title: "Article title"
date: 2026-06-21
draft: true
---

Body.
```

- The URL is `/articles/<filename>/`. Override it with the `slug:` front matter field.
- A `draft: true` article is shown only locally (`hugo server -D`) and is excluded from the production build. Set `draft: false` and push to publish.

## Deployment

Pushing to `main` triggers [`.github/workflows/pages.yaml`](.github/workflows/pages.yaml), which builds the site with Hugo and deploys it to GitHub Pages (custom domain `blog.iwamot.com`).

## Update theme

```shell
hugo mod get -u
hugo mod tidy
```
