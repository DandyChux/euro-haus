<script lang="ts">
	import Newsletter from "$lib/components/newsletter.svelte";
	import { buttonVariants } from "$lib/components/ui/button";
	import { formatCurrency } from "$lib/utils";
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();

	let currentPage = $state(1);
	const ITEMS_PER_PAGE = 9;

	let totalPages = $derived(Math.ceil(data.products.length / ITEMS_PER_PAGE));

	let paginatedProducts = $derived.by(() => {
		const start = (currentPage - 1) * ITEMS_PER_PAGE;
		const end = start + ITEMS_PER_PAGE;
		return data.products.slice(start, end);
	});

	function goToPage(page: number) {
		currentPage = page;
		const section = document.querySelector(".product-section");
		if (section) {
			section.scrollIntoView({ behavior: "smooth" });
		}
	}
</script>

<svelte:head
	><title>Catalog — Euro Haus</title><meta
		name="description"
		content="Browse Euro Haus apparel, event prints, and garage goods."
	/></svelte:head
>

<div id="top">
	<main>
		<section class="catalog-hero">
			<div class="wrap catalog-hero-grid">
				<div>
					<p class="eyebrow">The Haus goods</p>
					<h1>Made for the<br /><em>drive there.</em></h1>
				</div>
				<p>
					Small-run apparel, prints, and garage goods inspired by the
					machines and gatherings that bring us together.
				</p>
			</div>
			<div class="catalog-hero-image">
				<img
					src="https://images.unsplash.com/photo-1495555961986-6d4c1ecb7be3?auto=format&fit=crop&w=2400&q=85"
					alt="Classic European car detail in a workshop"
				/>
			</div>
		</section>

		<section class="product-section section wrap">
			<div class="product-toolbar">
				<div class="section-label">
					<span>01</span>
					<p>Current collection</p>
				</div>
				<p>{data.products.length} pieces · Small runs</p>
			</div>
			<div class="product-grid">
				{#each paginatedProducts as product (product.name)}
					<a
						class="product-card"
						href={`/catalog/${product.id}`}
						aria-label={`View ${product.name} in the live store`}
					>
						<div class="product-image">
							<img
								src={product.images[0]}
								alt={product.name}
							/><span>View item ↗</span>
						</div>
						<div class="product-info">
							<div>
								<p>{product.category}</p>
								<h2>{product.name}</h2>
							</div>
							<strong>{(product.price / 100).toFixed(2)}</strong>
							{#if product.compare_at_price}
								<s>{product.compare_at_price.toFixed(2)}</s>
							{/if}
						</div>
					</a>
				{/each}
			</div>

			{#if totalPages > 1}
				<div class="pagination">
					<button
						class="pagination-btn"
						disabled={currentPage === 1}
						onclick={() => goToPage(currentPage - 1)}
						aria-label="Previous page"
					>
						← Prev
					</button>

					<div class="pagination-pages">
						{#each Array.from({ length: totalPages }, (_, i) => i + 1) as page (page)}
							<button
								class="pagination-page-btn"
								class:active={currentPage === page}
								onclick={() => goToPage(page)}
								aria-label={`Go to page ${page}`}
							>
								{page.toString().padStart(2, "0")}
							</button>
						{/each}
					</div>

					<button
						class="pagination-btn"
						disabled={currentPage === totalPages}
						onclick={() => goToPage(currentPage + 1)}
						aria-label="Next page"
					>
						Next →
					</button>
				</div>
			{/if}
		</section>

		<!-- <section class="catalog-note">
			<div class="wrap">
				<p class="eyebrow light">Made in considered quantities</p>
				<h2>Less stuff.<br />Better souvenirs.</h2>
				<p>
					We create goods that feel useful beyond event day. New
					pieces arrive around major gatherings and may not return
					once they are gone.
				</p>
				<a
					class={buttonVariants({
						variant: "light",
						class: "catalog-cta",
					})}
					href="https://theeurohaus.com"
					target="_blank"
					rel="noreferrer"
					>Visit the live store <span aria-hidden="true">↗</span></a
				>
			</div>
		</section> -->
		<Newsletter />
	</main>
</div>

<style>
	.catalog-hero {
		padding-top: 110px;
	}
	.catalog-hero-grid {
		display: grid;
		grid-template-columns: 1fr 380px;
		gap: 100px;
		align-items: end;
		padding-bottom: 70px;
	}
	.catalog-hero h1 {
		max-width: 980px;
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(86px, 9.4vw, 178px);
		letter-spacing: -0.055em;
		line-height: 0.8;
		text-transform: uppercase;
	}
	.catalog-hero-grid > p {
		margin: 0;
		color: var(--muted);
		font-size: 17px;
	}
	.catalog-hero-image {
		height: 56vh;
		min-height: 520px;
	}
	.catalog-hero-image img {
		filter: saturate(0.62);
	}
	.product-toolbar {
		display: flex;
		align-items: start;
		justify-content: space-between;
		margin-bottom: 55px;
	}
	.product-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 60px 22px;
	}
	.pagination {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 60px;
		padding-top: 30px;
		border-top: 1px solid var(--border);
	}
	.pagination-pages {
		display: flex;
		gap: 15px;
	}
	.pagination-btn,
	.pagination-page-btn {
		background: none;
		border: none;
		font-family: inherit;
		font-size: 14px;
		text-transform: uppercase;
		cursor: pointer;
		color: var(--muted);
		transition:
			color 0.2s,
			opacity 0.2s;
		padding: 8px 12px;
	}
	.pagination-btn:hover:not(:disabled),
	.pagination-page-btn:hover {
		color: var(--foreground);
	}
	.pagination-btn:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}
	.pagination-page-btn.active {
		color: var(--foreground);
		font-weight: 600;
		border-bottom: 2px solid var(--foreground);
	}
	.product-image {
		position: relative;
		height: 500px;
		overflow: hidden;
		background: color-mix(in srgb, var(--primary) 12%, var(--background));
	}
	.product-image img {
		filter: saturate(0.6);
		transition: transform 0.5s;
	}
	.product-image span {
		position: absolute;
		right: 0;
		bottom: 0;
		padding: 11px 15px;
		background: var(--foreground);
		color: var(--background);
		font-size: 10px;
		text-transform: uppercase;
		transform: translateY(100%);
		transition: transform 0.25s;
	}
	.product-card:hover .product-image img {
		transform: scale(1.025);
	}
	.product-card:hover .product-image span {
		transform: none;
	}
	.product-info {
		display: flex;
		justify-content: space-between;
		gap: 24px;
		padding-top: 20px;
		border-top: 1px solid var(--border);
	}
	.product-info p {
		margin: 0;
		color: var(--muted);
		font-size: 10px;
		text-transform: uppercase;
	}
	.product-info h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: 26px;
		text-transform: uppercase;
	}
	.catalog-note {
		padding-block: 120px;
		background: var(--secondary);
		color: var(--background);
	}
	.catalog-note :global(.wrap) {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 60px;
	}
	.catalog-note h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(76px, 8vw, 140px);
		letter-spacing: -0.05em;
		line-height: 0.8;
		text-transform: uppercase;
	}
	.catalog-note .wrap > p:not(.eyebrow) {
		max-width: 460px;
	}
	.catalog-note :global(.catalog-cta) {
		grid-column: 2;
		justify-self: start;
	}
	@media (max-width: 800px) {
		.catalog-hero {
			padding-top: 70px;
		}
		.catalog-hero-grid {
			display: block;
		}
		.catalog-hero h1 {
			font-size: 70px;
		}
		.catalog-hero-grid > p {
			margin-top: 40px;
		}
		.catalog-hero-image {
			height: 450px;
			min-height: 0;
		}
		.product-grid {
			grid-template-columns: 1fr;
		}
		.product-image {
			height: 440px;
		}
		.catalog-note :global(.wrap) {
			display: block;
		}
		.catalog-note h2 {
			font-size: 76px;
		}
		.catalog-note :global(.catalog-cta) {
			margin-top: 30px;
		}
	}
</style>
