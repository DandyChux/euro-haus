import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";
import { superValidate } from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";

import { getProduct } from "$lib/services/stripe";
import { ProductSchema } from "$lib/schemas/product";

export const load: PageLoad = async ({ fetch, params }) => {
	const product = await getProduct(fetch, params.id);

	if (!product) {
		throw error(404, "Product not found");
	}

	const form = await superValidate(
		{
			...product,
			compare_at_price: product.compare_at_price ?? undefined,
			prices: product.prices ?? [],
		},
		zod4(ProductSchema),
	);

	return {
		form,
		product,
	};
};
