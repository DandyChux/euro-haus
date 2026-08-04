import { apiRequest } from "$lib/api";
import type {
	AgendaItem,
	Sponsor,
	Event,
	VehicleSubmission,
} from "$lib/schemas/event";
import type { EventGallery, MediaFile } from "$lib/schemas/media";
import type { Price } from "$lib/schemas/price";
import { isUpcoming, parseJsonField, toInt } from "$lib/utils";

type EventAPIResponse = {
	id: string;
	stripe_product_id: string;

	slug: string;
	name: string;
	description: string;
	long_description: string;
	images: string[];

	date: string;
	location: string;
	venue: string;
	organizer?: string;

	capacity: number;
	available_spots: number;

	status: Event["status"];
	active: boolean;
	featured: boolean;

	tags: string[];
	agenda: AgendaItem[];
	includes: string[];
	sponsors: Sponsor[];

	prices?: Price[];
};

type Fetcher = typeof fetch;

async function request<T>(
	fetcher: Fetcher,
	path: string,
	init?: RequestInit,
): Promise<T> {
	return apiRequest<T>(path, init, fetcher);
}

function normalizeEvent(event: EventAPIResponse): Event {
	if (event.prices?.length === 1) {
		event.prices[0].default = true;
	}

	return {
		id: event.id,
		stripe_product_id: event.stripe_product_id,

		slug: event.slug,
		name: event.name,
		description: event.description ?? "",
		long_description: event.long_description ?? "",
		images: event.images ?? [],

		date: event.date,
		location: event.location,
		venue: event.venue ?? "",
		organizer: event.organizer,

		capacity: event.capacity,
		available_spots: event.available_spots,

		status: event.status,
		active: event.active,
		featured: event.featured,

		tags: event.tags ?? [],
		agenda: event.agenda ?? [],
		includes: event.includes ?? [],
		sponsors: event.sponsors ?? [],

		prices: event.prices ?? [],
	};
}

export async function getAllEvents(
	fetcher: Fetcher,
	includeInactive = false,
): Promise<Event[]> {
	const query = includeInactive ? "?include_inactive=true" : "";

	const response = await request<EventAPIResponse[]>(
		fetcher,
		`/events${query}`,
	);

	return response.map(normalizeEvent);
}

export async function getEventByID(
	fetcher: Fetcher,
	id: string,
): Promise<Event | null> {
	try {
		const response = await request<EventAPIResponse>(
			fetcher,
			`/events/${encodeURIComponent(id)}`,
		);

		return normalizeEvent(response);
	} catch {
		return null;
	}
}

export async function getUpcomingEvents(fetcher: Fetcher, limit = 3) {
	const events = await getAllEvents(fetcher);
	const upcoming = events.filter(
		(event) =>
			isUpcoming(event.date) &&
			event.status !== "cancelled" &&
			event.status !== "sold_out",
	);
	return upcoming.slice(0, limit);
}

export async function getFeaturedEvents(fetcher: Fetcher, limit = 3) {
	const events = await getAllEvents(fetcher);
	const upcoming = events.filter(
		(event) =>
			isUpcoming(event.date) &&
			event.status !== "cancelled" &&
			event.status !== "sold_out",
	);

	const featured = upcoming.filter((event) => event.featured);
	const fallback = upcoming.filter((event) => !event.featured);
	return [...featured, ...fallback].slice(0, limit);
}

export async function getAllEventGalleries(
	fetcher: Fetcher,
): Promise<EventGallery[]> {
	const [events, media] = await Promise.all([
		getAllEvents(fetcher, true),
		getMedia(fetcher),
	]);

	return events
		.map((event) => {
			const prefix = `events/${event.id}/gallery/`;
			const images = media
				.filter(
					(file) =>
						file.key.startsWith(prefix) && file.type === "image",
				)
				.sort(
					(a, b) =>
						new Date(b.last_modified).getTime() -
						new Date(a.last_modified).getTime(),
				);

			return {
				id: event.id,
				event_name: event.name,
				event_slug: event.slug,
				description: event.description,
				date: event.date,
				location: event.location,
				images,
			};
		})
		.filter((gallery) => gallery.images.length > 0);
}

export async function getMedia(fetcher: Fetcher): Promise<MediaFile[]> {
	const response = await request<{ files?: MediaFile[] }>(fetcher, "/media");
	return response.files ?? [];
}

export async function getEventGallery(fetcher: Fetcher, slug: string) {
	const media = await getMedia(fetcher);
	const prefix = `events/${slug}/gallery/`;

	return media
		.filter((file) => file.key.startsWith(prefix) && file.type === "image")
		.sort(
			(a, b) =>
				new Date(b.last_modified).getTime() -
				new Date(a.last_modified).getTime(),
		);
}

export async function getSubmission(fetcher: Fetcher, id: string) {
	return request<VehicleSubmission>(fetcher, `/submissions/${id}`);
}
