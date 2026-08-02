import { z } from "zod";

export const MediaFileSchema = z.object({
	key: z.string(),
	url: z.string(),
	last_modified: z.string(),
	size: z.number(),
	type: z.enum(["image", "video", "other"]),
	folder: z.string(),
});

export const EventGallerySchema = z.object({
	id: z.string(),
	event_name: z.string(),
	event_slug: z.string(),
	description: z.string().optional(),
	date: z.string().optional(),
	location: z.string().optional(),
	images: z.array(MediaFileSchema),
});

export type MediaFile = z.infer<typeof MediaFileSchema>;
export type EventGallery = z.infer<typeof EventGallerySchema>;
