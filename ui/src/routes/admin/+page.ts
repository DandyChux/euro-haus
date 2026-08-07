import type { PageLoad } from "./$types";
import apiClient from "$lib/api";
import type { Product } from "$lib/types";

type CouponSummary = { valid?: boolean };

export const load: PageLoad = async ({ fetch }) => {
	let stats = {
		totalProducts: 0,
		totalEvents: 0,
		featuredItems: 0,
		mediaFiles: 0,
		pendingSubmissions: 0,
		activeCoupons: 0,
		isLoading: true,
		error: "",
	};

	try {
		const [productsRes, mediaRes, pendingRes, couponRes] =
			await Promise.all([
				apiClient.get<{ products?: Product[] }>(
					"/products?include_inactive=true",
					{},
					fetch,
				),
				apiClient.get<{ files?: string[] }>("/media", {}, fetch),
				apiClient.get<{ count?: number }>(
					"/admin/submissions/pending-count",
					{},
					fetch,
				),
				apiClient.get<{ coupons?: CouponSummary[] }>(
					"/admin/coupons",
					{},
					fetch,
				),
			]);

		const products = productsRes.products ?? [];
		const media = mediaRes.files ?? [];
		const coupons = couponRes.coupons ?? [];

		stats.totalProducts = products.filter(
			(item: any) => item.metadata?.type !== "event",
		).length;
		stats.totalEvents = products.filter(
			(item: any) => item.metadata?.type === "event",
		).length;
		stats.featuredItems = products.filter(
			(item: any) => item.metadata?.featured === "true",
		).length;
		stats.mediaFiles = media.length;
		stats.pendingSubmissions = pendingRes.count ?? 0;
		stats.activeCoupons = coupons.filter((coupon) => coupon.valid).length;
	} catch (error) {
		stats.error =
			error instanceof Error
				? error.message
				: "Unable to load dashboard stats.";
	} finally {
		stats.isLoading = false;
	}

	return {
		stats,
	};
};
