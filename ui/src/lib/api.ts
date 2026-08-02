import { goto } from "$app/navigation";
import { getAdminToken } from "./stores/auth.svelte";

// export const BASE_URL = import.meta.env.VITE_API_URL;
export const BASE_URL = "/api";

/**
 * Custom API error class with additional context
 */
export class ApiError extends Error {
	public readonly status: number;
	public readonly statusText: string;
	public readonly data: unknown;

	constructor(
		message: string,
		status: number,
		statusText: string,
		data?: unknown,
	) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.statusText = statusText;
		this.data = data;
	}

	get isUnauthorized(): boolean {
		return this.status === 401;
	}

	get isForbidden(): boolean {
		return this.status === 403;
	}

	get isNotFound(): boolean {
		return this.status === 404;
	}

	get isServerError(): boolean {
		return this.status >= 500;
	}
}

/**
 * Request configuration options
 */
export interface RequestConfig extends Omit<RequestInit, "body"> {
	body?: unknown;
	params?: Record<string, string | number | boolean | undefined>;
}

/**
 * Build URL with query parameters
 */
function buildUrl(
	endpoint: string,
	params?: Record<string, string | number | boolean | undefined>,
): string {
	// Ensure the endpoint starts with a forward slash
	const normalizedEndpoint = endpoint.startsWith("/")
		? endpoint
		: `/${endpoint}`;

	// Combine base URL and endpoint directly to preserve path components like /v1.0
	const fullUrl = `${BASE_URL}${normalizedEndpoint}`;
	const url = new URL(fullUrl, window.location.origin);

	if (params) {
		Object.entries(params).forEach(([key, value]) => {
			if (value !== undefined) {
				url.searchParams.append(key, String(value));
			}
		});
	}

	return url.toString();
}

function getErrorMessage(data: unknown, fallback: string): string {
	if (typeof data === "string" && data.trim()) {
		return data;
	}

	if (data && typeof data === "object") {
		const record = data as Record<string, unknown>;

		if (typeof record.message === "string" && record.message.trim()) {
			return record.message;
		}

		if (typeof record.error === "string" && record.error.trim()) {
			return record.error;
		}
	}

	return fallback || "Request failed";
}

async function parseResponseBody(response: Response): Promise<unknown> {
	if (response.status === 204) {
		return undefined;
	}

	const text = await response.text();

	if (!text) {
		return undefined;
	}

	try {
		return JSON.parse(text);
	} catch {
		return text;
	}
}

/**
 * Core request function
 */
async function request<T>(
	endpoint: string,
	config: RequestConfig = {},
	fetcher: typeof fetch = fetch,
): Promise<T> {
	const { body, params, headers: customHeaders, ...fetchConfig } = config;

	const url = buildUrl(endpoint, params);

	const isFormData =
		typeof FormData !== "undefined" && body instanceof FormData;

	const token = getAdminToken();

	const headers: HeadersInit = {
		Authorization: `Bearer ${token}`,
		...(isFormData ? {} : { "Content-Type": "application/json" }),
		...customHeaders,
	};

	const response = await fetcher(url, {
		...fetchConfig,
		headers,
		credentials: "include",
		body:
			body === undefined
				? undefined
				: isFormData
					? body
					: JSON.stringify(body),
	});

	const responseData = await parseResponseBody(response);

	if (!response.ok) {
		throw new ApiError(
			getErrorMessage(responseData, response.statusText),
			response.status,
			response.statusText,
			responseData,
		);
	}

	// Parse JSON response
	// try {
	// 	return (await response.json()) as T;
	// } catch {
	// 	return undefined as T;
	// }
	return responseData as T;
}

export interface UploadProgressEvent {
	loaded: number;
	total: number;
	percent: number;
}

function parseXhrResponse(responseText: string): unknown {
	if (!responseText) {
		return undefined;
	}

	try {
		return JSON.parse(responseText);
	} catch {
		return responseText;
	}
}

function uploadFormData<T>(
	endpoint: string,
	body: FormData,
	onProgress?: (progress: UploadProgressEvent) => void,
): Promise<T> {
	return new Promise((resolve, reject) => {
		const xhr = new XMLHttpRequest();

		xhr.open("POST", buildUrl(endpoint), true);

		const token = getAdminToken();

		xhr.setRequestHeader("Authorization", `Bearer ${token}`);

		// Do not set Content-Type manually.
		// XMLHttpRequest automatically adds the multipart boundary.
		xhr.withCredentials = true;

		xhr.upload.onprogress = (event) => {
			if (!event.lengthComputable) {
				return;
			}

			const percent = Math.round((event.loaded / event.total) * 100);

			onProgress?.({
				loaded: event.loaded,
				total: event.total,
				percent,
			});
		};

		xhr.onload = () => {
			const responseData = parseXhrResponse(xhr.responseText);

			if (xhr.status >= 200 && xhr.status < 300) {
				resolve(responseData as T);
				return;
			}

			reject(
				new ApiError(
					getErrorMessage(responseData, xhr.statusText),
					xhr.status,
					xhr.statusText,
					responseData,
				),
			);
		};

		xhr.onerror = () => {
			reject(
				new ApiError(
					"Network error. Please check your connection.",
					0,
					"Network Error",
				),
			);
		};

		xhr.onabort = () => {
			reject(new ApiError("Upload was cancelled.", 0, "Upload Aborted"));
		};

		xhr.send(body);
	});
}

/**
 * API client with typed HTTP methods
 */
export const apiClient = {
	get<T>(
		endpoint: string,
		config?: RequestConfig,
		fetcher: typeof fetch = fetch,
	): Promise<T> {
		return request<T>(endpoint, { ...config, method: "GET" }, fetcher);
	},

	post<T, B = unknown>(
		endpoint: string,
		body?: B,
		config?: RequestConfig,
		fetcher: typeof fetch = fetch,
	): Promise<T> {
		return request<T>(
			endpoint,
			{ ...config, method: "POST", body },
			fetcher,
		);
	},

	upload<T>(
		endpoint: string,
		body: FormData,
		onProgress?: (progress: UploadProgressEvent) => void,
	): Promise<T> {
		return uploadFormData<T>(endpoint, body, onProgress);
	},

	put<T, B = unknown>(
		endpoint: string,
		body?: B,
		config?: RequestConfig,
		fetcher: typeof fetch = fetch,
	): Promise<T> {
		return request<T>(
			endpoint,
			{ ...config, method: "PUT", body },
			fetcher,
		);
	},

	patch<T, B = unknown>(
		endpoint: string,
		body?: B,
		config?: RequestConfig,
		fetcher: typeof fetch = fetch,
	): Promise<T> {
		return request<T>(
			endpoint,
			{ ...config, method: "PATCH", body },
			fetcher,
		);
	},

	delete<T, B = unknown>(
		endpoint: string,
		body?: B,
		config?: RequestConfig,
		fetcher: typeof fetch = fetch,
	): Promise<T> {
		return request<T>(
			endpoint,
			{ ...config, method: "DELETE", body },
			fetcher,
		);
	},
};

/**
 * Generic request function for use with TanStack Query
 * @example
 * const { data } = createQuery({
 *   queryKey: ['posts'],
 *   queryFn: () => apiRequest<Post[]>({ endpoint: '/posts' })
 * });
 */
export async function apiRequest<T>(
	endpoint: string,
	config?: RequestConfig,
	fetcher: typeof fetch = fetch,
): Promise<T> {
	return request<T>(endpoint, config, fetcher);
}

export default apiClient;
