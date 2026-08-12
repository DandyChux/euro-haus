import { z } from "zod";

export const CartItemSchema = z.object({
	key: z.string(),
	id: z.string(),
	price_id: z.string().optional(),
	title: z.string(),
	description: z.string(),
	price: z.number(),
	quantity: z.number(),
	imageUrl: z.string().optional(),
	max_quantity: z.number().optional(),
	type: z.enum(["product", "event", "bundle"]).optional(),
	event_date: z.string().optional(),
	metadata: z.record(z.string(), z.unknown()).optional(),
});

export type CartItem = z.infer<typeof CartItemSchema>;

export const OrderSummarySchema = z.object({
	id: z.string(),
	status: z.string(),
	amount: z.number(),
	customer: z.object({
		email: z.string(),
		name: z.string(),
	}),
	items: z.array(
		z.object({
			id: z.string(),
			name: z.string(),
			quantity: z.number(),
			amount: z.number(),
		}),
	),
	created: z.number(),
	total_details: z.unknown().optional(),
});

export type OrderSummary = z.infer<typeof OrderSummarySchema>;
