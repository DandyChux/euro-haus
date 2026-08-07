import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";
import { superValidate } from "sveltekit-superforms";
import { zod4 } from "sveltekit-superforms/adapters";

import { getAdminEventByID } from "$lib/services/event";
import { eventSchema } from "$lib/schemas/event";

export const load: PageLoad = async ({ fetch, params }) => {
	const event = await getAdminEventByID(fetch, params.id);

	if (!event) {
		throw error(404, "Event not found");
	}

	const form = await superValidate(event, zod4(eventSchema));

	return {
		event,
		form,
	};
};
