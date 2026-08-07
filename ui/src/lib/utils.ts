import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export function formatCurrency(amount: number, currency = "USD") {
	return new Intl.NumberFormat("en-US", {
		style: "currency",
		currency,
		maximumFractionDigits: 2,
	}).format(amount);
}

export function formatDate(
	value?: string,
	options: Intl.DateTimeFormatOptions = {
		year: "numeric",
		month: "long",
		day: "numeric",
	},
) {
	if (!value) return "TBA";

	const date = new Date(value);

	if (Number.isNaN(date.getTime())) return value;

	return new Intl.DateTimeFormat("en-US", options).format(date);
}

export function parseJsonField<T>(value: unknown, fallback: T): T {
	if (value == null) return fallback;
	if (typeof value !== "string") return value as T;

	try {
		return JSON.parse(value) as T;
	} catch {
		return fallback;
	}
}

export function parseStringList(value: unknown): string[] {
	if (!value) return [];

	if (Array.isArray(value)) {
		return value.map(String).filter(Boolean);
	}

	if (typeof value !== "string") return [];

	try {
		const parsed = JSON.parse(value);
		return Array.isArray(parsed) ? parsed.map(String).filter(Boolean) : [];
	} catch {
		return value
			.split(",")
			.map((item) => item.trim())
			.filter(Boolean);
	}
}

export function toInt(value: unknown): number | undefined {
	const parsed = Number.parseInt(String(value ?? ""), 10);
	return Number.isFinite(parsed) ? parsed : undefined;
}

export function toFloat(value: unknown): number | undefined {
	const parsed = Number.parseFloat(String(value ?? ""));
	return Number.isFinite(parsed) ? parsed : undefined;
}

export function isUpcoming(date: string) {
	const now = new Date();
	const startOfToday = new Date(now);
	startOfToday.setHours(0, 0, 0, 0);

	return new Date(date) >= startOfToday;
}

export async function fetchExternalMetadata(
	metadata: Record<string, string>,
	fetcher: typeof fetch = fetch,
) {
	const processed: Record<string, unknown> = {};

	for (const [key, value] of Object.entries(metadata)) {
		if (key.endsWith("_external") && value === "true") {
			const field = key.replace("_external", "");
			const urlKey = `${field}_url`;
			const previewKey = `${field}_preview`;

			if (metadata[urlKey]) {
				try {
					const response = await fetcher(metadata[urlKey]);
					if (response.ok) {
						processed[field] = await response.json();
					} else if (metadata[previewKey]) {
						processed[field] = metadata[previewKey];
					}
				} catch {
					if (metadata[previewKey]) {
						processed[field] = metadata[previewKey];
					}
				}
			}
		} else if (
			!key.endsWith("_url") &&
			!key.endsWith("_preview") &&
			!key.endsWith("_external") &&
			!key.endsWith("_truncated")
		) {
			processed[key] = value;
		}
	}

	return processed;
}

/**
 * Generate optimized image URL via Imgproxy
 */
export function getOptimizedImageUrl(
	src: string,
	options?: {
		width?: number | "auto";
		format?: "webp" | "avif";
		quality?: number;
	},
): string {
	const IMGPROXY_URL = "https://img.theeurohaus.com";

	// If no width is provided, default to 0 (Imgproxy keeps original width)
	const width = options?.width === "auto" ? 0 : options?.width || 0;
	const format = options?.format || "webp";
	const quality = options?.quality || 85;

	// Imgproxy processing options:
	// rs:fill:WIDTH:HEIGHT / q:QUALITY / f:FORMAT
	const processingOptions = `rs:fill:${width}:0/q:${quality}/f:${format}`;

	// Imgproxy requires the source URL to be plain text at the end
	return `${IMGPROXY_URL}/insecure/${processingOptions}/plain/${src}`;
}

/**
 * Generate a srcset string for responsive images
 */
export function generateSrcSet(
	src: string,
	widths: number[] = [400, 800, 1200, 1600],
	format: "webp" | "avif" = "webp",
	quality: number = 85,
): string {
	return widths
		.map((width) => {
			const url = getOptimizedImageUrl(src, {
				width,
				format,
				quality,
			});
			return `${url} ${width}w`; // The 'w' tells the browser the true pixel width
		})
		.join(", ");
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChild<T> = T extends { child?: any } ? Omit<T, "child"> : T;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChildren<T> = T extends { children?: any }
	? Omit<T, "children">
	: T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & {
	ref?: U | null;
};
