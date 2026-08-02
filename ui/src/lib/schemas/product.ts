import { z } from "zod";
import { stripePriceSchema } from "./price";

export const stripeProductSchema = z.object({
	id: z.string(),
	name: z.string(),
	description: z.string().nullable(),
	images: z.array(z.string()),
	metadata: z.record(z.string(), z.string()),
	active: z.boolean(),
	default_price: z
		.object({
			id: z.string(),
			unit_amount: z.number(),
			currency: z.string(),
		})
		.nullable(),
	prices: z.array(stripePriceSchema).optional(),
	created: z.number().optional(),
	updated: z.number().optional(),
});

export type StripeProduct = z.infer<typeof stripeProductSchema>;

export const productSchema = z.object({
	id: z.string(),
	price_id: z.string().optional(),
	title: z.string(),
	description: z.string(),
	price: z.number(),
	compare_at_price: z.number().optional(),
	images: z.array(z.string()),
	is_new: z.boolean().optional(),
	in_stock: z.boolean(),
	featured: z.boolean().optional(),
	category: z.string().optional(),
	subcategory: z.string().optional(),
	tags: z.array(z.string()),
	max_quantity: z.number().optional(),
});

export type Product = z.infer<typeof productSchema>;

export const productVariantSchema = z.object({
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

export type ProductVariant = z.infer<typeof productVariantSchema>;

export const productVariantsSchema = productSchema.extend({
	variants: z.array(productVariantSchema),
});

export type ProductVariants = z.infer<typeof productVariantsSchema>;

export const bundleItemSchema = z.object({
	productId: z.string(),
	productName: z.string(),
	quantity: z.number(),
	price: z.number(),
});

export type BundleItem = z.infer<typeof bundleItemSchema>;

export const bundleSchema = productSchema.extend({
	bundle_items: z.array(bundleItemSchema),
	discount_type: z.enum(["percentage", "fixed"]),
	discount_value: z.number(),
	price: z.number().nonnegative(),
	in_stock: z.boolean(),
	max_quantity: z.number().int().min(1).optional(),
});

export type BundleProduct = z.infer<typeof bundleSchema>;

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
