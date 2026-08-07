import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import { getEventByID, getEventGallery } from "$lib/services/event";

export const load: PageLoad = async ({ fetch, params, url }) => {
	const [event, images] = await Promise.all([
		getEventByID(fetch, params.id),
		getEventGallery(fetch, params.id),
	]);

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
		title: event?.title ?? "Gallery",
		event,
		images: paginatedImages,
		currentPage,
		pageCount,
		start: start + 1,
		end,
	};
};
