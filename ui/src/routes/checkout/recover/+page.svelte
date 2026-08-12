<script lang="ts">
	import apiClient from "$lib/api.js";

	let { data } = $props();

	let loading = $state(false);
	let errorMessage = $state("");

	async function recoverSubmissionCheckout() {
		if (!data.submission?.id || !data.matchedPriceId) {
			errorMessage = "Unable to determine the submission price.";
			return;
		}

		loading = true;
		errorMessage = "";

		try {
			const response = await apiClient.post<{
				session_id: string;
				session_url: string;
				requires_approval: boolean;
			}>("/create-participant-checkout", {
				submission_id: data.submission.id,
				price_id: data.matchedPriceId,
				event_name: data.event?.name,
				quantity: 1,
			});

			if (!response.session_url) {
				throw new Error("Checkout session URL was not returned.");
			}

			window.location.href = response.session_url;
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to recover checkout.";
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Recover checkout · Euro Haus</title>
</svelte:head>

<section class="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
	<div class="rounded-3xl border border-white/10 bg-white/5 p-8">
		{#if data.error}
			<h1 class="text-3xl font-semibold text-white">Recovery failed</h1>
			<p class="mt-4 text-base leading-7">{data.error}</p>
			<div class="mt-8 flex flex-wrap gap-3">
				<a
					href="/"
					class="rounded-full bg-white px-5 py-3 text-sm font-medium"
					>Return home</a
				>
				<a
					href="/events"
					class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium text-white"
					>Browse events</a
				>
			</div>
		{:else if data.type === "session"}
			<h1 class="text-3xl font-semibold text-white">
				Your Stripe session expired
			</h1>
			<p class="mt-4 text-base leading-7">
				For a regular product checkout, the safest recovery path is to
				go back to the cart and start a new Stripe session.
			</p>
			<div class="mt-8">
				<a
					href="/cart"
					class="rounded-full bg-white px-5 py-3 text-sm font-medium"
					>Return to cart</a
				>
			</div>
		{:else}
			<h1 class="text-3xl font-semibold text-white">
				Complete your registration
			</h1>
			<p class="mt-4 text-base leading-7">
				Your original payment session expired, but your vehicle
				submission is still saved.
			</p>

			{#if data.submission}
				<div
					class="mt-8 rounded-2xl border border-white/10 bg-black/20 p-5 text-sm"
				>
					<p class="">Submission</p>
					<p class="mt-1 text-white">
						{data.submission.vehicle_year}
						{data.submission.vehicle_make}
						{data.submission.vehicle_model}
					</p>
					<p class="mt-3">Participant</p>
					<p class="mt-1 text-white">
						{data.submission.participant_name}
					</p>
				</div>
			{/if}

			{#if errorMessage}
				<p
					class="mt-4 rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
				>
					{errorMessage}
				</p>
			{/if}

			<div class="mt-8 flex flex-wrap gap-3">
				<button
					class="rounded-full bg-white px-5 py-3 text-sm font-medium disabled:opacity-60"
					onclick={recoverSubmissionCheckout}
					disabled={loading || !data.matchedPriceId}
				>
					{loading ? "Creating checkout…" : "Resume checkout"}
				</button>
				<a
					href={data.event ? `/event/${data.event.id}` : "/events"}
					class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium text-white"
				>
					View event
				</a>
			</div>
		{/if}
	</div>
</section>
