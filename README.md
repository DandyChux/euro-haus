# Euro Haus

A custom website and internal admin platform built for **Euro Haus**, an
automotive brand focused on community events, media, and premium merchandise.

This repository is shared for **portfolio/review purposes only**. It is not an
open-source project, and the code is not licensed for reuse, modification, or
deployment. See [`LICENSE`](./LICENSE).

## Overview

Euro Haus is a full-stack web platform designed to support both the public-facing
brand experience and the internal operational workflows behind it.

The project combines:

- a public website for showcasing the brand
- event discovery and ticketing flows
- merchandise browsing and checkout
- vehicle submission and approval workflows
- an internal admin dashboard for managing products, events, media, discounts,
  and submissions

Rather than being a generic template site, this project was built around real
business needs and the operational requirements of a live brand.

## What this project includes

### Public experience
- branded landing page and marketing content
- event listing and event detail pages
- product catalog and product detail pages
- cart and checkout flows
- gallery and media presentation
- YouTube/content integration

### Business operations
- admin authentication
- product and event management
- coupon and promotion management
- media library tooling
- vehicle submission review and approval workflows
- fulfillment/payment-related admin actions

## Technical scope

This project includes a custom full-stack architecture with:

- **Frontend:** SvelteKit, Svelte 5, TypeScript, Tailwind CSS
- **Backend:** Go
- **Database:** PostgreSQL
- **Payments:** Stripe
- **Storage / media:** S3-compatible object storage
- **Email workflows:** Mailgun
- **Deployment / infrastructure:** Docker, Caddy

## Highlights

Some of the more interesting implementation areas in this codebase include:

- integrating Stripe for both merchandise and event-related purchase flows
- handling event-specific participant submission workflows
- building internal admin tools alongside the public site
- supporting media upload and management for business content
- structuring a project that serves both marketing and operations needs in one
  system

## Why this repo is public

This repository is public so potential customers and collaborators
can review the quality and scope of the work.

However, this project was created for a real business context and is shared for
**evaluation only**. No rights are granted to copy, reuse, modify, redistribute,
or deploy any portion of this codebase or its assets.

## Notes

- This repository is **not** intended to be used as a starter, template, or
  installable product.
- Some configuration, secrets, and business-specific data are intentionally not
  included.
- Public availability of the source does **not** grant permission to use it.

## License

Proprietary. All rights reserved.

See [`LICENSE`](./LICENSE).
