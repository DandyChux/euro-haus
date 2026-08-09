import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import {
	getEventByID,
	getEventGallery,
	getAllEvents,
} from "$lib/services/event";

export const load: PageLoad = async ({ fetch, params, url }) => {
	const [allEvents, images] = await Promise.all([
		getAllEvents(fetch, true),
		getEventGallery(fetch, params.slug),
	]);

	let event = allEvents.find((e) => e.slug === params.slug);

	if (!event && images.length === 0) {
		error(404, "Gallery not found");
	}

	const PHOTO_PAGE_SIZE = 24;
	const currentPage = Number(url.searchParams.get("page") ?? "1");
	const pageCount = Math.max(1, Math.ceil(images.length / PHOTO_PAGE_SIZE));
	const start = (currentPage - 1) * PHOTO_PAGE_SIZE;
	const end = Math.min(start + PHOTO_PAGE_SIZE, images.length);
	const paginatedImages = images.slice(start, end);

	return {
		title: event?.name ?? "Event",
		event,
		images: paginatedImages,
		currentPage,
		pageCount,
		start: start + 1,
		end,
	};
};
