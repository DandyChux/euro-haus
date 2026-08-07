<script lang="ts">
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import Newsletter from "$lib/components/newsletter.svelte";
	import { Badge } from "$lib/components/ui/badge";
	import { buttonVariants } from "$lib/components/ui/button";
	import * as Select from "$lib/components/ui/select";
	import type { EventGallery } from "$lib/schemas/media";
	import { formatDate } from "$lib/utils";
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();

	const albumsPerPage = 2;
	let selectedEvent = $derived(page.url.searchParams.get("event") ?? "all");
	let requestedPage = $derived(
		Number(page.url.searchParams.get("page") ?? "1"),
	);
	let filteredAlbums = $derived(
		selectedEvent === "all"
			? data.albums
			: data.albums.filter(
					(album: EventGallery) => album.id === selectedEvent,
				),
	);
	let pageCount = $derived(
		Math.max(1, Math.ceil(filteredAlbums.length / albumsPerPage)),
	);
	let currentPage = $derived(
		Number.isFinite(requestedPage)
			? Math.min(Math.max(Math.trunc(requestedPage), 1), pageCount)
			: 1,
	);
	let visibleAlbums = $derived(
		filteredAlbums.slice(
			(currentPage - 1) * albumsPerPage,
			currentPage * albumsPerPage,
		),
	);
	let visibleStart = $derived(
		filteredAlbums.length === 0 ? 0 : (currentPage - 1) * albumsPerPage + 1,
	);
	let visibleEnd = $derived(
		Math.min(currentPage * albumsPerPage, filteredAlbums.length),
	);

	function gallerySearch(eventId: string, targetPage = 1) {
		const params = new URLSearchParams();
		if (eventId !== "all") params.set("event", eventId);
		if (targetPage > 1) params.set("page", String(targetPage));
		const query = params.toString();
		return query ? `?${query}` : "";
	}

	function updateEvent(eventId: string) {
		goto(resolve("/gallery") + gallerySearch(eventId), {
			keepFocus: true,
			noScroll: true,
		});
	}
</script>

<svelte:head
	><title>Gallery — Euro Haus</title><meta
		name="description"
		content="Scenes, machines, and people from the Euro Haus community."
	/></svelte:head
>

