import apiClient from "./api";
import {
	adminAuth,
	clearAdminSession,
	storeAdminSession,
} from "./stores/auth.svelte";

export async function validateAdminSession() {
	if (!adminAuth.token) {
		adminAuth.isAuthenticated = false;
		return false;
	}

	adminAuth.checking = true;
	adminAuth.lastError = "";

	try {
		const payload = await apiClient.get<{
			valid: boolean;
			expires_at?: number;
			error?: string;
		}>("/auth/validate", {
			headers: {
				Authorization: `Bearer ${adminAuth.token}`,
			},
		});

		if (!payload.valid) {
			adminAuth.lastError = payload.error ?? "Session expired";
			clearAdminSession();
			return false;
		}

		adminAuth.isAuthenticated = true;
		adminAuth.expiresAt = payload.expires_at ?? 0;
		return true;
	} catch (error) {
		adminAuth.lastError =
			error instanceof Error
				? error.message
				: "Unable to validate session";
		clearAdminSession();
		return false;
	} finally {
		adminAuth.checking = false;
	}
}

export async function loginAdmin(email: string, password: string) {
	adminAuth.checking = true;
	adminAuth.lastError = "";

	try {
		const body = {
			email: email.trim().toLowerCase(),
			password,
		};
		const payload = await apiClient.post<{
			success?: boolean;
			token?: string;
			message?: string;
		}>("/auth/login", body, {
			headers: {
				"content-type": "application/json",
			},
		});

		if (!payload.success || !payload.token) {
			throw new Error(payload.message ?? "Invalid email or password");
		}

		storeAdminSession(payload.token);

		const valid = await validateAdminSession();
		if (!valid) {
			throw new Error("The account does not have administrator access");
		}

		return true;
	} catch (error) {
		adminAuth.lastError =
			error instanceof Error ? error.message : "Unable to sign in";

		clearAdminSession();
		return false;
	} finally {
		adminAuth.checking = false;
	}
}

export async function logoutAdmin() {
	try {
		if (adminAuth.token) {
			await apiClient.post("/auth/logout", null, {
				headers: {
					authorization: `Bearer ${adminAuth.token}`,
				},
			});
		}
	} catch {
		// ignore logout transport failures
	} finally {
		clearAdminSession();
	}
}
