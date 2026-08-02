<script lang="ts">
	import { resolve } from "$app/paths";
	import Newsletter from "$lib/components/newsletter.svelte";
	import Picture from "$lib/components/picture.svelte";
	import { Button, buttonVariants } from "$lib/components/ui/button/index.js";
	import Video from "$lib/components/video.svelte";
	import { formatDate, generateSrcSet } from "$lib/utils";

	let { data } = $props();
</script>

<svelte:head>
	<title>Euro Haus</title>
	<meta
		name="description"
		content="Euro Haus events, culture, and premium products for European automotive enthusiasts."
	/>
</svelte:head>

<div id="top">
	<section class="hero" aria-labelledby="hero-title">
		<Video
			src="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/videos/euro-haus-intro.mp4"
			name="Hero Background Video"
			description="Main hero section background video"
			autoplay
			loop
			class="hero-video"
		/>
		<div class="hero-shade"></div>
		<div class="hero-content wrap">
			<p class="eyebrow light">European automotive culture · Florida</p>
			<h1 id="hero-title">Built around<br />the drive.</h1>
			<div class="hero-bottom">
				<p>
					Curated events for the cars we love<br />and the people
					behind them.
				</p>
				<a
					class={buttonVariants({ variant: "default" })}
					href={resolve("/events")}
					>Explore events <span aria-hidden="true">↗</span></a
				>
			</div>
		</div>
		<p class="hero-index" aria-hidden="true">01 / 04</p>
	</section>

	<section class="intro section wrap" id="story">
		<div class="section-label">
			<span>01</span>
			<p>Who we are</p>
		</div>
		<div class="intro-copy">
			<p class="eyebrow">More than a car meet</p>
			<h2>Come for the cars.<br /><em>Stay for the people.</em></h2>
			<div class="intro-detail">
				<p>
					Euro Haus creates thoughtful experiences around European
					automotive culture. No velvet ropes, no egos—just remarkable
					machines and genuine connection.
				</p>
				<a
					class={buttonVariants({ variant: "link" })}
					href={resolve("/about")}
					>Meet the community <span aria-hidden="true">→</span></a
				>
			</div>
		</div>
	</section>

	<section class="events section" id="events">
		<div class="wrap">
			<div class="section-heading">
				<div class="section-label">
					<span>02</span>
					<p>On the calendar</p>
				</div>
				<div>
					<p class="eyebrow">Find your next drive</p>
					<h2>Upcoming events</h2>
				</div>
			</div>
			<div class="event-list">
				{#each data.upcomingEvents as event (event.name)}
					<a
						class="event-row"
						href={resolve("/events")}
						aria-label={`Learn about ${event.name}`}
					>
						<div class="event-date">
							<strong
								>{new Date(event.date).toLocaleDateString(
									"en-US",
									{ hour: "numeric" },
								)}</strong
							>
						</div>
						<div class="event-name">
							<p>Event</p>
							<h3>{event.name}</h3>
						</div>
						<div class="event-place">
							<p>{event.location}</p>
							<span>{event.description}</span>
						</div>
						<span
							class={buttonVariants({
								variant: "outline",
								size: "icon-lg",
								class: "event-arrow",
							})}
							aria-hidden="true">↗</span
						>
					</a>
				{/each}
			</div>
			<div class="events-foot">
				<p>New dates are announced throughout the year.</p>
				<a
					class={buttonVariants({ variant: "link" })}
					href={resolve("/events")}
					>View all events <span aria-hidden="true">→</span></a
				>
			</div>
		</div>
	</section>

	<section class="feature">
		<div class="feature-image">
			<Picture
				src="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/PANA1640.jpg"
				alt="Blue GT4"
				loading="lazy"
				sizes="(max-width: 800px) 100vw, 40vw"
				sources={[
					{
						type: "image/webp",
						srcset: generateSrcSet(
							"https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/PANA1640.jpg",
							[800, 1800],
							"webp",
							85,
						),
					},
				]}
			/>
		</div>
		<div class="feature-copy">
			<p class="eyebrow light">The signature event</p>
			<p class="feature-number">{new Date().getFullYear()}</p>
			<h2>Oktoberfest</h2>
			<p>
				European and exotic cars meet the heart of Oktoberfest
				tradition. A full afternoon of rare builds, good food, music,
				and community.
			</p>
			{#if data.signatureEvent}
				<div class="feature-meta">
					<span>{data.signatureEvent.location}</span><span
						>{formatDate(data.signatureEvent.date)}</span
					>
				</div>
				<a
					class={buttonVariants({ variant: "outline" })}
					href={resolve("/event/[id]", {
						id: data.signatureEvent.id,
					})}>Event details <span aria-hidden="true">↗</span></a
				>
			{/if}
		</div>
	</section>

	<section class="gallery section" id="gallery">
		<div class="wrap gallery-head">
			<div class="section-label">
				<span>03</span>
				<p>From the community</p>
			</div>
			<div>
				<p class="eyebrow">Recent stories</p>
				<h2>Life at Euro Haus</h2>
			</div>
			<a
				class={buttonVariants({ variant: "link" })}
				href={resolve("/gallery")}
				>View the gallery <span aria-hidden="true">↗</span></a
			>
		</div>
		<div class="gallery-grid">
			<figure>
				<Picture
					src="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-110-bc12a7a5.jpg"
					alt="White Audi sports car"
					class="grayscale-75"
					loading="lazy"
					sizes="(max-width: 800px) 85vw, 40vw"
					sources={[
						{
							type: "image/webp",
							srcset: generateSrcSet(
								"https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-110-bc12a7a5.jpg",
								[800, 1200],
								"webp",
								85,
							),
						},
					]}
				/>
				<figcaption>
					<span>Machines</span>
					<p>Details make the difference.</p>
				</figcaption>
			</figure>
			<figure>
				<Picture
					src="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-014-2c0ea3ea.jpg"
					alt="White BMW"
					class="grayscale-75"
					loading="lazy"
					sizes="(max-width: 800px) 85vw, 40vw"
					sources={[
						{
							type: "image/webp",
							srcset: generateSrcSet(
								"https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-014-2c0ea3ea.jpg",
								[800, 1200],
								"webp",
								85,
							),
						},
					]}
				/>
				<figcaption>
					<span>Heritage</span>
					<p>Old soul, new stories.</p>
				</figcaption>
			</figure>
			<figure>
				<Picture
					src="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-117-ea1d75ac.jpg"
					alt="White Audi sports car"
					class="grayscale-75"
					loading="lazy"
					sizes="(max-width: 800px) 85vw, 40vw"
					sources={[
						{
							type: "image/webp",
							srcset: generateSrcSet(
								"https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-117-ea1d75ac.jpg",
								[800, 1200],
								"webp",
								85,
							),
						},
					]}
				/>
				<figcaption>
					<span>People</span>
					<p>The reason we gather.</p>
				</figcaption>
			</figure>
		</div>
	</section>

	<Newsletter />
</div>

<style>
	.hero {
		position: relative;
		height: calc(100svh - 85px);
		min-height: 670px;
		overflow: hidden;
		background: var(--foreground);
		color: var(--background);
	}

	:global(.hero-video) {
		position: absolute;
		inset: 0;
		z-index: 0;
		display: block;
		width: 100%;
		height: 100%;
		pointer-events: none;
		object-fit: cover;
		filter: brightness(0.75);
	}

	.hero-shade {
		position: absolute;
		inset: 0;
		z-index: 1;
		background: color-mix(in srgb, var(--foreground) 52%, transparent);
	}

	.hero-content {
		position: relative;
		z-index: 2;
		display: flex;
		height: 100%;
		flex-direction: column;
		justify-content: center;
		gap: clamp(42px, 8vh, 96px);
		padding-top: 0;
		padding-bottom: 0;
	}

	.hero-index {
		position: absolute;
		right: 24px;
		bottom: 22px;
		z-index: 2;
		margin: 0;
	}

	.hero h1 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(105px, 13vw, 230px);
		font-weight: 800;
		letter-spacing: -0.06em;
		line-height: 0.72;
		text-transform: uppercase;
	}
	.hero-bottom {
		display: flex;
		align-items: end;
		justify-content: space-between;
	}
	.hero-bottom p {
		margin: 0;
	}
	.intro {
		display: grid;
		grid-template-columns: 320px 1fr;
		gap: 70px;
	}
	.intro h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(75px, 8vw, 145px);
		letter-spacing: -0.05em;
		line-height: 0.82;
		text-transform: uppercase;
	}
	.intro h2 em {
		color: var(--primary);
		font-style: normal;
	}
	.intro-detail {
		display: grid;
		grid-template-columns: 1fr 220px;
		gap: 80px;
		align-items: end;
		max-width: 900px;
		margin: 60px 0 0 auto;
	}
	.intro-detail p {
		margin: 0;
		color: var(--muted);
		font-size: 16px;
	}
	.events {
		background: color-mix(in srgb, var(--primary) 8%, var(--background));
	}
	.events :global(.section-heading) {
		margin-bottom: 70px;
	}
	.event-list {
		border-top: 1px solid var(--line);
	}
	.event-row {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr 54px;
		gap: 35px;
		align-items: center;
		padding: 30px 0;
		border-bottom: 1px solid var(--line);
		transition: padding 0.25s;
	}
	.event-row:hover {
		padding-left: 12px;
	}
	.event-date {
		display: flex;
		gap: 10px;
		align-items: end;
	}
	.event-date strong {
		font-family: var(--font-display);
		font-size: 52px;
		line-height: 1;
	}
	.event-date span,
	.event-name p,
	.event-place span {
		font-size: 10px;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.event-name p,
	.event-place p,
	.event-place span {
		margin: 0;
		color: var(--muted);
	}
	.event-name h3 {
		margin: 3px 0 0;
		font-family: var(--font-display);
		font-size: 34px;
		text-transform: uppercase;
	}
	.events-foot {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 35px;
	}
	.feature {
		display: grid;
		grid-template-columns: 1.1fr 0.9fr;
		min-height: 800px;
		background: var(--secondary);
		color: var(--background);
	}
	.feature-copy {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		justify-content: center;
		padding: 80px 8vw;
	}
	.feature-number {
		margin: 0;
		color: var(--accent);
		font-family: var(--font-display);
		font-size: 24px;
	}
	.feature-copy h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(90px, 9vw, 160px);
		letter-spacing: -0.055em;
		line-height: 0.8;
		text-transform: uppercase;
	}
	.feature-copy > p:not(.eyebrow, .feature-number) {
		max-width: 480px;
		margin: 35px 0;
		color: color-mix(in srgb, var(--background) 72%, var(--secondary));
	}
	.feature-meta {
		display: flex;
		gap: 30px;
		margin-bottom: 35px;
		font-size: 10px;
		text-transform: uppercase;
	}
	.gallery-head {
		display: grid;
		grid-template-columns: 320px 1fr auto;
		gap: 70px;
		align-items: end;
		margin-bottom: 65px;
	}
	.gallery-head h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(70px, 7vw, 125px);
		line-height: 0.82;
		text-transform: uppercase;
	}
	.gallery-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 20px;
	}
	.gallery-grid figure {
		margin: 0;
	}
	.gallery-grid figure:nth-child(2) {
		margin-top: 90px;
	}
	:global(.gallery-grid img) {
		height: 540px;
		/*filter: saturate(0.7);*/
	}
	.gallery-grid figcaption {
		display: flex;
		justify-content: space-between;
		padding-top: 14px;
		border-top: 1px solid var(--line);
	}
	.gallery-grid figcaption span {
		color: var(--muted);
		font-size: 10px;
		text-transform: uppercase;
	}
	.gallery-grid figcaption p {
		margin: 0;
		font-size: 12px;
	}
	@media (max-width: 800px) {
		.hero {
			height: 670px;
			min-height: calc(100svh - 70px);
		}

		.hero-content {
			justify-content: center;
			gap: 42px;
			padding-top: 0;
			padding-bottom: 0;
		}

		.hero h1 {
			font-size: clamp(72px, 25vw, 110px);
		}

		.hero-bottom {
			align-items: start;
			flex-direction: column;
			gap: 28px;
		}

		.hero-index {
			display: none;
		}
		.intro {
			display: block;
		}
		.intro h2 {
			font-size: 58px;
		}
		.intro-detail {
			display: block;
			margin-top: 42px;
		}
		.intro-detail :global([data-slot="button"]) {
			margin-top: 30px;
		}
		.event-row {
			grid-template-columns: 1fr 1fr 42px;
			gap: 15px;
			padding: 22px 0;
		}
		.event-place {
			grid-column: 2;
		}
		.event-date strong {
			font-size: 36px;
		}
		.event-row :global(.event-arrow) {
			grid-column: 3;
			grid-row: 1/3;
			width: 40px;
			height: 40px;
		}
		.feature {
			display: block;
		}
		.feature-image {
			height: 480px;
		}
		.feature-copy {
			padding: 70px 24px;
		}
		.feature-copy h2 {
			font-size: 76px;
		}
		.gallery-head {
			display: block;
		}
		.gallery-head :global([data-slot="button"]) {
			margin-top: 35px;
		}
		.gallery-grid {
			display: flex;
			overflow-x: auto;
			padding-inline: 16px;
		}
		.gallery-grid figure {
			min-width: 82vw;
			margin: 0 !important;
		}
		:global(.gallery-grid picture) {
			height: 440px;
		}
	}
</style>
