<script lang="ts">
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();

	let isRecovering = $state(false);
	let recoveryError = $state("");

	async function recoverSubmissionCheckout() {
		if (!data.submission || !data.event || !data.matchedPriceId) return;

		isRecovering = true;
		recoveryError = "";

		try {
			const response = await fetch("/api/create-participant-checkout", {
				method: "POST",
				headers: {
					"content-type": "application/json",
				},
				body: JSON.stringify({
					submissionId: data.submission.id,
					price_id: data.matchedPriceId,
					eventName: data.event.name,
					quantity: data.submission.ticketQuantity || 1,
				}),
			});

			const payload = await response.json();

			if (!response.ok) {
				throw new Error(
					payload.message ?? "Unable to recover checkout.",
				);
			}

			if (payload.sessionUrl) {
				window.location.href = payload.sessionUrl;
				return;
			}

			throw new Error("The API did not return a Stripe checkout URL.");
		} catch (error) {
			recoveryError =
				error instanceof Error
					? error.message
					: "Unable to recover checkout.";
			isRecovering = false;
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
						{data.submission.vehicleYear}
						{data.submission.vehicleMake}
						{data.submission.vehicleModel}
					</p>
					<p class="mt-3">Participant</p>
					<p class="mt-1 text-white">
						{data.submission.participantName}
					</p>
				</div>
			{/if}

			{#if recoveryError}
				<p
					class="mt-4 rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
				>
					{recoveryError}
				</p>
			{/if}

			<div class="mt-8 flex flex-wrap gap-3">
				<button
					class="rounded-full bg-white px-5 py-3 text-sm font-medium disabled:opacity-60"
					onclick={recoverSubmissionCheckout}
					disabled={isRecovering || !data.matchedPriceId}
				>
					{isRecovering ? "Creating checkout…" : "Resume checkout"}
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
