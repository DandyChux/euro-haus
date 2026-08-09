import { z } from "zod";
import { priceSchema, type Price } from "./price";

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

export const vehicleSubmissionSchema = z.object({
	id: z.string(),
	event_id: z.string(),
	participant_name: z.string(),
	participant_email: z.email(),
	participant_phone: z.string().optional(),
	vehicle_year: z.string(),
	vehicle_make: z.string(),
	vehicle_model: z.string(),
	vehicle_description: z.string().optional(),
	vehicle_modifications: z.string().optional(),
	images: z.array(z.string()),
	status: z.enum(["pending", "approved", "denied"]),
	submitted_at: z.string(),
	reviewed_at: z.string().optional(),
	reviewed_by: z.string().optional(),
	review_notes: z.string().optional(),
	checkout_session_id: z.string().optional(),
	payment_intent_id: z.string().optional(),
	price_id: z.string().optional(),
});

export type VehicleSubmission = z.infer<typeof vehicleSubmissionSchema>;

export const vehicleSubmissionFormSchema = vehicleSubmissionSchema
	.pick({
		event_id: true,
		participant_name: true,
		participant_email: true,
		participant_phone: true,
		vehicle_year: true,
		vehicle_make: true,
		vehicle_model: true,
		vehicle_description: true,
		vehicle_modifications: true,
		price_id: true,
	})
	.extend({
		participant_name: z.string().trim().min(2, "Enter your full name."),
		participant_email: z.email("Enter a valid email address.").trim(),
		participant_phone: z.string().trim().optional(),
		vehicle_year: z
			.string()
			.trim()
			.regex(/^\d{4}$/, "Enter a four-digit year."),
		vehicle_make: z.string().trim().min(2, "Enter the vehicle make."),
		vehicle_model: z.string().trim().min(2, "Enter the vehicle model."),
		vehicle_description: z.string().trim().optional(),
		vehicle_modifications: z.string().optional(),
		event_id: z.string().trim().min(1),
		price_id: z.string().trim().min(1),
	});

export type VehicleSubmissionFormData = z.infer<
	typeof vehicleSubmissionFormSchema
>;

export const eventCheckInSchema = z.object({
	code: z.string().trim().min(1, "Ticket code is required"),
});

export type EventCheckIn = z.infer<typeof eventCheckInSchema>;
