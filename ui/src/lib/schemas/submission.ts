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

export type SubmissionRequirementType = z.infer<
	typeof submissionRequirementTypeSchema
>;

export type RequirementAnswer = z.infer<typeof requirementAnswerSchema>;

export type SubmissionRequirement = z.infer<typeof submissionRequirementSchema>;

export type SubmissionRequirementAnswer = z.infer<
	typeof submissionRequirementAnswerSchema
>;
