# Blog Server

A lightweight Go server that serves blog content from a directory of Markdown files. It exposes a JSON REST API for topics and articles, converts Markdown to HTML on the fly, and can automatically sync content from a remote Git repository.

## How it works

Content is organised into **topics** (directories) and **articles** (Markdown files within those directories). On startup the server reads the content directory, builds an in-memory index, and serves it over HTTP. If a remote Git repository is configured, the server will clone it on startup and periodically pull updates.

### Content structure

```
content/
├── my-topic/
│   ├── README.md        ← topic file
│   ├── first-article.md
│   └── second-article.md
└── another-topic/
    ├── README.md
    └── some-article.md
```

### File headers

Each Markdown file must include a header block with metadata. Two formats are supported:

**HTML comment block** (placed at the very top of the file):
```markdown
<!--
title: My Article
slug: my-article
description: A short description
published: 2024-01-15
updated: 2024-03-01
hidden: false
priority: 10
image: cover.png
-->

Article content here...
```

**YAML frontmatter**:
```markdown
---
title: My Article
slug: my-article
---

Article content here...
```

| Header | Description |
|--------|-------------|
| `title` | Display title. Falls back to the filename if omitted. |
| `slug` | URL slug. Falls back to the filename if omitted. |
| `description` | Short summary. |
| `published` | Publish date (`YYYY-MM-DD`). Articles without this are not returned as published. |
| `updated` | Last updated date (`YYYY-MM-DD`). |
| `hidden` | Set to `true` to hide from listings. |
| `priority` | Integer used for ordering. Higher values rank first. |
| `image` | Image filename, served from the asset directory. |
| `series` | Name of the series this article belongs to. See [Series](#series). |

Any unrecognised headers are stored as freeform `metadata` and included in API responses.

### Series

Articles can be grouped into a series by giving them the same `series` header:

```markdown
<!--
title: Building a Blog Server, Part 2
series: building-a-blog-server
published: 2024-02-01
-->
```

Series are global — articles from different topics can belong to the same series. Names are matched case-insensitively.

When fetching a single article that belongs to a series, the response includes a `series` object listing every article in that series (including the requested one), ordered by published date (oldest first, undated articles last). Unpublished articles are included with `"published": false` so upcoming parts can be surfaced (e.g. greyed out) in a UI; hidden articles are excluded entirely.

```json
{
  "title": "Building a Blog Server, Part 2",
  "series": {
    "name": "building-a-blog-server",
    "articles": [
      { "title": "Part 1", "slug": "part-1", "topicSlug": "go", "url": "/topics/go/articles/part-1", "publishedAt": 1704067200, "published": true },
      { "title": "Part 2", "slug": "part-2", "topicSlug": "go", "url": "/topics/go/articles/part-2", "publishedAt": 1706745600, "published": true },
      { "title": "Part 3", "slug": "part-3", "topicSlug": "go", "url": "/topics/go/articles/part-3", "publishedAt": 0, "published": false }
    ]
  }
}
```

Articles without a `series` header omit the field entirely.

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status` | Health check. Returns readiness and last indexed time. |
| `GET` | `/overview` | Returns the overview file rendered as HTML. |
| `GET` | `/recent?limit=N` | Returns the N most recently published articles (default: 3). |
| `GET` | `/topics` | Lists all topics. |
| `GET` | `/topics/{topic}` | Returns a single topic with its content rendered as HTML. |
| `GET` | `/topics/{topic}/articles` | Lists all articles for a topic. |
| `GET` | `/topics/{topic}/articles/{article}` | Returns a single article with its content rendered as HTML. |

Static assets are served at `/{CONTENT_ASSET_DIR}/`.

## Configuration

All configuration is via environment variables.

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Port to listen on. |
| `ALLOWED_ORIGINS` | _(none)_ | Comma-separated list of allowed CORS origins. |
| `ENVIRONMENT` | `development` | Environment name. |
| `STAGING_MODE` | `false` | When `true`, unpublished (future-dated or undated) articles are included in the `/recent` feed, so drafts can be previewed before going live. Hidden articles are always excluded. Run a separate instance with this enabled to use as a staging preview. |

### Content

| Variable | Default | Description |
|----------|---------|-------------|
| `CONTENT_PATH` | `./content` | Path to the content directory. If `CONTENT_REPO` is set, the repo is cloned here. |
| `CONTENT_REPO` | _(none)_ | Remote Git repository URL to clone and sync content from. |
| `CONTENT_UPDATE_INTERVAL_SECONDS` | `300` | How often to pull updates from the remote repo. |
| `CONTENT_ASSET_DIR` | `images` | Subdirectory within `CONTENT_PATH` that holds static assets. |
| `STATIC_ASSET_URL` | `images` | URL prefix used when rewriting image links in content. |
| `TOPIC_FILE` | `README.md` | Filename used to identify a topic within a directory. |

### Cache invalidation

When content changes, the server can notify an external site to purge its cache. Both variables must be set for invalidation to be enabled.

| Variable | Default | Description |
|----------|---------|-------------|
| `BLOG_SITE_HOST` | _(none)_ | Base URL of the site to notify (e.g. `https://example.com`). |
| `BLOG_SITE_SECRET` | _(none)_ | Secret token passed in the revalidation request. |

### Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `INFO` | Log level: `DEBUG`, `INFO`, `WARN`, or `ERROR`. |
| `LOG_FORMAT` | `json` | Log format: `json` or `text`. |

### Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `SENTRY_DSN` | _(none)_ | Sentry DSN for error reporting. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _(none)_ | OTLP gRPC endpoint for traces and metrics (e.g. `http://alloy:4317`). Telemetry is disabled if unset. |
| `OTEL_SERVICE_NAME` | `blog-server` | Service name attached to all telemetry. |

When `OTEL_EXPORTER_OTLP_ENDPOINT` is set, the server exports both traces and metrics (including host and Go runtime metrics) to the configured OTLP gRPC endpoint.

## Running locally

```bash
go run ./cmd/server
```

With a local content directory:

```bash
CONTENT_PATH=./my-content go run ./cmd/server
```

With a remote repository:

```bash
CONTENT_REPO=https://github.com/you/your-content.git \
CONTENT_PATH=/tmp/content \
go run ./cmd/server
```

## Docker

Build and run with Docker:

```bash
docker build -t blog-server .
docker run -p 3000:3000 -e CONTENT_REPO=https://github.com/you/your-content.git blog-server
```

Pass a local content directory by mounting it as a volume:

```bash
docker run -p 3000:3000 -v /path/to/content:/content -e CONTENT_PATH=/content blog-server
```

### docker-compose

`docker-compose.yaml` reads its configuration from a `.env` file in the repo root (falling back to the defaults above for anything unset).

Easiest way to configure and start it:

```bash
./scripts/run.sh
```

The first time you run it, it prompts for each setting (showing defaults in brackets — press enter to accept) and writes `.env`. On every run after that, it finds the existing `.env` and starts straight away with no prompts.

Other useful modes:

```bash
./scripts/run.sh --reconfigure # re-prompt for every setting, even though .env exists
./scripts/run.sh --configure   # just write/update .env, don't start the server
./scripts/run.sh --yes         # never prompt (first run uses defaults), then run
```

Alternatively, copy `.env.example` to `.env` and edit it by hand, then run `docker compose up --build` directly.
