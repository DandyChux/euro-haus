<script lang="ts">
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import ProductForm from "$lib/components/admin/product-form.svelte";
	import type { ProductVariants } from "$lib/schemas/product";

	let { data } = $props();

	async function saveProduct(product: ProductVariants) {
		const response = await apiClient.post<{
			product_id: string;
		}>("/admin/create-product", {
			name: product.name,
			description: product.description,
			price: Math.round(product.price * 100),
			currency: "usd",
			images: product.images,
		});

		toast.success("Product created.");

		await goto(`/admin/products/${response.product_id}`);
	}
</script>

<svelte:head>
	<title>New product · Euro Haus</title>
</svelte:head>

<div class="space-y-6">
	<header>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Create product</h1>
		<p class="mt-3 text-sm text-muted-foreground">
			Create a product and its initial pricing configuration.
		</p>
	</header>

	<ProductForm data={{ form: data.form }} onsaved={saveProduct} />
</div>
