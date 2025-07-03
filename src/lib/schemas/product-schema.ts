import * as z from 'zod';

// Base schema for all products
export const baseSchema = z.object({
	name: z.string().min(1, 'Name is required'),
	description: z.string().min(1, 'Description is required'),
	price: z.string().min(1, 'Price is required').regex(/^\d+\.?\d{0,2}$/, 'Invalid price format'),
	imageUrl: z.string().url('Invalid URL').or(z.literal('')),
	featured: z.boolean(),
	maxQuantity: z.string().regex(/^\d+$/, 'Must be a number'),
});

// Regular product schema
export const productSchema = baseSchema.extend({
	type: z.literal('product'),
	category: z.enum(['merchandise', 'apparel', 'accessories', 'collectibles']),
	inStock: z.boolean(),
	isNew: z.boolean(),
	compareAtPrice: z.string().regex(/^\d*\.?\d{0,2}$/, 'Invalid price format').optional().or(z.literal('')),
});

// Sponsor schema
export const sponsorSchema = z.object({
	name: z.string().min(1, 'Sponsor name is required'),
	logoUrl: z.string().url('Invalid logo URL').or(z.literal('')),
	link: z.string().url('Invalid sponsor URL').optional().or(z.literal('')),
});

// Event schema
export const eventSchema = baseSchema.extend({
	type: z.literal('event'),
	slug: z.string().min(1, 'Slug is required').regex(/^[a-z0-9-]+$/, 'Invalid slug format'),
	eventDate: z.string().min(1, 'Event date is required'),
	eventTime: z.string().min(1, 'Event time is required'),
	location: z.string().min(1, 'Location is required'),
	capacity: z.string().regex(/^\d+$/, 'Must be a number'),
	availableSpots: z.string().regex(/^\d+$/, 'Must be a number'),
	organizer: z.string(),
	status: z.enum(['upcoming', 'ongoing', 'completed', 'cancelled', 'soldout']),
	tags: z.array(z.object({ value: z.string() })).min(1),
	agenda: z.array(z.object({
		time: z.string().min(1, 'Time is required'),
		activity: z.string().min(1, 'Activity is required'),
	})).min(1),
	includes: z.array(z.object({ value: z.string() })).min(1),
	sponsors: z.array(sponsorSchema).optional(),
});

// Combined schema
export const formSchema = z.discriminatedUnion('type', [productSchema, eventSchema]);

// Type exports
export type BaseFormData = z.infer<typeof baseSchema>;
export type ProductFormData = z.infer<typeof productSchema>;
export type EventFormData = z.infer<typeof eventSchema>;
export type FormData = z.infer<typeof formSchema>;
