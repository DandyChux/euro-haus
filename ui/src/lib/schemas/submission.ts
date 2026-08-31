import { z } from "zod";

export const submissionRequirementTypeSchema = z.enum([
	"text",
	"textarea",
	"select",
	"radio",
	"boolean",
	"number",
]);

export const requirementAnswerSchema = z.union([
	z.string(),
	z.number(),
	z.boolean(),
]);

export const submissionRequirementSchema = z.object({
	id: z.string(),
	price_id: z.string(),
	key: z.string(),
	label: z.string(),
	type: submissionRequirementTypeSchema,
	required: z.boolean(),
	options: z.array(z.string()).default([]),
	sort_order: z.number(),
	active: z.boolean(),
});

export const submissionRequirementAnswerSchema = z.object({
	id: z.string(),
	requirement_id: z.string(),
	key: z.string(),
	label: z.string(),
	type: submissionRequirementTypeSchema,
	value: requirementAnswerSchema,
});

export const vehicleSubmissionSchema = z.object({
	id: z.string(),

	event_id: z.string(),
	event_slug: z.string().optional(),

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
	checkout_created_at: z.string().optional(),
	checkout_completed: z.boolean(),
	checkout_completed_at: z.string().optional(),

	payment_intent_id: z.string().optional(),
	payment_succeeded_before_approval: z.boolean(),
	payment_succeeded_at: z.string().optional(),
	payment_captured: z.boolean(),
	payment_captured_at: z.string().optional(),

	price_id: z.string().optional(),
	price_nickname: z.string().optional(),
	promotion_code: z.string().optional(),

	requires_approval: z.boolean(),
	awaiting_approval: z.boolean(),

	approval_email_sent: z.boolean(),
	approval_email_sent_at: z.string().optional(),
	approval_email_resent: z.boolean(),

	ticket_id: z.string().optional(),
	ticket_created_at: z.string().optional(),
	ticket_email_sent: z.boolean(),
	ticket_email_sent_at: z.string().optional(),

	email_updated_at: z.string().optional(),
	previous_email: z.string().optional(),
	email_resent_count: z.number(),

	recovery_attempts: z.number(),
	recovery_last_sent_at: z.string().optional(),

	refund_id: z.string().optional(),
	refund_amount: z.number(),
	refund_issued_at: z.string().optional(),

	revoked_at: z.string().optional(),
	revoked_by: z.string().optional(),
	revocation_reason: z.string().optional(),

	created_at: z.string(),
	updated_at: z.string(),

	requirement_answers: z
		.array(
			z.object({
				id: z.string(),
				requirement_id: z.string(),
				key: z.string(),
				label: z.string(),
				type: submissionRequirementTypeSchema,
				value: z.unknown(),
			}),
		)
		.optional(),
});

export const issueSubmissionSchema = vehicleSubmissionSchema.extend({
	issues: z.array(z.string()).optional(),
	email_sent: z.boolean().optional(),
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

export type SubmissionRequirementType = z.infer<
	typeof submissionRequirementTypeSchema
>;

export type RequirementAnswer = z.infer<typeof requirementAnswerSchema>;

export type SubmissionRequirement = z.infer<typeof submissionRequirementSchema>;

export type SubmissionRequirementAnswer = z.infer<
	typeof submissionRequirementAnswerSchema
>;

export type IssueSubmission = z.infer<typeof issueSubmissionSchema>;
