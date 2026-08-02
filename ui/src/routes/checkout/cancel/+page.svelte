<script lang="ts">
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Checkout canceled · Euro Haus</title>
</svelte:head>

<section class="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
	<div class="rounded-3xl border border-white/10 bg-white/5 p-10 text-center">
		<h1 class="text-3xl font-semibold">Checkout canceled</h1>
		<p class="mt-4 text-base leading-7">
			No payment was made. Your cart is still intact, so you can pick back
			up whenever you’re ready.
		</p>
		<div class="mt-8 flex flex-wrap justify-center gap-3">
			<!-- {#if data.event}
				<a
					href={`/event/${data.event.id}`}
					class="rounded-full bg-white px-5 py-3 text-sm font-medium"
				>
					Return to event
				</a>
			{:else}
				<a
					href="/events"
					class="rounded-full bg-white px-5 py-3 text-sm font-medium"
				>
					View events
				</a>
			{/if} -->
			<a
				href="/cart"
				class="rounded-full bg-white px-5 py-3 text-sm font-medium"
				>Return to cart</a
			>
			<a
				href="/catalog"
				class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium"
				>Continue shopping</a
			>
		</div>
	</div>
</section>
