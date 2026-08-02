<script lang="ts">
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import ProductForm from "$lib/components/admin/product-form.svelte";
	import type { ProductVariants } from "$lib/schemas/product";

	let { data } = $props();

	async function saveProduct(product: ProductVariants) {
		await apiClient.put(`/admin/update-product/${data.product.id}`, {
			name: product.title,
			description: product.description,
			price: Math.round(product.price * 100),
			currency: "usd",
			images: product.images,
		});

		toast.success("Product updated.");

		await goto("/admin/products");
	}
</script>

<svelte:head>
	<title>Edit product · Euro Haus</title>
</svelte:head>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4">
		<div>
			<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
			<h1 class="mt-2 text-3xl font-semibold">Edit product</h1>
		</div>

		<a href="/admin/products" class="rounded-full border px-4 py-2 text-sm">
			Back to products
		</a>
	</header>

	<ProductForm data={{ form: data.form }} onsaved={saveProduct} />
</div>
