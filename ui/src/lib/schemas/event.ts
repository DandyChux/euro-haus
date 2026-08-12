import { z } from "zod";
import { priceSchema, type Price } from "./price";
import { submissionRequirementTypeSchema } from "./submission";

export const agendaItemSchema = z.object({
	time: z.string(),
	activity: z.string(),
});

export type AgendaItem = z.infer<typeof agendaItemSchema>;

export const sponsorSchema = z.object({
	name: z.string(),
	tier: z.string(),
	logo: z.string().optional(),
	url: z.string().optional(),
	description: z.string().optional(),
});

export type Sponsor = z.infer<typeof sponsorSchema>;

export const eventStatusSchema = z.enum([
	"upcoming",
	"ongoing",
	"completed",
	"cancelled",
	"sold_out",
]);

export type EventStatus = z.infer<typeof eventStatusSchema>;

export const eventSchema = z.object({
	id: z.string(),
	stripe_product_id: z.string(),

	name: z.string(),
	description: z.string(),
	long_description: z.string(),
	images: z.array(z.string()),

	slug: z.string(),
	date: z.string(),
	location: z.string(),
	venue: z.string(),
	organizer: z.string().optional(),

	capacity: z.number(),
	available_spots: z.number(),

	status: eventStatusSchema,

	tags: z.array(z.string()),
	agenda: z.array(agendaItemSchema),
	includes: z.array(z.string()),
	sponsors: z.array(sponsorSchema),

	active: z.boolean(),
	featured: z.boolean(),

	prices: z.array(priceSchema),
});

export type Event = z.infer<typeof eventSchema>;

export const eventAttendeeSchema = z.object({
	token: z.string(),
	customer_name: z.string(),
	customer_email: z.email(),
	ticket_type: z.string(),
	quantity: z.number(),
	checked_in: z.boolean(),
	checked_in_at: z.string().optional(),
	created_at: z.string().optional(),
});

export type EventAttendee = z.infer<typeof eventAttendeeSchema>;

export const ticketSchema = z.object({
	token: z.string(),
	event_id: z.string(),
	stripe_product_id: z.string(),
	stripe_session_id: z.string(),
	stripe_payment_intent_id: z.string(),
	event_name: z.string(),
	status: z.string().default("active"),
	customer_name: z.string(),
	customer_email: z.email(),
	ticket_type: z.string().default("General"),
	quantity: z.number().int().default(1),
	checked_in: z.boolean().default(false),
	checked_in_at: z.iso.datetime().optional(),
	invalidated: z.boolean().default(false),
	invalidated_reason: z.string(),
	invalidated_at: z.iso.datetime().optional(),
	created_at: z.iso.datetime(),
	updated_at: z.iso.datetime(),
});

export type Ticket = z.infer<typeof ticketSchema>;

export const ticketInfoSchema = z.object({
	valid: z.boolean(),
	customer_name: z.string(),
	customer_email: z.email(),
	event_id: z.string(),
	quantity: z.number().int(),
	checked_in: z.boolean(),
	checked_in_at: z.string().optional(),
	ticket_type: z.string(),
	ticket_code: z.string(),
});

export type TicketInfo = z.infer<typeof ticketInfoSchema>;

export const includedProductSchema = z.object({
	id: z.string(),
	name: z.string(),
	description: z.string().optional(),
	images: z.array(z.string()).optional(),
	quantity: z.number(),
	sortOrder: z.number().optional(),
	default_price: z
		.object({
			id: z.string(),
			unit_amount: z.number(),
			currency: z.string(),
		})
		.optional(),
});

export type IncludedProduct = z.infer<typeof includedProductSchema>;

export const eventCheckInSchema = z.object({
	code: z.string().trim().min(1, "Ticket code is required"),
});

export type EventCheckIn = z.infer<typeof eventCheckInSchema>;
