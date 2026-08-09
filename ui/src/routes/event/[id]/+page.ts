import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

import { getEventByID } from "$lib/services/event";
import { getEventLinkedProducts } from "$lib/services/stripe";
import {
	vehicleSubmissionFormSchema,
	type VehicleSubmissionFormData,
} from "$lib/schemas/event";
import { zod4 } from "sveltekit-superforms/adapters";
import { superValidate } from "sveltekit-superforms";

export const load: PageLoad = async ({ fetch, params, url }) => {
	const event = await getEventByID(fetch, params.id);

	if (!event) {
		error(404, "Event not found");
	}

	const form = await superValidate<VehicleSubmissionFormData>(
		{
			event_id: event.id,
			participant_name: "",
			participant_email: "",
			participant_phone: "",
			vehicle_year: "",
			vehicle_make: "",
			vehicle_model: "",
			vehicle_description: "",
			vehicle_modifications: "",
			price_id: "",
		},
		zod4(vehicleSubmissionFormSchema),
	);

	return {
		event,
		form,
		linked_products: await getEventLinkedProducts(fetch, event.id),
		checkout: url.searchParams.get("checkout"),
		title: event.name,
	};
};
