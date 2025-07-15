import * as z from 'zod';

export const contentPlacementSchema = z.object({
	id: z.string(),
	name: z.string().min(1, 'Name is required'),
	description: z.string(),
	page: z.string().min(1, 'Page is required'),
	type: z.enum(['image', 'video', 'document', 'text']),
	mediaUrl: z.string().url('Must be a valid URL').optional(),
	mediaKey: z.string().optional(), // S3 key for reference
	textContent: z.string().optional(),
	defaultText: z.string().optional(),
	html: z.boolean().optional(), // Whether the content should be rendered as HTML
	updatedAt: z.string(),
	updatedBy: z.string().optional(),
});

export const updateContentPlacementSchema = z.object({
	mediaUrl: z.string().url('Must be a valid URL'),
	mediaKey: z.string().optional(),
	textContent: z.string().optional(),
});

export const textPlacementRegistrationSchema = z.object({
	id: z.string(),
	name: z.string().min(1, 'Name is required'),
	description: z.string(),
	page: z.string().min(1, 'Page is required'),
	type: z.literal('text'),
	defaultText: z.string(),
	currentText: z.string().optional(),
	html: z.boolean().optional(),
});

export type ContentPlacement = z.infer<typeof contentPlacementSchema>;
export type UpdateContentPlacement = z.infer<typeof updateContentPlacementSchema>;
export type TextPlacementRegistration = z.infer<typeof textPlacementRegistrationSchema>;
