import type { PageLoad } from "./$types";
import { getEventByID } from "$lib/services/event";

export const load: PageLoad = async ({ fetch, url }) => {
	const eventID = url.searchParams.get("event_id");

	return {
		event: eventID ? await getEventByID(fetch, eventID) : null,
		title: "Checkout canceled",
	};
};
