import type { PageLoad } from "./$types";
import { getAllEventGalleries } from "$lib/services/event";

export const load: PageLoad = async ({ fetch }) => {
	return {
		albums: await getAllEventGalleries(fetch),
		title: "Gallery",
	};
};
