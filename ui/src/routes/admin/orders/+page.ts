import type { PageLoad } from "./$types";
import apiClient from "$lib/api";

export type Fulfillment = {
	id: string;
	order_id: string;
	product_name: string;
	quantity: number;
	customer_email: string;
	customer_name: string;
	shipping_address: string;
	status: string;
	type: string;
	tracking_number?: string;
	tracking_carrier?: string;
	notes?: string;
	created_at: string;
	updated_at?: string;
	shipped_at?: string;
	delivered_at?: string;
};

export const load: PageLoad = async ({ fetch, url }) => {
	const status = url.searchParams.get("status") ?? "";

	try {
		const response = await apiClient.get<{ fulfillments?: Fulfillment[] }>(
			"/admin/fulfillments",
			{ params: { status: status || undefined } },
			fetch,
		);

		return {
			orders: response.fulfillments ?? [],
			status,
			error: "",
		};
	} catch (cause) {
		return {
			orders: [],
			status,
			error:
				cause instanceof Error
					? cause.message
					: "Unable to load orders.",
		};
	}
};
