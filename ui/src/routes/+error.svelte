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
				rgb(218 185 126 / 0.14),
				transparent 30rem
			),
			linear-gradient(135deg, #121212 0%, #20201d 52%, #0d0d0c 100%);
		color: #f5f1e8;
		font-family: Georgia, "Times New Roman", serif;
	}

	.error-card {
		position: relative;
		z-index: 1;
		width: min(100%, 38rem);
		border: 1px solid rgb(255 255 255 / 0.14);
		border-radius: 2rem;
		padding: clamp(2rem, 6vw, 4.5rem) clamp(1.5rem, 6vw, 4rem);
		background: rgb(20 20 18 / 0.78);
		box-shadow: 0 2rem 6rem rgb(0 0 0 / 0.35);
		text-align: center;
		backdrop-filter: blur(1.2rem);
	}

	.error-mark {
		display: grid;
		width: 4.5rem;
		height: 4.5rem;
		margin: 0 auto 2rem;
		place-items: center;
		border: 1px solid rgb(218 185 126 / 0.7);
		border-radius: 50%;
		color: #dab97e;
		font-family: Arial, sans-serif;
		font-size: 1rem;
		letter-spacing: 0.2em;
	}

	.error-kicker {
		margin: 0;
		color: #dab97e;
		font-family: Arial, sans-serif;
		font-size: 0.7rem;
		font-weight: 700;
		letter-spacing: 0.28em;
		text-transform: uppercase;
	}

	h1 {
		margin: 1rem 0 0;
		font-size: clamp(2.5rem, 8vw, 4.8rem);
		font-weight: 400;
		letter-spacing: -0.05em;
		line-height: 0.95;
	}

	.error-message {
		max-width: 28rem;
		margin: 1.5rem auto 0;
		color: rgb(245 241 232 / 0.72);
		font-family: Arial, sans-serif;
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
		border-radius: 999px;
		padding: 0.8rem 1.2rem;
		font-family: Arial, sans-serif;
		font-size: 0.85rem;
		text-decoration: none;
		transition:
			border-color 150ms ease,
			background 150ms ease,
			color 150ms ease;
	}

	.primary-action {
		border: 1px solid #dab97e;
		background: #dab97e;
		color: #171613;
		cursor: pointer;
	}

	.primary-action:hover {
		border-color: #ead09f;
		background: #ead09f;
	}

	.secondary-action {
		border: 1px solid rgb(255 255 255 / 0.18);
		background: transparent;
		color: #f5f1e8;
		cursor: pointer;
	}

	.secondary-action:hover {
		border-color: rgb(218 185 126 / 0.75);
		color: #dab97e;
	}

	.error-help {
		margin: 2rem 0 0;
		color: rgb(245 241 232 / 0.42);
		font-family: Arial, sans-serif;
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
		background: #dab97e;
	}

	.error-glow-two {
		bottom: -14rem;
		left: -10rem;
		background: #7e9a9a;
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
