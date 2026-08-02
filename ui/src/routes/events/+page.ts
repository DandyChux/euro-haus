import type { PageLoad } from "./$types";
import { getAllEvents } from "$lib/services/event";
import { isUpcoming } from "$lib/utils";

export const load: PageLoad = async ({ fetch }) => {
	const events = await getAllEvents(fetch);

	return {
		events: events.filter(
			(event) => isUpcoming(event.date) && event.status !== "cancelled",
		),
		title: "Events",
	};
};
