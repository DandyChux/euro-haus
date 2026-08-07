<script lang="ts">
	import { Button } from "$lib/components/ui/button";
	import {
		cart,
		cartSubtotal,
		clearCart,
		removeFromCart,
		updateCartItemQuantity,
	} from "$lib/stores/cart.svelte";
	import { formatCurrency } from "$lib/utils";

	let checkoutState = $state<"idle" | "loading" | "error">("idle");
	let checkoutError = $state("");
	let subtotal = $derived(cartSubtotal());

	async function checkout() {
		if (cart.items.length === 0) return;

		checkoutState = "loading";
		checkoutError = "";

		try {
			const response = await fetch("/api/create-checkout-session", {
				method: "POST",
				headers: {
					"content-type": "application/json",
				},
				body: JSON.stringify({
					line_items: cart.items.map((item) =>
						item.priceId
							? {
									price: item.priceId,
									quantity: item.quantity,
								}
							: {
									price_data: {
										currency: "usd",
										product_data: {
											name: item.title,
											description: item.description,
											images: item.imageUrl
												? [item.imageUrl]
												: [],
											metadata: {
												type: item.type ?? "product",
											},
										},
										unit_amount: Math.round(
											item.price * 100,
										),
									},
									quantity: item.quantity,
								},
					),
					mode: "payment",
					success_url: `${window.location.origin}/checkout/success?session_id={CHECKOUT_SESSION_ID}`,
					cancel_url: `${window.location.origin}/checkout/cancel`,
					allow_promotion_codes: true,
				}),
			});

			if (!response.ok) {
				throw new Error(await response.text());
			}

			const payload = await response.json();

			if (payload.url) {
				window.location.href = payload.url;
				return;
			}

			throw new Error("Stripe did not return a checkout URL.");
		} catch (error) {
			checkoutState = "error";
			checkoutError =
				error instanceof Error
					? error.message
					: "Unable to start checkout.";
		}
	}
</script>

<svelte:head>
	<title>Cart · Euro Haus</title>
</svelte:head>

<section
	class="mx-auto flex min-h-[70vh] max-w-7xl flex-col gap-8 px-4 py-12 sm:px-6 lg:px-8"
>
	<div
		class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between"
	>
		<div>
			<p class="text-sm uppercase tracking-[0.3em]">Store</p>
			<h1 class="text-4xl font-semibold">Shopping cart</h1>
		</div>

		{#if cart.items.length > 0}
			<button
				class="self-start rounded-full border border-white/10 px-4 py-2 text-sm hover:border-white/30"
				onclick={clearCart}
			>
				Clear cart
			</button>
		{/if}
	</div>

	{#if cart.items.length === 0}
		<div
			class="rounded-3xl border border-dashed border-white/10 bg-white/5 p-10 text-center"
		>
			<h2 class="text-2xl font-medium">Your cart is empty</h2>
			<p class="mt-3">
				Browse products or events and add something that catches your
				eye.
			</p>
			<div class="mt-6 flex flex-wrap justify-center gap-3">
				<a
					href="/catalog"
					class="rounded-full bg-white px-5 py-3 text-sm font-medium"
					>Browse catalog</a
				>
				<a
					href="/events"
					class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium"
					>Browse events</a
				>
			</div>
		</div>
	{:else}
		<div class="grid gap-8 lg:grid-cols-[minmax(0,1fr)_22rem]">
			<div class="space-y-4">
				{#each cart.items as item (item.key)}
					<article
						class="grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-4 sm:grid-cols-[7rem_minmax(0,1fr)_auto] sm:items-center"
					>
						<div
							class="aspect-square overflow-hidden rounded-2xl bg-zinc-900"
						>
							{#if item.imageUrl}
								<img
									src={item.imageUrl}
									alt={item.title}
									class="h-full w-full object-cover"
								/>
							{:else}
								<div
									class="flex h-full items-center justify-center text-sm"
								>
									No image
								</div>
							{/if}
						</div>

						<div class="min-w-0">
							<h2 class="truncate text-lg font-medium">
								{item.title}
							</h2>
							{#if item.description}
								<p class="mt-1 line-clamp-2 text-sm">
									{item.description}
								</p>
							{/if}
							<p class="mt-3 text-sm font-medium">
								{formatCurrency(item.price)}
							</p>
						</div>

						<div class="flex flex-col gap-3 sm:items-end">
							<button
								class="text-sm hover:text-primary"
								onclick={() => removeFromCart(item.key)}
							>
								Remove
							</button>

							<div
								class="flex items-center gap-3 rounded-full border border-white/10 px-3 py-2"
							>
								<button
									class="text-lg leading-none hover:text-primary"
									onclick={() =>
										updateCartItemQuantity(
											item.key,
											item.quantity - 1,
										)}
									disabled={item.quantity <= 1}
								>
									−
								</button>
								<span class="min-w-6 text-center text-sm"
									>{item.quantity}</span
								>
								<button
									class="text-lg leading-none hover:text-primary"
									onclick={() =>
										updateCartItemQuantity(
											item.key,
											item.quantity + 1,
										)}
									disabled={item.max_quantity
										? item.quantity >= item.max_quantity
										: false}
								>
									+
								</button>
							</div>
						</div>
					</article>
				{/each}
			</div>

			<aside class="rounded-3xl border border-white/10 bg-white/5 p-6">
				<h2 class="text-xl font-medium">Order summary</h2>
				<dl class="mt-6 space-y-4 text-sm">
					<div class="flex items-center justify-between">
						<dt>Items</dt>
						<dd>
							{cart.items.reduce(
								(total, item) => total + item.quantity,
								0,
							)}
						</dd>
					</div>
					<div class="flex items-center justify-between">
						<dt>Subtotal</dt>
						<dd>{formatCurrency(subtotal)}</dd>
					</div>
					<div class="flex items-center justify-between">
						<dt>Shipping & tax</dt>
						<dd>Calculated at checkout</dd>
					</div>
				</dl>

				<Button
					class="mt-6 w-full"
					onclick={checkout}
					disabled={checkoutState === "loading"}
				>
					{checkoutState === "loading"
						? "Redirecting to checkout…"
						: "Checkout with Stripe"}
				</Button>

				{#if checkoutError}
					<p
						class="mt-4 rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
					>
						{checkoutError}
					</p>
				{/if}
			</aside>
		</div>
	{/if}
</section>
