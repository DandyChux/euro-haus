import { z } from "zod";
import type { IncludedProduct } from "./event";

export const priceSchema = z.object({
	id: z.string(),
	stripe_product_id: z.string(),

	unit_amount: z.number(),
	currency: z.string(),
	nickname: z.string().optional(),
	description: z.string().optional(),

	active: z.boolean(),

	features: z.array(z.string()),
	default: z.boolean(),
	most_popular: z.boolean(),
	requires_approval: z.boolean(),
	requires_submission: z.boolean(),
	included_products: z
		.array(z.custom<IncludedProduct>())
		.nullish()
		.default([]),
	quantity: z.number().optional(),
	sold_out: z.boolean().nullish().default(false),
});

export const priceEditSchema = priceSchema.pick({
	id: true,
	nickname: true,
	description: true,
	features: true,
	most_popular: true,
	requires_approval: true,
	requires_submission: true,
});

export const stripePriceSchema = z.object({
	id: z.string(),
	unit_amount: z.number(),
	currency: z.string(),
	nickname: z.string().nullable().optional(),
	metadata: z.record(z.string(), z.string()),
});

export type Price = z.infer<typeof priceSchema>;
export type StripePrice = z.infer<typeof stripePriceSchema>;
export type PriceEditData = z.infer<typeof priceEditSchema>;

/* Helper functions */
export function getDefaultPrice(prices: Price[]): Price | undefined {
	return (
		prices.find((price) => price.default) ??
		(prices.length === 1 ? prices[0] : undefined)
	);
}

export function getPriceName(price: Price): string {
	return price.nickname || "Standard";
}

export function getPriceAmount(price: Price): number {
	return price.unit_amount / 100;
}

export function priceRequiresVehicleSubmission(price: Price): boolean {
	return price.requires_submission === true;
}

export function priceIsSoldOut(price: Price): boolean {
	return price.sold_out === true;
}
