# Euro Haus Stripe Product Management Guide

## Overview

This guide explains how to create and manage products in Stripe for the Euro Haus website. There are two types of products:

1. **Regular Products** - Physical merchandise (shirts, hats, etc.)
2. **Event Tickets** - Tickets for car meetups and track days

## Table of Contents

- [Creating Regular Products](#creating-regular-products)
- [Creating Event Tickets](#creating-event-tickets)
- [Using Product Templates](#using-product-templates)
- [Managing Inventory](#managing-inventory)
- [Best Practices](#best-practices)

---

## Creating Regular Products

### Step 1: Access Stripe Dashboard
1. Log in to [Stripe Dashboard](https://dashboard.stripe.com)
2. Navigate to **Products** from the left menu
3. Click **+ Add product**

### Step 2: Fill Basic Information
- **Name**: Product name (e.g., "Euro Haus Classic T-Shirt")
- **Description**: Brief description of the product
- **Image**: Upload a high-quality product image (recommended: 1200x1200px)

### Step 3: Set Pricing
- **Price**: Enter the price (e.g., $29.99)
- **Currency**: USD
- **Billing period**: One time

### Step 4: Add Metadata
Click **Additional options** → **Metadata** and add these fields:

| Key | Value | Description |
|-----|-------|-------------|
| `type` | `product` | Identifies this as a regular product |
| `category` | `apparel` or `accessories` | Product category |
| `featured` | `true` or `false` | Show on homepage |
| `in_stock` | `true` or `false` | Availability status |
| `is_new` | `true` or `false` | Show "New" badge |
| `max_quantity` | `10` | Max items per order |
| `compare_at_price` | `39.99` | Original price (for sales) |

### Example Regular Product

Name: Euro Haus Classic T-Shirt
Description: Premium cotton t-shirt with embroidered Euro Haus logo
Price: $29.99
Image: [Upload product photo]

Metadata:
- type: product
- category: apparel
- featured: true
- in_stock: true
- is_new: false
- max_quantity: 10

---

## Creating Event Tickets

Event tickets require additional metadata to display properly on the website.

### Step 1: Basic Information
- **Name**: Event name with date (e.g., "Porsche Club Meetup - June 2025")
- **Description**: Brief event description
- **Image**: Event banner image (recommended: 1920x1080px)

### Step 2: Set Ticket Price
- **Price**: Ticket price (e.g., $149.99)
- **Currency**: USD
- **Billing period**: One time

### Step 3: Add Event Metadata
Add ALL of these metadata fields for events:

| Key | Value | Required | Description |
|-----|-------|----------|-------------|
| `type` | `event` | ✅ Yes | Identifies this as an event |
| `slug` | `porsche-club-june-2025` | ✅ Yes | URL-friendly name |
| `event_date` | `2025-06-01T09:00:00Z` | ✅ Yes | ISO format date/time |
| `location` | `Euro Haus HQ, Orlando` | ✅ Yes | Event location |
| `capacity` | `100` | ✅ Yes | Total available tickets |
| `available_spots` | `100` | ✅ Yes | Current available tickets |
| `organizer` | `Euro Haus Events` | No | Event organizer |
| `status` | `upcoming` | No | Event status |
| `featured` | `true` | No | Show on homepage |
| `tags` | `["Porsche", "Track Day"]` | No | Event categories |
| `agenda` | See below | No | Event schedule |
| `includes` | See below | No | What's included |

### Complex Metadata Fields

**Tags** (JSON array as string):
