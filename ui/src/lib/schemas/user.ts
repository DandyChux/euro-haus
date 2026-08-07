// ui/src/lib/schemas/admin.ts
import { z } from "zod";

export const createUserSchema = z.object({
	name: z.string().trim().min(1, "Name is required"),
	email: z.email("Enter a valid email address").trim(),
	password: z.string().min(12, "Password must be at least 12 characters"),
});

export type CreateUser = z.infer<typeof createUserSchema>;
