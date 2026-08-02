<script lang="ts">
	import { page } from "$app/state";

	import { Badge } from "$lib/components/ui/badge";
	import { Button, buttonVariants } from "$lib/components/ui/button";
	import { cart } from "$lib/stores/cart.svelte";
	import { resolve } from "$app/paths";

	type NavLink = {
		title: string;
		href: string;
	};

	let open = $state(false);

	let cartCount = $derived(
		cart.items.reduce((total, item) => total + item.quantity, 0),
	);

	const logoUrl =
		"https://euro-haus.nyc3.cdn.digitaloceanspaces.com/graphics/eurohaus-logo.png";

	const navLinks = [
		{ title: "About", href: "/about" },
		{ title: "Events", href: "/events" },
		// { title: "Pricing", href: "/pricing-guide" },
		{ title: "Catalog", href: "/catalog" },
		{ title: "Gallery", href: "/gallery" },
	] satisfies NavLink[];

	function closeMenu() {
		open = false;
	}
</script>

<header class="site-header">
	<!-- <div class="event-bar">
		<p><span>Next up</span> 2026 Oktoberfest · Tampa, FL</p>
		<a href={resolve("/events")}
			>See event <span aria-hidden="true">↗</span></a
		>
	</div> -->
	<div class="nav-shell">
		<a
			class="wordmark"
			href={resolve("/")}
			aria-label="Euro Haus home"
			onclick={closeMenu}
		>
			<img
				src={logoUrl}
				alt="Euro Haus logo"
				width="132"
				height="50"
				class="nav-logo"
				loading="eager"
			/>
		</a>

		<nav class:open aria-label="Main navigation">
			{#each navLinks as link}
				<a href={link.href} onclick={closeMenu}>{link.title}</a>
			{/each}
		</nav>

		<a
			class={buttonVariants({
				variant: "ghost",
				size: "default",
				class: "cart-link",
			})}
			href={resolve("/cart")}
			aria-label={`Shopping cart, ${cartCount} ${
				cartCount === 1 ? "item" : "items"
			}`}
		>
			<span>Cart</span>
			<Badge aria-hidden="true">{cartCount}</Badge>
		</a>

		<a
			class={buttonVariants({
				variant: "ghost",
				size: "icon-lg",
				class: "admin-link",
			})}
			href={resolve("/admin")}
			aria-label="Open admin dashboard"
			title="Admin dashboard"
			onclick={closeMenu}
		>
			<svg
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="1.7"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
			>
				<path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z" />
				<path
					d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-1.9 1.9-.06-.06a1.7 1.7 0 0 0-1.88-.34 1.7 1.7 0 0 0-1.03 1.56V20h-2.7v-.09a1.7 1.7 0 0 0-1.03-1.56 1.7 1.7 0 0 0-1.88.34l-.06.06-1.9-1.9.06-.06A1.7 1.7 0 0 0 7.76 15a1.7 1.7 0 0 0-1.56-1.03H6v-2.7h.2A1.7 1.7 0 0 0 7.76 10a1.7 1.7 0 0 0-.34-1.88l-.06-.06 1.9-1.9.06.06a1.7 1.7 0 0 0 1.88.34A1.7 1.7 0 0 0 12.23 5V4h2.7v1a1.7 1.7 0 0 0 1.03 1.56 1.7 1.7 0 0 0 1.88-.34l.06-.06 1.9 1.9-.06.06a1.7 1.7 0 0 0-.34 1.88 1.7 1.7 0 0 0 1.56 1.03H21v2.7h-.04A1.7 1.7 0 0 0 19.4 15Z"
				/>
			</svg>
			<span class="admin-label">Admin</span>
		</a>

		<Button
			class="menu-toggle"
			variant="ghost"
			size="icon-lg"
			type="button"
			aria-label="Toggle menu"
			aria-expanded={open}
			onclick={() => (open = !open)}
		>
			<span></span><span></span>
		</Button>
	</div>
</header>

<style>
	@reference "#layout.css";

	.site-header {
		position: sticky;
		top: 0;
		z-index: 20;
		background: var(--background) / 50;
		width: 100%;
		@apply backdrop-blur supports-backdrop-filter:bg-background/30;
	}
	.event-bar {
		height: 34px;
		padding-inline: 24px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: var(--foreground);
		color: var(--background);
		font-size: 10px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.event-bar p {
		margin: 0;
	}
	.event-bar p span {
		margin-right: 18px;
		color: var(--accent);
	}
	.event-bar a {
		border-bottom: 1px solid
			color-mix(in srgb, var(--background) 42%, var(--primary));
	}
	.nav-shell {
		height: 88px;
		padding-inline: 24px;
		display: flex;
		align-items: center;
		border-bottom: 1px solid var(--color-border);
	}
	.wordmark {
		width: 166px;
		display: flex;
		align-items: center;
		font-family: var(--font-display);
		font-size: 24px;
		font-weight: 700;
		letter-spacing: -0.04em;
		line-height: 1;
	}
	.wordmark strong {
		padding: 6px 7px;
		background: var(--primary);
		color: var(--background);
	}
	.wordmark span {
		padding-right: 5px;
	}
	nav {
		display: flex;
		gap: 38px;
		margin-inline: auto;
	}
	nav a {
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	nav a:hover {
		color: var(--primary);
	}
	.admin-label {
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	:global(.admin-link) {
		display: inline-flex;
		width: auto;
		min-width: 40px;
		align-items: center;
		gap: 7px;
		padding-inline: 10px;
		color: var(--muted);
	}

	:global(.admin-link:hover) {
		color: var(--foreground);
	}

	:global(.admin-link svg) {
		width: 17px;
		height: 17px;
	}
	:global(.nav-cta) {
		width: 166px;
		min-width: 0;
		height: auto;
		padding-bottom: 7px;
	}
	:global(.menu-toggle) {
		display: none;
		border: 0;
		background: transparent;
	}
	:global(.cart-link) {
		display: inline-flex;
		width: auto;
		min-width: 0;
		align-items: center;
		gap: 8px;
	}

	@media (max-width: 800px) {
		.event-bar {
			padding-inline: 16px;
		}
		.event-bar p span {
			display: none;
		}
		.nav-shell {
			height: 70px;
			padding-inline: 16px;
		}
		.nav-logo {
			height: 50px;
			width: auto;
		}
		nav {
			position: absolute;
			top: 104px;
			left: 0;
			right: 0;
			display: none;
			flex-direction: column;
			gap: 0;
			margin: 0;
			padding: 12px 16px 24px;
			background: var(--background);
			border-bottom: 1px solid var(--color-border);
		}
		nav.open {
			display: flex;
		}
		nav a {
			padding: 18px 0;
			border-bottom: 1px solid var(--color-border);
		}
		.admin-label {
			display: none;
		}

		:global(.admin-link) {
			margin-left: 0px;
			margin-right: 4px;
			padding: 8px;
		}

		:global(.nav-cta) {
			display: none;
		}
		:global(.menu-toggle) {
			width: 38px;
			height: 38px;
			margin-left: auto;
			display: flex;
			flex-direction: column;
			justify-content: center;
			gap: 7px;
			padding: 8px;
		}
		:global(.menu-toggle span) {
			display: block;
			width: 100%;
			height: 1px;
			background: var(--foreground);
		}
		:global(.cart-link) {
			display: inline-flex;
			margin-left: auto;
			margin-right: 8px;
		}
	}
</style>
