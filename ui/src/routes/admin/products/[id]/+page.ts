import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";
import { superValidate } from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";

import { getProduct } from "$lib/services/stripe";
import { ProductVariantsSchema } from "$lib/schemas/product";

export const load: PageLoad = async ({ fetch, params }) => {
	const product = await getProduct(fetch, params.id);

	if (!product) {
		throw error(404, "Product not found");
	}

	const form = await superValidate(
		{
			id: product.id,

			name: product.name,
			description: product.description ?? "",
			price: product.price ?? 0,
			compare_at_price: product.compare_at_price ?? undefined,

			images: product.images,
			is_new: false,
			in_stock: product.in_stock,
			featured: false,

			category: "merchandise",
			subcategory: "",
			tags: [],
			max_quantity: undefined,

			variants: [],
		},
		zod4(ProductVariantsSchema),
	);

	return {
		form,
		product,
	};
};
