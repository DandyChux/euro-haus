<script lang="ts">
	import { addToCart } from "$lib/stores/cart.svelte";
	import { cn, formatCurrency } from "$lib/utils";
	import type { PageProps } from "./$types";
	import { Button } from "$lib/components/ui/button";
	import {
		isBundleProduct,
		isProductWithVariants,
	} from "$lib/schemas/product";

	let { data }: PageProps = $props();

	let activeImage = $state(0);
	let quantity = $state(1);
	let selectedVariantId = $state("");

	let activeVariantId = $derived(
		selectedVariantId ||
			(isProductWithVariants(data.product)
				? (data.product.variants[0]?.id ?? "")
				: ""),
	);

	let selectedVariant = $derived.by(() =>
		isProductWithVariants(data.product)
			? (data.product.variants.find(
					(variant) => variant.id === activeVariantId,
				) ?? null)
			: null,
	);

	let images = $derived(
		selectedVariant?.images.length
			? selectedVariant.images
			: data.product.images,
	);

	let currentPrice = $derived(selectedVariant?.price ?? data.product.price);
	let maxQuantity = $derived(
		selectedVariant?.stock_quantity ?? data.product.max_quantity ?? 10,
	);

	function handleAddToCart() {
		if (!data.product.in_stock) return;
		if (isProductWithVariants(data.product) && !activeVariantId) return;

		const variantLabel = selectedVariant
			? ` · ${selectedVariant.variant}`
			: "";

		addToCart({
			id: data.product.id,
			price_id: selectedVariant?.price_id,
			title: `${data.product.name}${variantLabel}`,
			description: data.product.description,
			price: currentPrice,
			quantity,
			imageUrl: images[activeImage] ?? images[0],
			max_quantity: maxQuantity,
			type: isBundleProduct(data.product) ? "bundle" : "product",
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
				<strong class="text-3xl">{formatCurrency(currentPrice)}</strong>
				{#if data.product.compare_at_price}
					<span class="text-lg line-through">
						{formatCurrency(data.product.compare_at_price)}
					</span>
				{/if}
			</div>

			{#if isProductWithVariants(data.product)}
				<div class="space-y-3">
					<h2 class="text-sm uppercase tracking-[0.2em]">Options</h2>
					<div class="flex flex-wrap gap-3">
						{#each data.product.variants as variant (variant.id)}
							<Button
								variant="outline"
								class={cn("", {
									"border-primary bg-primary/15":
										activeVariantId === variant.id,
								})}
								disabled={!variant.in_stock}
								onclick={() => {
									selectedVariantId = variant.id;
									activeImage = 0;
								}}
							>
								{variant.size || variant.variant}
							</Button>
						{/each}
					</div>
				</div>
			{/if}

			{#if isBundleProduct(data.product) && data.product.bundle_items.length > 0}
				<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
					<h2 class="text-lg font-medium">What’s inside</h2>
					<ul class="mt-4 space-y-3 text-sm">
						{#each data.product.bundle_items as item (`${item.productId}:${item.quantity}`)}
							<li class="flex items-center justify-between gap-4">
								<span>{item.productName}</span>
								<span>x{item.quantity}</span>
							</li>
						{/each}
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
									bundle.discount_value,
								)}
							</a>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	</div>
</section>
