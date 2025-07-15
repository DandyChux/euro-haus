import * as z from 'zod';

// Price tier schema for events
export const priceTierSchema = z.object({
	name: z.string().min(1, 'Tier name is required'),
	price: z.string().min(1, 'Price is required').regex(/^\d+\.?\d{0,2}$/, 'Invalid price format'),
	description: z.string().optional(),
	features: z.array(z.string()).optional(),
	maxQuantity: z.string().regex(/^\d+$/, 'Must be a number').optional().or(z.literal('')),
	sortOrder: z.number(),
});

// Product variant schema for size/color variations
export const productVariantSchema = z.object({
	variantName: z.string().min(1, 'Variant name is required'),
	size: z.string().optional(),
	color: z.string().optional(),
	price: z.string().min(1, 'Price is required').regex(/^\d+\.?\d{0,2}$/, 'Invalid price format'),
	sku: z.string().optional(),
	inStock: z.boolean(),
	sortOrder: z.number(),
});

// Base schema for all products
export const baseSchema = z.object({
	name: z.string().min(1, 'Name is required'),
	description: z.string().min(1, 'Description is required'),
	imageUrl: z.string().url('Invalid URL').or(z.literal('')),
	featured: z.boolean(),
});

// Regular product schema with variants
export const productSchema = baseSchema.extend({
	type: z.literal('product'),
	category: z.enum(['merchandise', 'apparel', 'accessories', 'collectibles']),
	hasVariants: z.boolean(),
	// Single price for products without variants
	price: z.string().min(1, 'Price is required').regex(/^\d+\.?\d{0,2}$/, 'Invalid price format').optional(),
	compareAtPrice: z.string().regex(/^\d*\.?\d{0,2}$/, 'Invalid price format').optional().or(z.literal('')),
	inStock: z.boolean().optional(),
	isNew: z.boolean(),
	maxQuantity: z.string().regex(/^\d+$/, 'Must be a number').optional().or(z.literal('')),
	// Variants for products with size/color options
	variants: z.array(productVariantSchema).optional(),
});

// Sponsor schema
export const sponsorSchema = z.object({
	name: z.string().min(1, 'Sponsor name is required'),
	logoUrl: z.string().url('Invalid logo URL').or(z.literal('')),
	link: z.string().url('Invalid sponsor URL').optional().or(z.literal('')),
});

// Sponsor tier schema
export const sponsorTierSchema = z.object({
	tierName: z.string().min(1, 'Tier name is required'),
	displayOrder: z.number().optional(),
	sponsors: z.array(sponsorSchema),
});

// Event schema with price tiers
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
	tags: z.array(z.object({ value: z.string() })).optional(),
	agenda: z.array(z.object({
		time: z.string().min(1, 'Time is required'),
		activity: z.string().min(1, 'Activity is required'),
	})).min(1),
	includes: z.array(z.object({ value: z.string() })).optional(),
	sponsors: z.array(sponsorSchema).optional(),
	sponsorTiers: z.array(sponsorTierSchema).optional(),
	// Price tiers for events
	hasTiers: z.boolean(),
	// Single price for events without tiers
	price: z.string().min(1, 'Price is required').regex(/^\d+\.?\d{0,2}$/, 'Invalid price format').optional(),
	maxQuantity: z.string().regex(/^\d+$/, 'Must be a number').optional(),
	// Multiple price tiers
	priceTiers: z.array(priceTierSchema).optional(),
});

// Combined schema
export const formSchema = z.discriminatedUnion('type', [productSchema, eventSchema]);

// Type exports
export type BaseFormData = z.infer<typeof baseSchema>;
export type ProductFormData = z.infer<typeof productSchema>;
export type EventFormData = z.infer<typeof eventSchema>;
export type FormData = z.infer<typeof formSchema>;
export type PriceTier = z.infer<typeof priceTierSchema>;
export type ProductVariant = z.infer<typeof productVariantSchema>;
export type Sponsor = z.infer<typeof sponsorSchema>;
export type SponsorTier = z.infer<typeof sponsorTierSchema>;
