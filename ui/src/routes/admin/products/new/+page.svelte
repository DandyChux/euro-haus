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
			type: product.type,
			price: product.price,
			currency: product.currency,
			compare_at_price: product.compare_at_price,
			images: product.images,
			is_new: product.is_new,
			in_stock: product.in_stock,
			active: product.active,
			featured: product.featured,
			category: product.category,
			subcategory: product.subcategory,
			tags: product.tags,
			max_quantity: product.max_quantity,
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
