import type { PageLoad } from "./$types";
import { superValidate } from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";

import { ProductVariantsSchema } from "$lib/schemas/product";

export const load: PageLoad = async () => {
	const form = await superValidate(
		{
			id: "",

			name: "",
			description: "",
			price: 0,
			compare_at_price: undefined,

			images: [],
			is_new: false,
			in_stock: true,
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
	};
};
