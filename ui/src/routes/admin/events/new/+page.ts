import type { PageLoad } from "./$types";
import { superValidate } from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";
import { eventSchema } from "$lib/schemas/event";

export const load: PageLoad = async ({ fetch }) => {
	const form = await superValidate(
		{
			id: "",
			stripe_product_id: "",

			name: "",
			description: "",
			long_description: "",
			images: [],

			slug: "",
			date: "",
			location: "",
			venue: "",
			organizer: "",

			capacity: 1,
			available_spots: 1,

			status: "upcoming",

			tags: [],
			agenda: [],
			includes: [],
			sponsors: [],

			active: true,
			featured: false,

			prices: [],
		},
		zod4(eventSchema),
	);

	return {
		form,
	};
};
