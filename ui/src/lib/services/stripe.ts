import apiClient, { apiRequest } from "$lib/api";
import type { Price } from "$lib/schemas/price";
import type {
	BundleProduct,
	Product,
	ProductVariant,
	ProductVariants,
} from "$lib/schemas/product";
import type { OrderSummary } from "$lib/schemas/session";

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

export async function getProducts(
	fetcher: Fetcher,
	includeInactive = false,
): Promise<Product[]> {
	const query = includeInactive ? "?include_inactive=true" : "";

	const response = await request<Product[]>(fetcher, `/products${query}`);

	return response ?? [];
}

export async function getProduct(
	fetcher: Fetcher,
	id: string,
): Promise<Product | null> {
	try {
		return await request<Product>(fetcher, `/products/${id}`);
	} catch {
		return null;
	}
}

export async function getBundleProducts(
	fetcher: Fetcher,
	includeInactive = false,
): Promise<BundleProduct[]> {
	const query = includeInactive ? "?include_inactive=true" : "";

	const response = await request<{ products?: BundleProduct[] }>(
		fetcher,
		`/products/bundles${query}`,
	);

	return response.products ?? [];
}

export function toBundleProduct(product: Product): BundleProduct {
	return {
		...product,
		bundle_items: product.bundle_items ?? [],
		discount_type: product.discount_type ?? "percentage",
		discount_value: product.discount_value ?? 0,
	};
}
toBundleProduct;

export async function getCatalogProducts(fetcher: Fetcher): Promise<Product[]> {
	const products = await getProducts(fetcher);

	return products.filter(
		(product) => product.type !== "event" && product.type !== "bundle",
	);
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
): Promise<Product[]> {
	try {
		const response = await request<{ linkedProducts?: Product[] }>(
			fetcher,
			`/events/${eventId}/linked-products`,
		);

		return (response.linkedProducts ?? [])
			.filter((product) => product.type !== "event")
			.map((product) =>
				product.type === "bundle" ? toBundleProduct(product) : product,
			);
	} catch {
		return [];
	}
}

export async function getProductWithVariants(
	fetcher: Fetcher,
	productId: string,
): Promise<ProductVariants | null> {
	const product = await getProduct(fetcher, productId);
	if (!product || product.type !== "product") return null;

	try {
		const response = await request<{ prices?: Price[] }>(
			fetcher,
			`/products/${productId}/prices`,
		);

		const variants: ProductVariant[] = (response.prices ?? []).map(
			(price) => ({
				id: price.id,
				price_id: price.id,
				size: price.size,
				color: price.color,
				variant: price.nickname || "Standard",
				price: price.unit_amount / 100,
				in_stock: price.active && !price.sold_out,
				stock_quantity: price.stock_quantity ?? undefined,
				images: [],
			}),
		);

		return {
			...product,
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
	try {
		return await request<BundleProduct>(fetcher, `/products/${productId}`);
	} catch {
		return null;
	}
}

export async function findBundlesForProduct(
	fetcher: Fetcher,
	productId: string,
): Promise<BundleProduct[]> {
	const bundles = await getBundleProducts(fetcher);

	return bundles.filter((bundle) =>
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
		name: data.bundle_items.map((item) => item.productName).join(" + "),
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
