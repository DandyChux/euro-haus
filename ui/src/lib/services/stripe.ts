import apiClient, { apiRequest } from "$lib/api";
import type { StripePrice } from "$lib/schemas/price";
import type {
	BundleItem,
	BundleProduct,
	Product,
	ProductVariant,
	ProductVariants,
	StripeProduct,
} from "$lib/schemas/product";
import type { OrderSummary } from "$lib/schemas/session";
import {
	fetchExternalMetadata,
	isUpcoming,
	parseJsonField,
	parseStringList,
	toFloat,
	toInt,
} from "$lib/utils";

type Fetcher = typeof fetch;

async function request<T>(
	fetcher: Fetcher,
	path: string,
	init?: RequestInit,
): Promise<T> {
	// const url = path.startsWith("/api") ? path : `/api${path}`;
	const response = await apiRequest<T>(path, init, fetcher);

	return response;
}

export async function getStripeProducts(
	fetcher: Fetcher,
	includeInactive = false,
): Promise<StripeProduct[]> {
	const query = includeInactive ? "?include_inactive=true" : "";
	const response = await request<{ products?: StripeProduct[] }>(
		fetcher,
		`/products${query}`,
	);
	return response.products ?? [];
}

export async function getStripeProduct(
	fetcher: Fetcher,
	id: string,
): Promise<StripeProduct | null> {
	try {
		return await request<StripeProduct>(fetcher, `/products/${id}`);
	} catch {
		return null;
	}
}

export async function getProduct(
	fetcher: Fetcher,
	id: string,
): Promise<Product | null> {
	const stripeProduct = await getStripeProduct(fetcher, id);
	if (!stripeProduct) return null;
	return transformStripeProduct(stripeProduct);
}

export function transformStripeProduct(stripeProduct: StripeProduct): Product {
	return {
		id: stripeProduct.id,
		price_id: stripeProduct.default_price?.id,
		title: stripeProduct.name,
		description: stripeProduct.description ?? "",
		price: (stripeProduct.default_price?.unit_amount ?? 0) / 100,
		compare_at_price: toFloat(stripeProduct.metadata.compare_at_price),
		images: stripeProduct.images,
		is_new: stripeProduct.metadata.is_new === "true",
		in_stock:
			stripeProduct.active && stripeProduct.metadata.in_stock !== "false",
		featured: stripeProduct.metadata.featured === "true",
		category: stripeProduct.metadata.category || "merchandise",
		subcategory: stripeProduct.metadata.subcategory || "general",
		tags: parseStringList(stripeProduct.metadata.tags),
		max_quantity: toInt(stripeProduct.metadata.max_quantity),
	};
}

export function transformStripeBundleProduct(
	stripeProduct: StripeProduct,
): BundleProduct {
	const price = (stripeProduct.default_price?.unit_amount ?? 0) / 100;
	const bundle_items = parseJsonField<BundleItem[]>(
		stripeProduct.metadata.bundle_items,
		[],
	);
	const totalValue = toFloat(stripeProduct.metadata.total_value) ?? 0;

	return {
		id: stripeProduct.id,
		price_id: stripeProduct.default_price?.id,
		title: stripeProduct.name,
		description: stripeProduct.description ?? "",
		price,
		images: stripeProduct.images,
		in_stock:
			stripeProduct.active && stripeProduct.metadata.in_stock !== "false",
		featured: stripeProduct.metadata.featured === "true",
		category: "bundle",
		subcategory: "bundle",
		tags: [],
		max_quantity: toInt(stripeProduct.metadata.max_quantity),
		bundle_items: bundle_items,
		discount_type:
			stripeProduct.metadata.discount_type === "fixed"
				? "fixed"
				: "percentage",
		discount_value: toFloat(stripeProduct.metadata.discount_value) ?? 0,
		// savings: totalValue - price,
	};
}

export async function getCatalogProducts(fetcher: Fetcher): Promise<Product[]> {
	const products = await getStripeProducts(fetcher);
	return products
		.filter(
			(product) =>
				product.metadata.type !== "event" &&
				product.metadata.type !== "bundle",
		)
		.map(transformStripeProduct);
}

export async function getFeaturedProducts(fetcher: Fetcher, limit = 3) {
	const products = await getCatalogProducts(fetcher);
	const featured = products.filter((product) => product.featured);
	const fallback = products.filter((product) => !product.featured);
	return [...featured, ...fallback].slice(0, limit);
}

export async function getEventLinkedProducts(
	fetcher: Fetcher,
	eventId: string,
) {
	try {
		const response = await request<{ linkedProducts?: StripeProduct[] }>(
			fetcher,
			`/events/${eventId}/linked-products`,
		);

		return (response.linkedProducts ?? [])
			.filter((product) => product.metadata.type !== "event")
			.map((product) =>
				product.metadata.type === "bundle"
					? transformStripeBundleProduct(product)
					: transformStripeProduct(product),
			);
	} catch {
		return [];
	}
}

export async function getProductWithVariants(
	fetcher: Fetcher,
	productId: string,
): Promise<ProductVariants | null> {
	const rawProduct = await getStripeProduct(fetcher, productId);
	if (!rawProduct || rawProduct.metadata.type !== "product") return null;

	try {
		const response = await request<{ prices?: StripePrice[] }>(
			fetcher,
			`/products/${productId}/prices`,
		);

		const variants: ProductVariant[] = (response.prices ?? []).map(
			(price) => ({
				id: price.id,
				price_id: price.id,
				size: price.metadata.size,
				color: price.metadata.color,
				variant: price.metadata.variant || price.nickname || "Standard",
				price: price.unit_amount / 100,
				in_stock:
					price.metadata.in_stock !== "false" &&
					(!price.metadata.stock_quantity ||
						(toInt(price.metadata.stock_quantity) ?? 0) > 0),
				stock_quantity: toInt(price.metadata.stock_quantity),
				images: parseJsonField<string[]>(price.metadata.images, []),
			}),
		);

		return {
			...transformStripeProduct(rawProduct),
			variants: variants.sort((a, b) => a.price - b.price),
		};
	} catch {
		return null;
	}
}

export async function getBundleProduct(
	fetcher: Fetcher,
	productId: string,
): Promise<BundleProduct | null> {
	const rawProduct = await getStripeProduct(fetcher, productId);
	if (!rawProduct || rawProduct.metadata.type !== "bundle") return null;
	return transformStripeBundleProduct(rawProduct);
}

export async function findBundlesForProduct(
	fetcher: Fetcher,
	productId: string,
) {
	const products = await getStripeProducts(fetcher);
	return products
		.filter((product) => product.metadata.type === "bundle")
		.map(transformStripeBundleProduct)
		.filter((bundle) =>
			bundle.bundle_items.some((item) => item.productId === productId),
		);
}

export async function getCheckoutSession(fetcher: Fetcher, sessionId: string) {
	return request<OrderSummary>(
		fetcher,
		`/checkout-session?session_id=${encodeURIComponent(sessionId)}`,
	);
}

async function saveBundle(productId: string, data: BundleProduct) {
	const payload: Partial<BundleProduct> = {
		id: productId,
		title: data.bundle_items.map((item) => item.productName).join(" + "),
		description: "",
		price: data.bundle_items.reduce(
			(total, item) => total + item.price * item.quantity,
			0,
		),
		in_stock: data.in_stock,
		max_quantity: data.max_quantity ?? undefined,
		bundle_items: data.bundle_items,
		discount_type: data.discount_type,
		discount_value: data.discount_value,
	};

	await apiClient.put(`/admin/products/${productId}`, payload);
}
