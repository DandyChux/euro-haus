import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

import { getEventByID } from "$lib/services/event";
import { getEventLinkedProducts } from "$lib/services/stripe";
import type { EventAttendee } from "$lib/schemas/event";

export const load: PageLoad = async ({ fetch, url }) => {
	const eventID = url.searchParams.get("id");

	if (!eventID) {
		return {
			event: null,
			linked_products: [],
			attendees: [] as EventAttendee[],
		};
	}

	const event = await getEventByID(fetch, eventID);

	if (!event) {
		error(404, "Event not found");
	}

	let attendees: EventAttendee[] = [];

	try {
		const response = await fetch(
			`/api/events/${encodeURIComponent(event.id)}/tickets`,
		);

		if (response.ok) {
			const payload = (await response.json()) as {
				tickets?: EventAttendee[];
			};

			attendees = payload.tickets ?? [];
		}
	} catch {
		attendees = [];
	}

	return {
		event,
		linked_products: await getEventLinkedProducts(fetch, event.id),
		attendees,
	};
};