<div id="top">
	<main>
		<section class="gallery-hero wrap">
			<div class="section-label">
				<!-- <span>04</span> -->
				<p>Event archive</p>
			</div>
			<div>
				<p class="eyebrow">Documented by event</p>
				<h1>Every gathering.<br /><em>Kept together.</em></h1>
			</div>
			<div class="hero-copy">
				<p>
					Complete albums from drives, meets, and the quiet moments in
					between.
				</p>
				<div>
					<span>{data.albums.length} albums</span><span
						>{data.albums.reduce(
							(total: number, album: EventGallery) =>
								total + album.images.length,
							0,
						)} photographs</span
					>
				</div>
			</div>
		</section>

		<section class="gallery-controls wrap" aria-label="Gallery controls">
			<div class="filter-field">
				<span>Filter by event</span>
				<Select.Root
					type="single"
					value={selectedEvent}
					onValueChange={updateEvent}
				>
					<Select.Trigger
						class="event-filter"
						aria-label="Filter gallery by event"
						>{selectedEvent === "all"
							? "All events"
							: (data.albums.find(
									(album: EventGallery) =>
										album.id === selectedEvent,
								)?.eventName ?? "All events")}</Select.Trigger
					>
					<Select.Content
						><Select.Group
							><Select.Label>Event archive</Select.Label
							><Select.Item value="all" label="All events"
								>All events</Select.Item
							>{#each data.albums as album (album.id)}<Select.Item
									value={album.id}
									label={album.eventName}
									>{album.eventName}</Select.Item
								>{/each}</Select.Group
						></Select.Content
					>
				</Select.Root>
			</div>
			<p aria-live="polite">
				Showing {visibleStart}–{visibleEnd} of {filteredAlbums.length}
				{filteredAlbums.length === 1 ? "album" : "albums"}
			</p>
		</section>

		<section class="album-list wrap" aria-label="Event albums">
			{#each visibleAlbums as album, albumIndex (album.id)}
				<article class="album" id={album.id}>
					<header class="album-header">
						<div class="album-number" aria-hidden="true">
							{String(
								(currentPage - 1) * albumsPerPage +
									albumIndex +
									1,
							).padStart(2, "0")}
						</div>
						<div class="album-title">
							<Badge variant="outline"
								>{album.images.length} photos</Badge
							>
							<h2>{album.eventName}</h2>
						</div>
						<div class="album-details">
							<p>{album.description}</p>
							<dl>
								<div>
									<dt>Date</dt>
									<dd>{formatDate(album.date)}</dd>
								</div>
								<div>
									<dt>Location</dt>
									<dd>{album.location}</dd>
								</div>
							</dl>
						</div>
					</header>
					<div class="photo-grid">
						{#each album.images.slice(0, 6) as photo, photoIndex (photo.url)}
							<figure class:featured={photoIndex === 0}>
								<img
									src={photo.url}
									alt={photo.key}
									loading="lazy"
								/>
								<figcaption>
									<span
										>{String(photoIndex + 1).padStart(
											2,
											"0",
										)}</span
									>
								</figcaption>
							</figure>
						{/each}
					</div>
					<footer class="album-footer">
						<p>Previewing 6 of {album.images.length} photographs</p>
						<a
							class={buttonVariants({ variant: "outline" })}
							href={resolve("/gallery/[id]", { id: album.id })}
							>View full album <span aria-hidden="true">→</span
							></a
						>
					</footer>
				</article>
			{/each}
		</section>

		{#if pageCount > 1}
			<nav class="pagination wrap" aria-label="Gallery pagination">
				<a
					class={buttonVariants({
						variant: "outline",
						size: "sm",
						class: currentPage === 1 ? "disabled" : "",
					})}
					href={resolve("/gallery") +
						gallerySearch(selectedEvent, currentPage - 1)}
					aria-disabled={currentPage === 1}
					tabindex={currentPage === 1 ? -1 : undefined}>← Previous</a
				>
				<div class="page-numbers">
					{#each Array.from({ length: pageCount }, (_, index) => index + 1) as pageNumber (pageNumber)}<a
							class={buttonVariants({
								variant:
									pageNumber === currentPage
										? "default"
										: "outline",
								size: "icon-sm",
							})}
							href={resolve("/gallery") +
								gallerySearch(selectedEvent, pageNumber)}
							aria-current={pageNumber === currentPage
								? "page"
								: undefined}
							aria-label={`Page ${pageNumber}`}>{pageNumber}</a
						>{/each}
				</div>
				<a
					class={buttonVariants({
						variant: "outline",
						size: "sm",
						class: currentPage === pageCount ? "disabled" : "",
					})}
					href={resolve("/gallery") +
						gallerySearch(selectedEvent, currentPage + 1)}
					aria-disabled={currentPage === pageCount}
					tabindex={currentPage === pageCount ? -1 : undefined}
					>Next →</a
				>
			</nav>
		{/if}

		<section class="gallery-quote">
			<div class="wrap">
				<p class="eyebrow light">Seen something good?</p>
				<blockquote>
					“The best car photos make you remember how the day felt.”
				</blockquote>
				<!-- <a
					class={buttonVariants({ variant: "light" })}
					href="mailto:info@theeurohaus.com?subject=Photo submission"
					>Submit your photos <span aria-hidden="true">↗</span></a
				> -->
			</div>
		</section>
		<Newsletter />
	</main>
</div>

<style>
	.gallery-hero {
		display: grid;
		grid-template-columns: 200px 1fr 310px;
		gap: 55px;
		align-items: end;
		padding-top: 120px;
		padding-bottom: 90px;
	}
	.gallery-hero h1 {
		max-width: 980px;
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(76px, 8.5vw, 154px);
		letter-spacing: -0.055em;
		line-height: 0.8;
		text-transform: uppercase;
	}
	.hero-copy > p {
		margin: 0;
		color: var(--muted);
		line-height: 1.6;
	}
	.hero-copy > div {
		display: flex;
		gap: 22px;
		margin-top: 25px;
		padding-top: 15px;
		border-top: 1px solid var(--border);
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.gallery-controls {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 30px;
		padding-top: 30px;
		padding-bottom: 30px;
		border-block: 1px solid var(--border);
	}
	.filter-field {
		width: min(360px, 100%);
	}
	.filter-field > span {
		display: block;
		margin-bottom: 9px;
		color: var(--primary);
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.14em;
		text-transform: uppercase;
	}
	.gallery-controls > p {
		margin: 0;
		color: var(--muted);
		font-size: 11px;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}
	.gallery-controls :global(.event-filter) {
		width: 100%;
		height: 48px;
		border-color: var(--foreground);
		background: var(--background);
	}
	.album {
		padding-block: 95px;
		border-bottom: 1px solid var(--border);
		scroll-margin-top: 100px;
	}
	.album-header {
		display: grid;
		grid-template-columns: 70px minmax(250px, 0.8fr) 1fr;
		gap: 4vw;
		align-items: start;
		margin-bottom: 42px;
	}
	.album-number {
		font-family: var(--font-display);
		font-size: 18px;
		color: var(--primary);
	}
	.album-title {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 18px;
	}
	.album-title h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(48px, 5vw, 78px);
		letter-spacing: -0.04em;
		line-height: 0.86;
		text-transform: uppercase;
	}
	.album-details {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 35px;
	}
	.album-details > p {
		margin: 0;
		color: var(--muted);
		line-height: 1.65;
		overflow: hidden;
		display: -webkit-box;
		-webkit-box-orient: vertical;
		line-clamp: 4;
		-webkit-line-clamp: 4;
	}
	dl {
		margin: 0;
	}
	dl div {
		display: grid;
		grid-template-columns: 70px 1fr;
		gap: 12px;
		padding: 10px 0;
		border-bottom: 1px solid var(--border);
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
	.photo-grid {
		display: grid;
		grid-template-columns: 1.35fr 0.65fr 0.65fr;
		grid-template-rows: repeat(2, minmax(230px, 1fr));
		gap: 12px;
	}
	.photo-grid figure {
		position: relative;
		min-height: 230px;
		margin: 0;
		overflow: hidden;
		background: var(--foreground);
	}
	.photo-grid figure.featured {
		grid-row: 1/3;
	}
	.photo-grid img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		filter: saturate(0.8);
		transition: transform 0.55s;
	}
	.photo-grid figure:hover img {
		transform: scale(1.025);
	}
	figcaption {
		position: absolute;
		inset: auto 0 0;
		display: flex;
		gap: 14px;
		padding: 35px 18px 16px;
		background: linear-gradient(
			transparent,
			color-mix(in srgb, var(--foreground) 82%, transparent)
		);
		color: var(--background);
		font-size: 11px;
		line-height: 1.4;
	}
	figcaption span {
		color: var(--accent);
		font-family: var(--font-display);
	}
	.album-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 30px;
		margin-top: 25px;
	}
	.album-footer p {
		margin: 0;
		color: var(--muted);
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.pagination {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 25px;
		padding-top: 42px;
		padding-bottom: 95px;
	}
	.page-numbers {
		display: flex;
		gap: 8px;
	}
	.pagination :global(.disabled) {
		pointer-events: none;
		opacity: 0.35;
	}
	.gallery-quote {
		padding-block: 130px;
		background: var(--foreground);
		color: var(--background);
	}
	.gallery-quote blockquote {
		max-width: 1100px;
		margin: 45px 0 60px;
		font-family: var(--font-display);
		font-size: clamp(65px, 7vw, 125px);
		letter-spacing: -0.045em;
		line-height: 0.88;
		text-transform: uppercase;
	}
	@media (max-width: 900px) {
		.gallery-hero {
			grid-template-columns: 1fr;
			gap: 38px;
			padding-top: 80px;
		}
		.gallery-hero > :global(.section-label) {
			margin-bottom: 10px;
		}
		.hero-copy {
			max-width: 560px;
		}
		.album-header {
			grid-template-columns: 50px 1fr;
		}
		.album-details {
			grid-column: 2;
			grid-template-columns: 1fr;
		}
		.photo-grid {
			grid-template-columns: 1fr 1fr;
			grid-template-rows: 360px 230px;
		}
		.photo-grid figure.featured {
			grid-column: 1/3;
			grid-row: auto;
		}
	}
	@media (max-width: 600px) {
		.gallery-hero h1 {
			font-size: 62px;
		}
		.hero-copy > div {
			flex-wrap: wrap;
		}
		.gallery-controls {
			align-items: stretch;
			flex-direction: column;
		}
		.album {
			padding-block: 65px;
		}
		.album-header {
			grid-template-columns: 1fr;
			gap: 20px;
		}
		.album-number {
			display: none;
		}
		.album-details {
			grid-column: auto;
			grid-template-columns: 1fr;
			gap: 20px;
		}
		.photo-grid {
			display: flex;
			flex-direction: column;
		}
		.photo-grid figure,
		.photo-grid figure.featured {
			height: 260px;
			min-height: 260px;
		}
		.photo-grid figure.featured {
			height: 410px;
		}
		.album-footer {
			align-items: stretch;
			flex-direction: column;
		}
		.pagination {
			flex-wrap: wrap;
		}
		.page-numbers {
			order: -1;
			width: 100%;
			justify-content: center;
		}
		.gallery-quote blockquote {
			font-size: 58px;
		}
	}
</style>
