<script lang="ts">
	import { addToCart } from "$lib/stores/cart.svelte";
	import { cn, formatCurrency } from "$lib/utils";
	import { Button } from "$lib/components/ui/button";

	let { data } = $props();

	let activeImage = $state(0);
	let quantity = $state(1);
	let selectedPriceId = $state("");

	// let activePriceId = $derived(
	// 	data.product.prices.find((price) => price.id === activePriceId),
	// );
	let activePriceId = $derived(
		selectedPriceId || data.product.prices[0]?.id || "",
	);

	let selectedPrice = $derived(
		data.product.prices.find((price) => price.id === activePriceId) ?? null,
	);

	let images = $derived(data.product.images);

	let currentPrice = $derived(
		selectedPrice?.unit_amount ?? data.product.price,
	);
	let maxQuantity = $derived(
		selectedPrice?.stock_quantity ?? data.product.max_quantity ?? 10,
	);

	function handleAddToCart() {
		if (!data.product.in_stock) return;

		addToCart({
			id: data.product.id,
			price_id: selectedPrice?.id,
			title: selectedPrice?.nickname
				? `${data.product.name} · ${selectedPrice.nickname}`
				: data.product.name,
			price: selectedPrice?.unit_amount
				? selectedPrice.unit_amount / 100
				: data.product.price / 100,
			description: data.product.description,
			quantity,
			imageUrl: images[activeImage] ?? images[0],
			max_quantity: maxQuantity,
		});
	}

	async function shareProduct() {
		const href = window.location.href;

		if (navigator.share) {
			await navigator.share({
				title: data.product.name,
				text: data.product.description,
				url: href,
			});
			return;
		}

		await navigator.clipboard.writeText(href);
	}
</script>

<svelte:head>
	<title>{data.product.name} · Euro Haus</title>
</svelte:head>

<section class="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
	<div class="mb-6 text-sm">
		<a href="/catalog" class="hover:text-primary">Catalog</a> /
		<span>{data.product.name}</span>
	</div>

	<div class="grid gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.9fr)]">
		<div class="space-y-4">
			<div
				class="aspect-square overflow-hidden rounded-3xl border border-white/10 bg-zinc-900"
			>
				{#if images[activeImage]}
					<img
						src={images[activeImage]}
						alt={data.product.name}
						class="h-full w-full object-contain"
					/>
				{:else}
					<div
						class="flex h-full items-center justify-center text-sm"
					>
						No image
					</div>
				{/if}
			</div>

			{#if images.length > 1}
				<div class="grid grid-cols-4 gap-3">
					{#each images as image, index (image)}
						<button
							class={[
								"overflow-hidden rounded-2xl border bg-zinc-900",
								activeImage === index
									? "border-white"
									: "border-white/10",
							]}
							onclick={() => (activeImage = index)}
						>
							<img
								src={image}
								alt={`${data.product.name} ${index + 1}`}
								class="aspect-square w-full object-cover"
							/>
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<div class="space-y-6">
			<div>
				<p class="text-sm uppercase tracking-[0.3em]">
					{data.product.category}
				</p>
				<h1 class="mt-3 text-4xl font-semibold">
					{data.product.name}
				</h1>
				<p class="mt-4 text-base leading-7">
					{data.product.description}
				</p>
			</div>

			<div
				class="flex flex-wrap gap-2 text-xs uppercase tracking-[0.2em]"
			>
				{#if data.product.featured}<span>Featured</span>{/if}
				{#if data.product.is_new}<span>New</span>{/if}
				{#if !data.product.in_stock}<span>Sold out</span>{/if}
			</div>

			<div class="flex items-center gap-3">
				<strong class="text-3xl"
					>{formatCurrency(currentPrice / 100)}</strong
				>

				{#if data.product.compare_at_price}
					<span class="text-lg line-through">
						{formatCurrency(data.product.compare_at_price)}
					</span>
				{/if}
			</div>

			{#if data.product.prices.length > 1}
				<div class="space-y-3">
					<h2 class="text-sm uppercase tracking-[0.2em]">Options</h2>

					<div class="flex flex-wrap gap-3">
						{#each data.product.prices as price (price.id)}
							<Button
								variant="outline"
								class={cn("", {
									"border-primary bg-primary/15":
										activePriceId === price.id,
								})}
								disabled={price.sold_out}
								onclick={() => {
									selectedPriceId = price.id;
								}}
							>
								{price.size || price.nickname || "Standard"}
							</Button>
						{/each}
					</div>
				</div>
			{/if}

			{#if data.product.type === "bundle" && data.product.bundle_items?.length}
				<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
					<h2 class="text-lg font-medium">What's inside</h2>

					<ul class="mt-4 space-y-3 text-sm">
						{#each data.product.bundle_items as item (`${item.product_id}:${item.quantity}`)}
							<li class="flex items-center justify-between gap-4">
								<span
									>{item.product_name} x {item.quantity}</span
								>
							</li>
						{/each}

						{#if data.product.discount_value}
							<span>
								Discount: {data.product.discount_value}
							</span>
						{/if}
					</ul>
				</div>
			{/if}

			<div
				class="flex items-center justify-between rounded-3xl border border-white/10 bg-white/5 px-4 py-3"
			>
				<span class="text-sm">Quantity</span>
				<div class="flex items-center gap-3">
					<Button
						variant="circle"
						size="icon-xs"
						onclick={() => (quantity = Math.max(1, quantity - 1))}
						>−</Button
					>
					<span class="min-w-6 text-center">{quantity}</span>
					<Button
						variant="circle"
						size="icon-xs"
						onclick={() =>
							(quantity = Math.min(maxQuantity, quantity + 1))}
						>+</Button
					>
				</div>
			</div>

			<div class="flex gap-3">
				<Button
					class="flex-1"
					onclick={handleAddToCart}
					disabled={!data.product.in_stock}
				>
					Add to cart
				</Button>
				<Button variant="light" onclick={shareProduct}>Share</Button>
			</div>

			{#if data.containingBundles.length > 0}
				<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
					<h2 class="text-lg font-medium">
						Also included in bundles
					</h2>
					<div class="mt-4 space-y-3">
						{#each data.containingBundles as bundle (bundle.id)}
							<a
								href={`/catalog/${bundle.id}`}
								class="block rounded-2xl border border-white/10 px-4 py-3 text-sm hover:border-white/20 hover:text-primary"
							>
								{bundle.name} · Save {formatCurrency(
									bundle.discount_value ?? 0,
								)}
							</a>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	</div>
</section>
