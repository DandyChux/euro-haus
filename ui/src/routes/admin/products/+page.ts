import type { PageLoad } from "./$types";
import { getProducts } from "$lib/services/stripe";
import { superValidate } from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";
import { priceEditSchema } from "$lib/schemas/price";

export const load: PageLoad = async ({ fetch }) => {
	const form = await superValidate(
		{
			id: "",
			nickname: "",
			description: "",
			features: [],
			most_popular: false,
			requires_approval: true,
			requires_submission: false,
		},
		zod4(priceEditSchema),
	);

	const products = await getProducts(fetch, true);

	return {
		products,
		form,
	};
};
