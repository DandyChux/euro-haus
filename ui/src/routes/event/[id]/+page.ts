import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

import { getEventByID } from "$lib/services/event";
import { getEventLinkedProducts } from "$lib/services/stripe";

export const load: PageLoad = async ({ fetch, params, url }) => {
	const event = await getEventByID(fetch, params.id);

	if (!event) {
		error(404, "Event not found");
	}

	return {
		event,
		linked_products: await getEventLinkedProducts(fetch, event.id),
		checkout: url.searchParams.get("checkout"),
		title: event.name,
	};
};
