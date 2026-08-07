<svelte:head>
	<title>Payment · Euro Haus</title>
</svelte:head>

<section class="mx-auto max-w-3xl px-4 py-16 text-center sm:px-6 lg:px-8">
	<p class="text-sm uppercase tracking-[0.3em]">Payment</p>
	<h1 class="mt-4 text-4xl font-semibold text-white">
		Stripe checkout handles payment
	</h1>
	<p class="mt-6 text-base leading-7">
		This route existed in the React app as a shell. In the SvelteKit port,
		checkout currently redirects directly to Stripe from the cart or event
		pages.
	</p>
	<div class="mt-8 flex justify-center gap-3">
		<a
			href="/cart"
			class="rounded-full bg-white px-5 py-3 text-sm font-medium"
			>Back to cart</a
		>
		<a
			href="/catalog"
			class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium text-white"
			>Browse catalog</a
		>
	</div>
</section>
