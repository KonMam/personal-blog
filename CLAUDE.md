# CLAUDE.md

## Project Overview

Personal blog at [mamonas.dev](https://mamonas.dev/) built with Hugo. Source code hosted on GitHub (`KonMam/personal-blog`), deployed via Cloudflare Pages from the `main` branch.

## Tech Stack

- **Hugo** v0.154.4 extended (darwin/arm64, Homebrew)
- **Theme:** Syl (custom theme, modified from re-terminal)
- **Hosting:** Cloudflare Pages
- **Analytics:** Google Analytics (`G-F0SL3JQ5SY`)

## Key Commands

```bash
hugo server              # Local dev server with live reload
hugo server -D           # Include draft posts
hugo new content posts/my-post-slug/index.md   # New post (page bundle)
hugo                     # Build to ./public
```

## Project Structure

```
├── hugo.toml              # Site configuration
├── content/
│   ├── about.md           # About page
│   └── posts/             # Blog posts (page bundles)
│       └── my-post/
│           ├── index.md   # Post content
│           ├── cover.png  # Cover image
│           └── *.png/mp4  # Post assets (images, videos)
├── layouts/
│   └── shortcodes/        # Project-level shortcode overrides
│       ├── image.html     # WebP-converting image shortcode
│       └── lazy-video.html # Lazy-loaded autoplay video
├── themes/Syl/            # Custom theme
│   ├── archetypes/posts.md
│   ├── assets/
│   │   ├── css/           # SCSS source files
│   │   │   ├── style.scss       # Entry point (imports all partials)
│   │   │   ├── variables.scss   # Dark theme CSS vars + breakpoints
│   │   │   ├── variables-light.scss
│   │   │   ├── color/           # Color variants (blue, orange, red, green, pink, paper)
│   │   │   ├── main.scss, post.scss, header.scss, ...
│   │   │   └── syntax.scss, prism.scss, code.scss
│   │   └── js/
│   │       ├── banner.js
│   │       └── menu.js
│   ├── layouts/
│   │   ├── _default/
│   │   │   ├── baseof.html   # Root template (head, body, container)
│   │   │   ├── index.html    # Homepage
│   │   │   ├── single.html   # Single post
│   │   │   ├── list.html     # List pages
│   │   │   ├── term.html     # Taxonomy term
│   │   │   └── terms.html    # Taxonomy list
│   │   ├── partials/         # header, footer, head, menu, cover, pagination, ...
│   │   └── shortcodes/       # Theme shortcodes (image, figure, code, prismjs)
│   └── 404.html
├── static/                # Static assets served as-is
├── data/                  # Hugo data files
├── i18n/                  # Translation strings
└── resources/             # Hugo processed resources cache
```

## Configuration (hugo.toml)

- **Theme color:** `blue` (options: orange, blue, red, green, pink, paper)
- **Menu:** Home (`/`), About (`/about`)
- **Markup:** Goldmark with `unsafe = true` (allows raw HTML in markdown), CSS-based syntax highlighting (`noClasses = false`)
- **Minification:** Enabled for output HTML and CSS

## Content Authoring

### Front Matter (TOML `+++` format)

```toml
+++
title = "My Post Title"
date = "2025-08-26T21:06:08+03:00"
tags = ["go", "generative art"]
description = """
Multi-line description used for SEO and post previews.
"""
# Optional fields:
# author = ""
# cover = ""
# coverCaption = ""
# keywords = ["", ""]
# showFullContent = false
# readingTime = false
# hideComments = false
# color = ""  # per-post color override
+++
```

### Posts as Page Bundles

Each post lives in its own directory under `content/posts/`:
```
content/posts/my-post-slug/
├── index.md    # Post content
├── cover.png   # Cover image (referenced in front matter)
└── *.png       # Images referenced via shortcodes
```

### Shortcodes

**`image`** (project-level override) — Converts to WebP, generates `<picture>` with fallback:
```
{{</* image src="my-image.png" alt="description" width="800" renderWidth="600" */>}}
```
- `src`: filename in the page bundle
- `width`: resize width for WebP conversion
- `renderWidth`: displayed width (defaults to `width`)

**`lazy-video`** — Lazy-loaded autoplay video:
```
{{</* lazy-video mp4="video.mp4" width="600" */>}}
```

## Theme Architecture (Syl)

### Template Hierarchy

`baseof.html` → defines the HTML shell with `{{ block "main" }}` and `{{ block "footer" }}`. Child templates (`index.html`, `single.html`, `list.html`) fill the `main` block.

### SCSS Structure

Entry point is `style.scss`, which conditionally imports `variables.scss` (dark) or `variables-light.scss` (paper theme), then all component partials. Color variants in `color/` override `--accent` via CSS custom properties. Breakpoints: `$phone: 684px`, `$tablet: 900px`.

### JS Assets

- `menu.js` — Mobile menu toggle
- `banner.js` — Dismissible banner

## Deployment

Code is on GitHub (`git@github.com:KonMam/personal-blog.git`). Deployed via Cloudflare Pages, building from the `main` branch. No CI config file in repo — Cloudflare Pages handles the build pipeline directly.

## Documentation Links

- [Hugo Documentation](https://gohugo.io/documentation/)
- [Hugo Templating](https://gohugo.io/templates/)
- [Cloudflare Pages](https://developers.cloudflare.com/pages/)
- [Sass/SCSS Reference](https://sass-lang.com/documentation/)
