import * as z from 'zod';

export const contentPlacementSchema = z.object({
	id: z.string(),
	name: z.string().min(1, 'Name is required'),
	description: z.string(),
	page: z.string().min(1, 'Page is required'),
	type: z.enum(['image', 'video', 'document']),
	mediaUrl: z.string().url('Must be a valid URL'),
	mediaKey: z.string().optional(), // S3 key for reference
	updatedAt: z.string(),
	updatedBy: z.string().optional(),
});

export const updateContentPlacementSchema = z.object({
	mediaUrl: z.string().url('Must be a valid URL'),
	mediaKey: z.string().optional(),
});

export type ContentPlacement = z.infer<typeof contentPlacementSchema>;
export type UpdateContentPlacement = z.infer<typeof updateContentPlacementSchema>;
