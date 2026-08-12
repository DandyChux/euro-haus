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

const issueSubmissionSchema = z.object({
	id: z.string(),
	event_id: z.string(),
	event_slug: z.string(),
	participant_name: z.string(),
	participant_email: z.string(),
	vehicle_year: z.string(),
	vehicle_make: z.string(),
	vehicle_model: z.string(),
	status: z.string(),
	submitted_at: z.string(),
	issues: z.array(z.string()).optional(),
	email_sent: z.boolean().optional(),
	checkout_session_id: z.string().optional(),
	payment_intent_id: z.string().optional(),
	ticket_id: z.string().optional(),
});

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
	price_nickname: z.string().optional(),
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
