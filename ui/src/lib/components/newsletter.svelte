<script lang="ts">
	import apiClient from "$lib/api";
	import { toast } from "svelte-sonner";

	function handleSubmit(event: SubmitEvent) {
		const form = event.target as HTMLFormElement;
		const email = form.email.value;

		apiClient
			.post("/subscribe", { email })
			.then(() => {
				toast.success("Thank you for subscribing!");
				form.reset();
			})
			.catch((error) => {
				toast.error("Something went wrong. Please try again.");
			});
	}
</script>

<section class="newsletter" id="contact">
	<div class="wrap newsletter-inner">
		<div>
			<p class="eyebrow light">Never miss a meet</p>
			<h2>Join the list.</h2>
		</div>
		<form onsubmit={(event) => event.preventDefault()}>
			<label class="sr-only" for="email">Email address</label>
			<input
				id="email"
				type="email"
				placeholder="Your email address"
				required
			/>
			<button type="submit" aria-label="Subscribe">→</button>
			<p>Event announcements, community stories, and nothing else.</p>
		</form>
	</div>
</section>

<style>
	.newsletter {
		padding-block: 100px;
		background: var(--primary);
		color: var(--background);
	}
	.newsletter-inner {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 120px;
		align-items: end;
	}
	h2 {
		margin: 0;
		font-family: var(--font-display);
		font-size: clamp(64px, 7vw, 112px);
		font-weight: 700;
		line-height: 0.86;
		letter-spacing: -0.045em;
		text-transform: uppercase;
	}
	form {
		display: grid;
		grid-template-columns: 1fr 54px;
	}
	form :global([data-slot="input"]) {
		min-width: 0;
		height: auto;
		padding: 16px 0;
		border: 0;
		border-bottom: 1px solid var(--background);
		background: transparent;
		color: var(--background);
		box-shadow: none;
	}
	form :global([data-slot="input"]::placeholder) {
		color: color-mix(in srgb, var(--background) 65%, var(--primary));
	}
	form :global([data-slot="button"]) {
		height: auto;
		border: 0;
		border-bottom: 1px solid var(--background);
		color: var(--background);
		font-size: 22px;
	}
	form p {
		grid-column: 1 / -1;
		margin: 12px 0 0;
		color: color-mix(in srgb, var(--background) 75%, var(--primary));
		font-size: 10px;
	}
	@media (max-width: 800px) {
		.newsletter {
			padding-block: 75px;
		}
		.newsletter-inner {
			display: block;
		}
		h2 {
			font-size: 58px;
		}
		form {
			margin-top: 50px;
		}
	}
</style>
