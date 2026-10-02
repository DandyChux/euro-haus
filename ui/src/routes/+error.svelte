<script lang="ts">
	import { goto } from "$app/navigation";
	import { page } from "$app/state";

	const status = $derived(page.status || 500);

	const message = $derived(
		page.error?.message ||
			(status === 404
				? "The page you’re looking for has moved or no longer exists."
				: "Something unexpected happened while loading this page."),
	);

	function reload() {
		window.location.reload();
	}

	function goBack() {
		if (window.history.length > 1) {
			window.history.back();
			return;
		}

		void goto("/");
	}
</script>

<svelte:head>
	<title>{status} · Euro Haus</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<main class="error-page">
	<div class="error-glow error-glow-one"></div>
	<div class="error-glow error-glow-two"></div>

	<section class="error-card" aria-labelledby="error-title">
		<div class="error-mark" aria-hidden="true">
			<span>EH</span>
		</div>

		<p class="error-kicker">Euro Haus · {status}</p>

		<h1 id="error-title">
			{status === 404 ? "Wrong turn." : "A rough corner in the road."}
		</h1>

		<p class="error-message">{message}</p>

		<div class="error-actions">
			<button type="button" class="primary-action" onclick={reload}>
				Try again
			</button>

			<button type="button" class="secondary-action" onclick={goBack}>
				Go back
			</button>

			<a class="secondary-action" href="/">Return home</a>
		</div>

		<p class="error-help">
			If the problem continues, wait a moment and try again. We’ll get you
			back on the road.
		</p>
	</section>
</main>

<style>
	.error-page {
		position: relative;
		display: grid;
		min-height: 100svh;
		place-items: center;
		overflow: hidden;
		padding: 2rem 1rem;
		background:
			radial-gradient(
				circle at 20% 15%,
				color-mix(in oklch, var(--accent) 14%, transparent),
				transparent 30rem
			),
			linear-gradient(
				135deg,
				var(--foreground) 0%,
				var(--secondary) 52%,
				var(--foreground) 100%
			);
		color: var(--background);
		font-family: var(--font-body);
	}

	.error-card {
		position: relative;
		z-index: 1;
		width: min(100%, 38rem);
		border: 1px solid
			color-mix(in oklch, var(--background) 14%, transparent);
		border-radius: var(--radius-lg);
		padding: clamp(2rem, 6vw, 4.5rem) clamp(1.5rem, 6vw, 4rem);
		background: color-mix(in oklch, var(--foreground) 78%, transparent);
		box-shadow: 0 2rem 6rem
			color-mix(in oklch, var(--foreground) 35%, transparent);
		text-align: center;
		backdrop-filter: blur(1.2rem);
	}

	.error-mark {
		display: grid;
		width: 4.5rem;
		height: 4.5rem;
		margin: 0 auto 2rem;
		place-items: center;
		border: 1px solid color-mix(in oklch, var(--accent) 70%, transparent);
		border-radius: 50%;
		color: var(--accent);
		font-family: var(--font-display);
		font-size: 1rem;
		letter-spacing: 0.2em;
	}

	.error-kicker {
		margin: 0;
		color: var(--accent);
		font-family: var(--font-body);
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.28em;
		text-transform: uppercase;
	}

	h1 {
		margin: 1rem 0 0;
		font-family: var(--font-display);
		font-size: clamp(2.5rem, 8vw, 4.8rem);
		font-weight: 700;
		letter-spacing: -0.05em;
		line-height: 0.95;
		text-transform: uppercase;
	}

	.error-message {
		max-width: 28rem;
		margin: 1.5rem auto 0;
		color: color-mix(in oklch, var(--background) 72%, transparent);
		font-family: var(--font-body);
		font-size: 0.95rem;
		line-height: 1.7;
	}

	.error-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.75rem;
		margin-top: 2rem;
	}

	.primary-action,
	.secondary-action {
		border-radius: var(--radius);
		padding: 0.8rem 1.2rem;
		font-family: var(--font-body);
		font-size: 0.85rem;
		text-decoration: none;
		transition:
			border-color 150ms ease,
			background 150ms ease,
			color 150ms ease;
	}

	.primary-action {
		border: 1px solid var(--primary);
		background: var(--primary);
		color: var(--primary-foreground);
		cursor: pointer;
	}

	.primary-action:hover {
		border-color: var(--accent);
		background: var(--accent);
		color: var(--accent-foreground);
	}

	.secondary-action {
		border: 1px solid
			color-mix(in oklch, var(--background) 18%, transparent);
		background: transparent;
		color: var(--background);
		cursor: pointer;
	}

	.secondary-action:hover {
		border-color: color-mix(in oklch, var(--accent) 75%, transparent);
		color: var(--accent);
	}

	.error-help {
		margin: 2rem 0 0;
		color: color-mix(in oklch, var(--background) 42%, transparent);
		font-family: var(--font-body);
		font-size: 0.75rem;
		line-height: 1.6;
	}

	.error-glow {
		position: absolute;
		width: 24rem;
		height: 24rem;
		border-radius: 50%;
		filter: blur(5rem);
		opacity: 0.14;
	}

	.error-glow-one {
		top: -12rem;
		right: -8rem;
		background: var(--accent);
	}

	.error-glow-two {
		bottom: -14rem;
		left: -10rem;
		background: var(--secondary);
	}

	@media (max-width: 32rem) {
		.error-actions {
			align-items: stretch;
			flex-direction: column;
		}

		.primary-action,
		.secondary-action {
			width: 100%;
		}
	}
</style>
