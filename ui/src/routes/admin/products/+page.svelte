<script lang="ts">
	import { apiClient } from "$lib/api";
	import Button, {
		buttonVariants,
	} from "$lib/components/ui/button/button.svelte";
	import Input from "$lib/components/ui/input/input.svelte";
	import type { Product } from "$lib/schemas/product";
	import { formatCurrency } from "$lib/utils";
	import { CirclePlus } from "@lucide/svelte";

	let { data } = $props();

	let products = $derived([...data.products]);
	let search = $state("");
	let typeFilter = $state<"all" | "product" | "bundle" | "event">("all");
	let statusMessage = $state("");
	let errorMessage = $state("");

	function liveHref(product: Product) {
		return `/catalog/${product.id}`;
	}

	let filteredProducts = $derived.by(() => {
		return products.filter((product) => {
			if (typeFilter !== "all" && product.type !== typeFilter)
				return false;

			if (!search.trim()) return true;

			const value = search.toLowerCase();
			return (
				product.name.toLowerCase().includes(value) ||
				(product.description ?? "").toLowerCase().includes(value)
			);
		});
	});

	async function deleteProduct(product: Product) {
		if (!window.confirm(`Delete "${product.name}"?`)) return;

		statusMessage = "";
		errorMessage = "";

		try {
			await apiClient.delete(`/admin/delete-product/${product.id}`);
			products = products.filter((entry) => entry.id !== product.id);
			statusMessage = `Deleted ${product.name}.`;
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to delete product.";
		}
	}
</script>

<svelte:head>
	<title>Admin products · Euro Haus</title>
</svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Product management</h1>
		<p class="mt-3 max-w-3xl text-sm leading-7">
			Browse and delete products from the Svelte admin. The full
			create/edit builder from React still needs a separate component
			port.
		</p>
	</div>
</header>

<section class="space-y-6">
	<a href="/admin/products/new" class={buttonVariants({ variant: "circle" })}>
		<CirclePlus />
		New Product
	</a>
	<div
		class="grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-5 md:grid-cols-[minmax(0,1fr)_12rem]"
	>
		<Input
			bind:value={search}
			placeholder="Search products"
			class="rounded-2xl px-4 py-3 outline-none focus:border-white/30"
		/>

		<select
			bind:value={typeFilter}
			class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		>
			<option value="all">All types</option>
			<option value="product">Products</option>
			<option value="bundle">Bundles</option>
			<option value="event">Events</option>
		</select>
	</div>

	{#if statusMessage}
		<p
			class="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100"
		>
			{statusMessage}
		</p>
	{/if}

	{#if errorMessage}
		<p
			class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
		>
			{errorMessage}
		</p>
	{/if}

	<div class="grid gap-4">
		{#each filteredProducts as product (product.id)}
			<article
				class="grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-5 md:grid-cols-[7rem_minmax(0,1fr)_auto] md:items-center"
			>
				<div
					class="aspect-square overflow-hidden rounded-2xl bg-zinc-900"
				>
					{#if product.images[0]}
						<img
							src={product.images[0]}
							alt={product.name}
							class="h-full w-full object-cover"
						/>
					{:else}
						<div
							class="flex h-full items-center justify-center text-xs"
						>
							No image
						</div>
					{/if}
				</div>

				<div class="min-w-0">
					<div
						class="flex flex-wrap gap-2 text-xs uppercase tracking-[0.2em]"
					>
						<span>{product.type || "product"}</span>
						<span>{product.active ? "active" : "inactive"}</span>
					</div>
					<h2 class="mt-2 truncate text-lg font-medium">
						{product.name}
					</h2>
					<p class="mt-2 line-clamp-2 text-sm">
						{product.description}
					</p>
					<p class="mt-3 text-sm">
						{formatCurrency(
							(product.default_price?.unit_amount ?? 0) / 100,
						)}
					</p>
				</div>

				<div class="flex flex-wrap gap-3">
					<a
						href={liveHref(product)}
						target="_blank"
						rel="noreferrer"
						class="rounded-full border border-white/10 px-4 py-2 text-sm"
					>
						View live
					</a>
					<Button
						variant="link"
						href={`/admin/products/${product.id}`}
						class={buttonVariants({ variant: "circle" })}
						onclick={() => void deleteProduct(product)}
					>
						Edit product
					</Button>
					<!-- <button
						class="rounded-full border border-destructive/30 px-4 py-2 text-sm text-destructive"
						onclick={() => void deleteProduct(product)}
					>
						Delete
					</button> -->
				</div>
			</article>
		{/each}
	</div>
</section>
