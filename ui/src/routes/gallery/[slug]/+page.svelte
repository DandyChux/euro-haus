<script lang="ts">
	import { resolve } from "$app/paths";
	import Newsletter from "$lib/components/newsletter.svelte";
	import { Badge } from "$lib/components/ui/badge";
	import { buttonVariants } from "$lib/components/ui/button";
	import { formatDate } from "$lib/utils";

	let { data } = $props();

	function pageSearch(pageNumber: number) {
		return pageNumber > 1 ? `?page=${pageNumber}` : "";
	}
</script>

<svelte:head>
	<title>{data.title} Gallery — Euro Haus</title>
	<meta
		name="description"
		content={`Browse all ${data.images.length} photographs from ${data.title}.`}
	/>
</svelte:head>

<div id="top">
	<main>
		<section class="album-hero wrap">
			<a
				class={buttonVariants({ variant: "link" })}
				href={resolve("/gallery")}>← Gallery archive</a
			>
			<div class="album-heading">
				<div>
					<p class="eyebrow">Event album</p>
					<h1>{data.event?.title}</h1>
				</div>
				<div class="album-intro">
					<p>{data.event?.description}</p>
					<dl>
						<div>
							<dt>Date</dt>
							<dd>{data.event?.date}</dd>
						</div>
						<div>
							<dt>Location</dt>
							<dd>{data.event?.location}</dd>
						</div>
						<div>
							<dt>Archive</dt>
							<dd>{data.images.length} photographs</dd>
						</div>
					</dl>
				</div>
			</div>
		</section>

		<section class="album-toolbar wrap" aria-label="Album page status">
			<Badge variant="outline"
				>Page {data.currentPage} of {data.pageCount}</Badge
			>
			<p aria-live="polite">
				Photos {data.start}–{data.end} of {data.images.length}
			</p>
		</section>

		<section
			class="full-grid wrap"
			aria-label={`${data.event?.title} photographs`}
		>
			{#each data.images as photo, index (photo.url)}
				<figure class:wide={index % 11 === 0 || index % 11 === 6}>
					<img
						src={photo.url}
						alt={photo.key}
						loading={index < 2 ? "eager" : "lazy"}
					/>
					<figcaption>
						<span
							>{String(data.start + index).padStart(3, "0")}</span
						>
					</figcaption>
				</figure>
			{/each}
		</section>

		{#if data.pageCount > 1}
			<nav class="pagination wrap" aria-label="Album pagination">
				<a
					class={buttonVariants({
						variant: "outline",
						size: "sm",
						class: data.currentPage === 1 ? "disabled" : "",
					})}
					href={resolve("/gallery/[slug]", {
						slug: data.event?.slug ?? "",
					}) + pageSearch(data.currentPage - 1)}
					aria-disabled={data.currentPage === 1}
					tabindex={data.currentPage === 1 ? -1 : undefined}
					>← Previous</a
				>
				<div class="page-numbers">
					{#each Array.from({ length: data.pageCount }, (_, index) => index + 1) as pageNumber (pageNumber)}
						<a
							class={buttonVariants({
								variant:
									pageNumber === data.currentPage
										? "default"
										: "outline",
								size: "icon-sm",
							})}
							href={resolve("/gallery/[slug]", {
								slug: data.event?.slug ?? "",
							}) + pageSearch(pageNumber)}
							aria-current={pageNumber === data.currentPage
								? "page"
								: undefined}
							aria-label={`Page ${pageNumber}`}>{pageNumber}</a
						>
					{/each}
				</div>
				<a
					class={buttonVariants({
						variant: "outline",
						size: "sm",
						class:
							data.currentPage === data.pageCount
								? "disabled"
								: "",
					})}
					href={resolve("/gallery/[slug]", {
						slug: data.event?.slug ?? "",
					}) + pageSearch(data.currentPage + 1)}
					aria-disabled={data.currentPage === data.pageCount}
					tabindex={data.currentPage === data.pageCount
						? -1
						: undefined}>Next →</a
				>
			</nav>
		{/if}
		<Newsletter />
	</main>
</div>

<style>
	.album-hero {
		padding-top: 105px;
		padding-bottom: 72px;
	}
	.album-heading {
		display: grid;
		grid-template-columns: minmax(0, 1.45fr) minmax(310px, 0.55fr);
		gap: 8vw;
		align-items: end;
		margin-top: 75px;
	}
	.album-heading h1 {
		max-width: 1050px;
		margin: 0;
		font-family: var(--display);
		font-size: clamp(82px, 10vw, 170px);
		letter-spacing: -0.055em;
		line-height: 0.78;
		text-transform: uppercase;
	}
	.album-intro > p {
		margin: 0;
		color: var(--muted);
		font-size: 17px;
		line-height: 1.65;
	}
	.album-intro dl {
		margin: 35px 0 0;
	}
	.album-intro dl div {
		display: grid;
		grid-template-columns: 80px 1fr;
		gap: 20px;
		padding: 12px 0;
		border-bottom: 1px solid var(--line);
	}
	dt {
		color: var(--muted);
		font-size: 9px;
		font-weight: 700;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}
	dd {
		margin: 0;
		font-size: 12px;
		font-weight: 600;
	}
	.album-toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 25px;
		padding-top: 26px;
		padding-bottom: 26px;
		border-block: 1px solid var(--line);
	}
	.album-toolbar p {
		margin: 0;
		color: var(--muted);
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.full-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		grid-auto-rows: 330px;
		gap: 12px;
		padding-top: 70px;
	}
	.full-grid figure {
		position: relative;
		margin: 0;
		overflow: hidden;
		background: var(--ink);
	}
	.full-grid figure.wide {
		grid-column: span 2;
	}
	.full-grid img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		filter: saturate(0.8);
		transition: transform 0.5s;
	}
	.full-grid figure:hover img {
		transform: scale(1.02);
	}
	figcaption {
		position: absolute;
		inset: auto 0 0;
		display: flex;
		gap: 14px;
		padding: 42px 18px 16px;
		background: linear-gradient(
			transparent,
			color-mix(in srgb, var(--ink) 84%, transparent)
		);
		color: var(--page);
		font-size: 11px;
		line-height: 1.4;
	}
	figcaption span {
		color: var(--gold);
		font-family: var(--display);
	}
	.pagination {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 25px;
		padding-top: 45px;
		padding-bottom: 100px;
	}
	.page-numbers {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 8px;
	}
	.pagination :global(.disabled) {
		pointer-events: none;
		opacity: 0.35;
	}
	@media (max-width: 900px) {
		.album-heading {
			grid-template-columns: 1fr;
			gap: 45px;
		}
		.album-intro {
			max-width: 620px;
		}
		.full-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
			grid-auto-rows: 300px;
		}
		.full-grid figure.wide {
			grid-column: span 2;
		}
	}
	@media (max-width: 600px) {
		.album-hero {
			padding-top: 75px;
		}
		.album-heading {
			margin-top: 55px;
		}
		.album-heading h1 {
			font-size: 66px;
		}
		.album-toolbar {
			align-items: flex-start;
			flex-direction: column;
		}
		.full-grid {
			grid-template-columns: 1fr;
			grid-auto-rows: 270px;
			padding-top: 45px;
		}
		.full-grid figure.wide {
			grid-column: auto;
		}
		.pagination {
			flex-wrap: wrap;
		}
		.page-numbers {
			order: -1;
			width: 100%;
		}
	}
</style>
