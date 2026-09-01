import { z } from "zod";
import { priceSchema } from "./price";

export const BundleItemSchema = z.object({
	productId: z.string(),
	productName: z.string(),
	quantity: z.number(),
	price: z.number(),
});

export type BundleItem = z.infer<typeof BundleItemSchema>;

export const ProductSchema = z.object({
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
	featured: z.boolean(),

	category: z.string(),
	subcategory: z.string(),
	tags: z.array(z.string()),
	max_quantity: z.number().nullable().optional(),

	active: z.boolean(),

	default_price: priceSchema.nullable(),
	prices: z.array(priceSchema).default([]),

	bundle_items: z.array(BundleItemSchema).optional(),
	discount_type: z.enum(["percentage", "fixed"]).optional(),
	discount_value: z.number().optional(),

	created: z.number(),
	updated: z.number(),
});

export type Product = z.infer<typeof ProductSchema>;

export const BundleSchema = ProductSchema.extend({
	bundle_items: z.array(BundleItemSchema),
	discount_type: z.enum(["percentage", "fixed"]),
	discount_value: z.number(),
});

export type BundleProduct = z.infer<typeof BundleSchema>;

export const ProductVariantSchema = z.object({
	id: z.string(),
	price_id: z.string(),
	size: z.string().optional(),
	color: z.string().optional(),
	variant: z.string(),
	price: z.number(),
	in_stock: z.boolean(),
	stock_quantity: z.number().optional(),
	images: z.array(z.string()),
});

export type ProductVariant = z.infer<typeof ProductVariantSchema>;

export const ProductVariantsSchema = ProductSchema.extend({
	variants: z.array(ProductVariantSchema),
});

export type ProductVariants = z.infer<typeof ProductVariantsSchema>;

/* Helper functions */
export function isProductWithVariants(
	product: Product | ProductVariants | BundleProduct,
): product is ProductVariants {
	return "variants" in product;
}

export function isBundleProduct(
	product: Product | ProductVariants | BundleProduct,
): product is BundleProduct {
	return "bundle_items" in product;
}
