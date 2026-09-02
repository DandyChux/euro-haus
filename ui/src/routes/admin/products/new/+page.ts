import type { PageLoad } from "./$types";
import { superValidate } from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";

import { ProductSchema } from "$lib/schemas/product";

export const load: PageLoad = async () => {
	const form = await superValidate(
		{
			id: "",
			name: "",
			description: "",
			type: "product",
			price: 0,
			currency: "usd",
			compare_at_price: undefined,
			images: [],
			is_new: false,
			in_stock: true,
			active: true,
			featured: false,
			category: "merchandise",
			subcategory: "",
			tags: [],
			max_quantity: undefined,
			default_price: null,
			prices: [],
			created: 0,
			updated: 0,
		},
		zod4(ProductSchema),
	);

	return {
		form,
	};
};
