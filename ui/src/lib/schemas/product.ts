import { z } from "zod";
import { priceSchema } from "./price";

export const BundleItemSchema = z.object({
	product_id: z.string(),
	product_name: z.string(),
	quantity: z.number(),
	price: z.number(),
});

export type BundleItem = z.infer<typeof BundleItemSchema>;

const ProductBaseSchema = z.object({
	id: z.string(),
	name: z.string(),
	description: z.string(),
	images: z.array(z.string()),
	type: z.string(),

	price: z.number(),
	currency: z.string(),
	compare_at_price: z.number().nullable().optional(),

	is_new: z.boolean(),
	in_stock: z.boolean(),
	active: z.boolean(),
	featured: z.boolean(),

	category: z.string(),
	subcategory: z.string(),
	tags: z.array(z.string()),
	max_quantity: z.number().nullable().optional(),

	default_price: priceSchema.nullable(),
	prices: z.array(priceSchema).default([]),

	created: z.number(),
	updated: z.number(),
});

const BundleProductSchema = ProductBaseSchema.extend({
	type: z.literal("bundle"),
	bundle_items: z.array(BundleItemSchema),
	discount_type: z.enum(["percentage", "fixed"]),
	discount_value: z.number(),
});

const NonBundleProductSchema = ProductBaseSchema.extend({
	type: z.string().refine((type) => type !== "bundle"),
	bundle_items: z.array(BundleItemSchema).default([]),
	discount_type: z.enum(["percentage", "fixed"]).optional(),
	discount_value: z.number().optional(),
});

export const ProductSchema = z.union([
	BundleProductSchema,
	NonBundleProductSchema,
]);

export type Product = z.infer<typeof ProductSchema>;
