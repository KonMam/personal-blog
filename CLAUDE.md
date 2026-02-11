# CLAUDE.md

## Project Overview

Personal blog at [mamonas.dev](https://mamonas.dev/) built with Hugo. Source code hosted on GitHub (`KonMam/personal-blog`), deployed via Cloudflare Pages from the `main` branch.

## Tech Stack

- **Hugo** v0.154.4 extended (darwin/arm64, Homebrew)
- **Theme:** Syl (custom theme, heavily modernized from re-terminal)
- **Font:** Inter (Google Fonts)
- **Hosting:** Cloudflare Pages
- **Analytics:** Cloudflare Web Analytics (server-side, no client JS)
- **Comments:** giscus (GitHub Discussions-backed)

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
│   ├── about.md           # About page (uses layout = "about")
│   └── posts/             # Blog posts (page bundles)
│       └── my-post/
│           ├── index.md   # Post content
│           └── *.png/mp4  # Post assets (images, videos)
├── layouts/
│   ├── about.html         # Custom About page layout
│   ├── _default/
│   │   └── terms.html     # Project-level taxonomy list override
│   └── shortcodes/        # Project-level shortcode overrides
│       ├── image.html     # WebP-converting image shortcode
│       └── lazy-video.html # Lazy-loaded autoplay video
├── themes/Syl/            # Custom theme
│   ├── archetypes/posts.md
│   ├── assets/
│   │   ├── css/           # SCSS source files (14 files, ~1100 lines total)
│   │   │   ├── style.scss       # Entry point (imports all partials)
│   │   │   ├── variables.scss   # Dark + light theme vars (data-theme), card vars, breakpoints
│   │   │   ├── color/           # Color variants (blue, orange, red, green, pink)
│   │   │   ├── main.scss        # Base styles, typography, blockquotes, tables, code, container
│   │   │   ├── post.scss        # Post title, meta, tags (pills), content links, cover
│   │   │   ├── cards.scss       # Homepage/listing card grid
│   │   │   ├── header.scss      # Header layout, inline nav, theme toggle
│   │   │   ├── menu.scss        # Mobile menu dropdown
│   │   │   ├── logo.scss        # Logo styling
│   │   │   ├── about.scss       # About page styles
│   │   │   ├── terms.scss       # Taxonomy list (pill cards)
│   │   │   ├── pagination.scss  # Post navigation
│   │   │   ├── footer.scss      # Footer (centered)
│   │   │   ├── buttons.scss     # Base button styles (used by pagination)
│   │   │   └── syntax.scss      # Chroma syntax highlighting (Hugo built-in)
│   │   └── js/
│   │       ├── menu.js          # Mobile menu toggle
│   │       └── theme-toggle.js  # Dark/light mode toggle with localStorage
│   ├── layouts/
│   │   ├── _default/
│   │   │   ├── baseof.html   # Root template (head, body, container, data-theme)
│   │   │   ├── index.html    # Homepage (card grid, no title)
│   │   │   ├── single.html   # Single post
│   │   │   ├── list.html     # List pages (card grid)
│   │   │   └── term.html     # Taxonomy term (card grid)
│   │   ├── partials/
│   │   │   ├── head.html     # Meta tags, FOUC prevention, fonts, OG tags, stylesheets
│   │   │   ├── header.html   # Logo + inline nav + theme toggle (single row)
│   │   │   ├── footer.html   # Copyright + social icons + JS bundle
│   │   │   ├── post-card.html # Card partial (badge, title, date, description — no covers)
│   │   │   ├── comments.html  # giscus integration with theme sync
│   │   │   └── cover, pagination, mobile-menu, logo
│   └── 404.html              # Modern 404 page (.not-found styles)
├── static/                # Static assets served as-is
├── data/                  # Hugo data files
├── i18n/                  # Translation strings
└── resources/             # Hugo processed resources cache
```

## Configuration (hugo.toml)

- **Theme color:** `blue` (accent: `#6C8CFF` muted indigo)
- **Taxonomies:** `tags` and `categories`
- **Menu:** Home (weight 1), About (weight 2), Categories (weight 3) — rendered inline in header
- **Markup:** Goldmark with `unsafe = true`, CSS-based syntax highlighting (`noClasses = false`)
- **Minification:** Enabled for output HTML and CSS

## Design System

### Dark/Light Mode

- Controlled via `data-theme` attribute on `<html>` (`dark` or `light`)
- FOUC prevention: inline `<script>` in `<head>` reads `localStorage('theme')` or `prefers-color-scheme` before paint
- Toggle button in header with sun/moon SVG icons
- `theme-toggle.js` handles click, localStorage persistence, and OS preference changes
- All colors use CSS custom properties defined in `variables.scss` under `:root` (dark) and `[data-theme="light"]`
- giscus comments theme syncs via `postMessage` API with a `MutationObserver`

### Color Palette

- **Accent:** `#6C8CFF` (muted indigo/periwinkle)
- **Dark background:** `#111118`
- **Dark text:** `#b8bcc8` (neutral cool gray)
- **Light background:** `#ffffff`
- **Light text:** `#1a1a2e` (dark navy)
- **Borders:** `rgba(255,255,255,0.08)` dark / `rgba(0,0,0,0.08)` light

### Typography

- **Font:** Inter (400, 500, 600, 700) via Google Fonts
- **Body:** 1rem, line-height 1.7
- **Headings:** h1 1.5rem → h2 1.25rem → h3 1.1rem → h4-h6 1rem
- **Logo:** 1.2rem bold
- **Inline code:** monospace, subtle `var(--border-color)` background, rounded corners
- **Code blocks:** subtle background fill, 1px border, `border-radius: 8px`

### Components

- **Cards:** used on homepage, list, and term pages. Background `var(--card-background)`, 1px border, 8px radius, hover lift + shadow. Text-only (no cover images).
- **Tags:** pill-style with rounded background, displayed on single post pages
- **Terms:** flex-wrap grid of pill cards with post count
- **Blockquotes:** `border-left: 3px solid var(--accent)`, no prompt symbol
- **Tables:** solid borders with `var(--border-color)`, no accent coloring on headers
- **Figcaptions:** italic, muted text — no background
- **Article links:** accent colored with subtle underline, stronger on hover
- **Post meta:** muted text with `·` (middle dot) separator
- **Pagination:** card-style buttons with hover lift

## Content Authoring

### Front Matter (TOML `+++` format)

```toml
+++
title = "My Post Title"
date = "2025-08-26T21:06:08+03:00"
tags = ["go", "generative art"]
categories = ["tech"]
description = """
Multi-line description used for SEO and post previews.
"""
# Optional fields:
# author = ""
# keywords = ["", ""]
# readingTime = false
# hideComments = false
# layout = "about"  # for custom layouts
+++
```

### Posts as Page Bundles

Each post lives in its own directory under `content/posts/`:
```
content/posts/my-post-slug/
├── index.md    # Post content
└── *.png       # Images referenced via shortcodes
```

### Shortcodes

**`image`** (project-level override) — Converts to WebP, generates `<picture>` with fallback:
```
{{</* image src="my-image.png" alt="description" width="800" renderWidth="600" */>}}
```

**`lazy-video`** — Lazy-loaded autoplay video:
```
{{</* lazy-video mp4="video.mp4" width="600" */>}}
```

## Deployment

Code is on GitHub (`git@github.com:KonMam/personal-blog.git`). Deployed via Cloudflare Pages, building from the `main` branch. No CI config file in repo — Cloudflare Pages handles the build pipeline directly.

## Documentation Links

- [Hugo Documentation](https://gohugo.io/documentation/)
- [Hugo Templating](https://gohugo.io/templates/)
- [Cloudflare Pages](https://developers.cloudflare.com/pages/)
- [Sass/SCSS Reference](https://sass-lang.com/documentation/)
- [giscus](https://giscus.app/) — GitHub Discussions-backed comments
