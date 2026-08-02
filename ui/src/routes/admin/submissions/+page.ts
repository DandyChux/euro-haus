import type { PageLoad } from "./$types";
import { getAllEvents } from "$lib/services/event";

export const load: PageLoad = async ({ fetch }) => {
	return {
		events: await getAllEvents(fetch, true),
	};
};
