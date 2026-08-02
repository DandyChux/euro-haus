import { browser } from "$app/environment";

const STORAGE_KEY = "euro-haus-admin-token";

export const adminAuth = $state({
	token: "",
	isAuthenticated: false,
	checking: false,
	expiresAt: 0,
	lastError: "",
});

export function restoreAdminSession() {
	if (!browser) return;

	adminAuth.token = localStorage.getItem(STORAGE_KEY) ?? "";
	adminAuth.isAuthenticated = Boolean(adminAuth.token);
}

export function getAdminToken(): string {
	if (browser && !adminAuth.token) {
		restoreAdminSession();
	}

	return adminAuth.token.trim();
}

export function storeAdminSession(token: string) {
	adminAuth.token = token;
	adminAuth.isAuthenticated = true;
	adminAuth.lastError = "";

	if (browser) {
		localStorage.setItem(STORAGE_KEY, token);
	}
}

export function clearAdminSession() {
	adminAuth.token = "";
	adminAuth.isAuthenticated = false;
	adminAuth.expiresAt = 0;

	if (browser) {
		localStorage.removeItem(STORAGE_KEY);
	}
}
