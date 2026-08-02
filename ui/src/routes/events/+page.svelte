<script lang="ts">
	import { resolve } from "$app/paths";
	import Newsletter from "$lib/components/newsletter.svelte";
	import Picture from "$lib/components/picture.svelte";
	import { buttonVariants } from "$lib/components/ui/button";
	import type { Event } from "$lib/schemas/event";
	import { formatCurrency, formatDate, generateSrcSet } from "$lib/utils";
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();

	const getPriceDisplay = (event: Event) => {
		const activePrices = event.prices.filter((price) => price.active);

		if (activePrices.length === 0) {
			return "Price unavailable";
		}

		const lowestPrice = Math.min(
			...activePrices.map((price) => price.unit_amount / 100),
		);

		return `$${lowestPrice.toFixed(2)} USD`;
	};
</script>

<svelte:head
	><title>Events — Euro Haus</title><meta
		name="description"
		content="Explore the 2026 Euro Haus event calendar and purchase tickets for upcoming European automotive gatherings."
	/></svelte:head
>

<div id="top">
	<main>
		<section class="events-hero">
			<div class="wrap events-hero-grid">
				<div>
					<p class="eyebrow light">The 2026 season</p>
					<h1>Pick a date.<br /><em>Take the drive.</em></h1>
				</div>
				<div class="events-hero-note">
					<p>
						Three distinct gatherings. One welcoming community.
						Choose an event for the full itinerary and secure Stripe
						checkout.
					</p>
					<a
						class={buttonVariants({ variant: "default" })}
						href="#calendar"
						>View the calendar <span aria-hidden="true">↓</span></a
					>
				</div>
			</div>
			<Picture
				src="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-094-b4da1050.jpg"
				alt="European performance cars gathered outdoors"
				sizes="(max-width: 640px) 100vw, (max-width: 1200px) 1800px, 4000px"
				sources={[
					{
						type: "image/webp",
						srcset: generateSrcSet(
							"https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-094-b4da1050.jpg",
							[800, 1800, 4000],
							"webp",
							85,
						),
					},
				]}
			/>
		</section>

		<section class="event-cards section wrap" id="calendar">
			<div class="section-heading">
				<div class="section-label">
					<span>01</span>
					<p>Upcoming</p>
				</div>
				<div>
					<p class="eyebrow">The full calendar</p>
					<h2>ways to gather</h2>
				</div>
			</div>
			{#each data.events as event, index (event.id)}
				<article class="event-card">
					<div class="event-card-image">
						<img
							src={event.images[0]}
							alt={`${event.name} automotive event`}
						/>
						<span>0{index + 1}</span>
					</div>
					<div class="event-card-copy">
						<p class="eyebrow">Event</p>
						<h3>{event.name}</h3>
						<p>{event.description}</p>
					</div>
					<div class="event-card-meta">
						<dl>
							<div>
								<dt>Date</dt>
								<dd>{formatDate(event.date)}</dd>
							</div>
							<div>
								<dt>Place</dt>
								<dd>{event.location}</dd>
							</div>
							<div>
								<dt>Tickets</dt>
								<dd>
									From {getPriceDisplay(event)}
								</dd>
							</div>
						</dl>
						<a
							class={buttonVariants({
								variant: "circle",
								size: "icon-lg",
							})}
							href={resolve(`/event/${event.id}`)}
							aria-label={`View ${event.name} and purchase tickets`}
							>↗</a
						>
					</div>
				</article>
			{/each}
		</section>

		<section class="event-guide section">
			<div class="wrap">
				<div class="section-heading">
					<div class="section-label">
						<span>02</span>
						<p>Good to know</p>
					</div>
					<div>
						<p class="eyebrow">Before you arrive</p>
						<h2>Come as you are</h2>
					</div>
				</div>
				<div class="guide-grid">
					<article>
						<span>01</span>
						<h3>Bring your car</h3>
						<p>
							Featured parking varies by event. General enthusiast
							parking is always welcome while space allows.
						</p>
					</article>
					<article>
						<span>02</span>
						<h3>Bring your people</h3>
						<p>
							Our gatherings are designed for friends, families,
							photographers, and curious passersby alike.
						</p>
					</article>
					<article>
						<span>03</span>
						<h3>Respect the place</h3>
						<p>
							Drive responsibly, follow venue guidance, and help
							us leave every location better than we found it.
						</p>
					</article>
				</div>
			</div>
		</section>
		<Newsletter />
	</main>
</div>

<style>
	.events-hero {
		padding-top: 110px;
		background: var(--foreground);
		color: var(--background);
	}
	.events-hero-grid {
		display: grid;
		grid-template-columns: 1fr 350px;
		gap: 80px;
		align-items: end;
		padding-bottom: 70px;
	}
	.events-hero h1 {
		max-width: 980px;
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(86px, 9.4vw, 178px);
		letter-spacing: -0.055em;
		line-height: 0.8;
		text-transform: uppercase;
	}
	.events-hero-note p {
		margin: 0 0 35px;
	}
	:global(.events-hero img) {
		height: 60vh;
		min-height: 560px;
		filter: saturate(0.65);
	}
	.event-cards :global(.section-heading) {
		margin-bottom: 80px;
	}
	.event-card {
		display: grid;
		grid-template-columns: 1.1fr 0.9fr 0.8fr;
		min-height: 440px;
		border-top: 1px solid var(--border);
	}
	.event-card:last-child {
		border-bottom: 1px solid var(--border);
	}
	.event-card-image {
		position: relative;
		overflow: hidden;
		margin: 26px 42px 26px 0;
	}
	.event-card-image > span {
		position: absolute;
		top: 0;
		left: 0;
		padding: 9px 12px;
		background: var(--primary);
		color: var(--background);
	}
	.event-card-copy {
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 40px;
		border-left: 1px solid var(--border);
	}
	.event-card-copy h3 {
		margin: 0;
		font-family: var(--font-display);
		font-size: 55px;
		text-transform: uppercase;
	}
	.event-card-copy > p:last-child {
		color: var(--muted);
	}
	.event-card-meta {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		padding: 40px;
		border-left: 1px solid var(--border);
	}
	.event-card-meta dl {
		width: 100%;
		margin: 0;
	}
	.event-card-meta dl div {
		display: grid;
		grid-template-columns: 60px 1fr;
		gap: 20px;
		padding: 14px 0;
		border-bottom: 1px solid var(--border);
	}
	.event-card-meta dt {
		font-size: 10px;
		text-transform: uppercase;
	}
	.event-card-meta dd {
		margin: 0;
		color: var(--muted);
		font-size: 12px;
	}
	.event-guide {
		background: color-mix(in srgb, var(--primary) 12%, var(--background));
	}
	.guide-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		margin-top: 70px;
		border: 1px solid var(--border);
	}
	.guide-grid article {
		min-height: 330px;
		padding: 32px;
		border-right: 1px solid var(--border);
	}
	.guide-grid h3 {
		margin: 100px 0 18px;
		font-family: var(--font-display);
		font-size: 38px;
		text-transform: uppercase;
	}
	.guide-grid p {
		color: var(--muted);
	}
	@media (max-width: 800px) {
		.events-hero {
			padding-top: 70px;
		}
		.events-hero-grid {
			display: block;
		}
		.events-hero h1 {
			font-size: 70px;
		}
		.events-hero-note {
			margin-top: 40px;
		}
		:global(.events-hero img) {
			height: 450px;
			min-height: 0;
		}
		.event-card {
			display: block;
			padding-block: 25px;
		}
		.event-card-image {
			height: 330px;
			margin: 0;
		}
		.event-card-copy,
		.event-card-meta {
			padding: 30px 0;
			border-left: 0;
		}
		.event-card-copy h3 {
			font-size: 45px;
		}
		.guide-grid {
			display: block;
		}
		.guide-grid article {
			min-height: 260px;
		}
		.guide-grid h3 {
			margin-top: 70px;
		}
	}
</style>
