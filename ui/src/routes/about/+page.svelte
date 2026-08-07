<script>
	import { resolve } from "$app/paths";
	import Newsletter from "$lib/components/newsletter.svelte";
	import PageMasthead from "$lib/components/page-masthead.svelte";
	import Picture from "$lib/components/picture.svelte";
	import { buttonVariants } from "$lib/components/ui/button";
	import { generateSrcSet } from "$lib/utils";
</script>

<svelte:head
	><title>About — Euro Haus</title><meta
		name="description"
		content="The story and values behind Euro Haus, Florida's community for European automotive culture."
	/></svelte:head
>

<div id="top">
	<main>
		<PageMasthead
			eyebrow="Our story"
			index="01"
			title="Cars bring us together."
			intro="Euro Haus is a Florida-based community built for people who believe the best part of car culture is who you meet along the way."
			image="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/events/oktoberfest-2025/gallery/eurohauspics-080-50b00065.jpg"
			alt="Classic European car gathered with other enthusiasts"
		/>

		<section class="about-manifesto section wrap">
			<div class="section-label">
				<span>01</span>
				<p>Why we gather</p>
			</div>
			<div class="manifesto-copy">
				<p class="eyebrow">The road here</p>
				<h2>
					Started with a drive.<br /><em>Grew into a community.</em>
				</h2>
				<div class="manifesto-text">
					<p>
						What began as a small group of friends sharing a love
						for European cars has become a home for collectors,
						builders, first-time owners, and anyone who appreciates
						a good machine.
					</p>
					<p>
						We design gatherings that feel welcoming, considered,
						and worth the drive. No velvet ropes. No gatekeeping.
						Just memorable cars and people who are glad you came.
					</p>
				</div>
			</div>
		</section>

		<section class="about-split">
			<div class="about-split-image">
				<Picture
					src="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/PANA1748.jpg"
					alt="Europeans cars in a lineup"
					sizes="(max-width: 768px) 100vw, 768px"
					sources={[
						{
							type: "image/webp",
							srcset: generateSrcSet(
								"https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/PANA1748.jpg",
								[800, 1200],
								"webp",
								85,
							),
						},
					]}
				/>
			</div>
			<div class="about-split-copy">
				<p class="eyebrow light">Our point of view</p>
				<h2>Enthusiasm<br />over ego.</h2>
				<p>
					A car can be rare, fast, imperfect, inherited, or still a
					work in progress. If it means something to you, it belongs
					in the conversation.
				</p>
				<a
					class={buttonVariants({ variant: "light" })}
					href={resolve("/events")}
					>Meet us at an event <span aria-hidden="true">↗</span></a
				>
			</div>
		</section>

		<section class="values section wrap">
			<div class="section-heading">
				<div class="section-label">
					<span>02</span>
					<p>What guides us</p>
				</div>
				<div>
					<p class="eyebrow">Built into every event</p>
					<h2>Our principles</h2>
				</div>
			</div>
			<div class="value-grid">
				<article>
					<span>01</span>
					<h3>Open doors</h3>
					<p>
						Every generation, badge, budget, and level of knowledge
						is welcome here.
					</p>
				</article>
				<article>
					<span>02</span>
					<h3>Good details</h3>
					<p>
						From location to music to the way cars are placed,
						thoughtful choices shape the experience.
					</p>
				</article>
				<article>
					<span>03</span>
					<h3>Real connection</h3>
					<p>
						The cars start the conversation. The friendships are
						what bring us back.
					</p>
				</article>
			</div>
		</section>

		<section class="about-stats">
			<div class="wrap stat-grid">
				<div><strong>2023</strong><span>Founded in Florida</span></div>
				<div><strong>12+</strong><span>Gatherings hosted</span></div>
				<div><strong>2K+</strong><span>Community members</span></div>
				<div><strong>1</strong><span>Shared obsession</span></div>
			</div>
		</section>
		<Newsletter />
	</main>
</div>

<style>
	.about-manifesto {
		display: grid;
		grid-template-columns: 320px 1fr;
		gap: 70px;
	}
	.manifesto-copy h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(70px, 7.5vw, 135px);
		letter-spacing: -0.05em;
		line-height: 0.82;
		text-transform: uppercase;
	}
	.manifesto-text {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 70px;
		max-width: 900px;
		margin: 60px 0 0 auto;
		color: var(--muted);
		font-size: 16px;
	}
	.manifesto-text p {
		margin: 0;
	}
	.about-split {
		display: grid;
		grid-template-columns: 1.15fr 0.85fr;
		min-height: 780px;
		background: var(--foreground);
		color: var(--background);
	}
	:global(.about-split-image img) {
		filter: saturate(0.78);
		height: 780px;
	}
	.about-split-copy {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		justify-content: center;
		padding: 80px 8vw;
	}
	.about-split-copy h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(80px, 8vw, 140px);
		letter-spacing: -0.055em;
		line-height: 0.78;
		text-transform: uppercase;
	}
	.about-split-copy > p:not(.eyebrow) {
		max-width: 480px;
		margin: 42px 0;
		color: color-mix(in srgb, var(--background) 62%, var(--primary));
	}
	.values :global(.section-heading) {
		margin-bottom: 70px;
	}
	.value-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		border: 1px solid var(--border);
	}
	.value-grid article {
		min-height: 330px;
		padding: 32px;
		border-right: 1px solid var(--border);
	}
	.value-grid h3 {
		margin: 100px 0 18px;
		font-family: var(--font-display);
		font-size: 38px;
		text-transform: uppercase;
	}
	.value-grid p {
		color: var(--muted);
	}
	.about-stats {
		padding-block: 80px;
		background: var(--secondary);
		color: var(--background);
	}
	.stat-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
	}
	.stat-grid div {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 20px 32px;
		border-right: 1px solid
			color-mix(in srgb, var(--background) 35%, transparent);
	}
	.stat-grid strong {
		font-family: var(--font-display);
		font-size: 80px;
		line-height: 1;
	}
	@media (max-width: 800px) {
		.about-manifesto,
		.about-split {
			display: block;
		}
		.about-manifesto > :global(.section-label) {
			margin-bottom: 45px;
		}
		.manifesto-copy h2 {
			font-size: 58px;
		}
		.manifesto-text {
			display: block;
			margin-top: 40px;
		}
		.manifesto-text p + p {
			margin-top: 22px;
		}
		.about-split-image {
			height: 480px;
		}
		.about-split-copy {
			padding: 75px 24px;
		}
		.about-split-copy h2 {
			font-size: 70px;
		}
		.value-grid,
		.stat-grid {
			display: block;
		}
		.value-grid article {
			min-height: 260px;
		}
		.value-grid h3 {
			margin-top: 70px;
		}
		.stat-grid div {
			padding: 25px 0;
			border-right: 0;
			border-bottom: 1px solid
				color-mix(in srgb, var(--background) 35%, transparent);
		}
	}
</style>
