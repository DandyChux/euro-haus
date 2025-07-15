import { apiClient } from '../api';
import { Sponsor, SponsorTier } from '../schemas/product-schema';

export interface StripeProduct {
	id: string;
	name: string;
	description: string | null;
	images: string[];
	metadata: Record<string, string>;
	active: boolean;
	default_price: {
		id: string;
		unit_amount: number;
		currency: string;
	} | null;
	created: number;
	updated: number;
}

export interface Product {
	id: string;
	priceId?: string; // Stripe price ID for checkout
	title: string;
	description: string;
	price: number;
	compareAtPrice?: number;
	imageUrl: string;
	isNew?: boolean;
	inStock?: boolean;
	featured?: boolean;
	category?: string;
	maxQuantity?: number;
}

export interface ProductVariant {
	id: string;
	priceId: string;
	size?: string;
	color?: string;
	variant: string;
	price: number;
	inStock: boolean;
	images?: string[];
}

export interface ProductWithVariants extends Product {
	variants: ProductVariant[];
}

export interface EventProduct extends Product {
	slug: string;
	date: string;
	location: string;
	capacity?: number;
	availableSpots?: number;
	organizer?: string;
	tags?: string[];
	status?: 'upcoming' | 'ongoing' | 'completed' | 'cancelled' | 'soldout';
	agenda?: { time: string; activity: string }[];
	includes?: string[];
	sponsors?: Sponsor[];
	sponsorTiers?: SponsorTier[];
	hasTiers?: boolean;
	lowestPrice?: number;
}

export interface TieredPrice {
	id: string;
	priceId: string;
	name: string;
	amount: number;
	currency: string;
	description?: string;
	features?: string[];
	maxQuantity?: number;
	soldOut?: boolean;
}

export interface EventWithTiers extends EventProduct {
	priceTiers: TieredPrice[];
}

export const stripeService = {
	async getAllProducts(): Promise<Product[]> {
		try {
			const response = await apiClient.get<{ products: StripeProduct[] }>('/products');

			if (!response.data.products || response.data.products.length === 0) {
				return [];
			}

			// Filter out event products for regular product listing
			return response.data.products
				.filter(p => p.metadata.type !== 'event')
				.map(this.transformStripeProduct);
		} catch (error) {
			console.error('Failed to fetch products from Stripe:', error);
			throw new Error('Failed to load products');
		}
	},

	async getAllEvents(): Promise<EventProduct[]> {
		try {
			const response = await apiClient.get<{ products: StripeProduct[] }>('/products');

			if (!response.data.products || response.data.products.length === 0) {
				return [];
			}

			// Filter for event products only
			return response.data.products
				.filter(p => p.metadata.type === 'event')
				.map(p => this.transformStripeEventProduct(p))
				.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());
		} catch (error) {
			console.error('Failed to fetch events from Stripe:', error);
			throw new Error('Failed to load events');
		}
	},

	async getEventBySlug(slug: string): Promise<EventProduct | null> {
		try {
			const events = await this.getAllEvents();
			return events.find(event => event.slug === slug) || null;
		} catch (error) {
			console.error('Failed to fetch event:', error);
			return null;
		}
	},

	async getFeaturedProducts(limit: number = 4): Promise<Product[]> {
		try {
			const allProducts = await this.getAllProducts();

			if (allProducts.length === 0) {
				return [];
			}

			const featuredProducts = allProducts.filter(product => product.featured);

			if (featuredProducts.length >= limit) {
				return featuredProducts.slice(0, limit);
			}

			const remainingSlots = limit - featuredProducts.length;
			const otherProducts = allProducts
				.filter(product => !product.featured)
				.slice(0, remainingSlots);

			return [...featuredProducts, ...otherProducts];
		} catch (error) {
			console.error('Failed to fetch featured products:', error);
			throw new Error('Failed to load featured products');
		}
	},

	async getFeaturedEvents(limit: number = 3): Promise<EventProduct[]> {
		try {
			const allEvents = await this.getAllEvents();

			if (allEvents.length === 0) {
				return [];
			}

			// Get featured upcoming events
			const now = new Date();
			const upcomingEvents = allEvents.filter(event =>
				new Date(event.date) > now && event.status !== 'cancelled' && event.status !== 'soldout'
			);

			const featuredEvents = upcomingEvents.filter(event => event.featured);

			if (featuredEvents.length >= limit) {
				return featuredEvents.slice(0, limit);
			}

			// If not enough featured events, supplement with other upcoming events
			const remainingSlots = limit - featuredEvents.length;
			const otherEvents = upcomingEvents
				.filter(event => !event.featured)
				.slice(0, remainingSlots);

			return [...featuredEvents, ...otherEvents];
		} catch (error) {
			console.error('Failed to fetch featured events:', error);
			throw new Error('Failed to load featured events');
		}
	},

	transformStripeProduct(stripeProduct: StripeProduct): Product {
		const price = stripeProduct.default_price?.unit_amount ?
			stripeProduct.default_price.unit_amount / 100 : 0;

		const compareAtPrice = stripeProduct.metadata.compare_at_price ?
			parseFloat(stripeProduct.metadata.compare_at_price) : undefined;

		return {
			id: stripeProduct.id,
			priceId: stripeProduct.default_price?.id,
			title: stripeProduct.name,
			description: stripeProduct.description || '',
			price,
			compareAtPrice,
			imageUrl: stripeProduct.images[0] || '/placeholder.svg?height=400&width=400',
			isNew: stripeProduct.metadata.is_new === 'true',
			inStock: stripeProduct.active && stripeProduct.metadata.in_stock !== 'false',
			featured: stripeProduct.metadata.featured === 'true',
			category: stripeProduct.metadata.category || 'merchandise',
			maxQuantity: stripeProduct.metadata.max_quantity ? parseInt(stripeProduct.metadata.max_quantity) : undefined,
		};
	},

	transformStripeEventProduct(stripeProduct: StripeProduct): EventProduct {
		const baseProduct = this.transformStripeProduct(stripeProduct);

		// Parse event-specific metadata
		const availableSpots = stripeProduct.metadata.available_spots ?
			parseInt(stripeProduct.metadata.available_spots) : undefined;

		const capacity = stripeProduct.metadata.capacity ?
			parseInt(stripeProduct.metadata.capacity) : undefined;

		// Determine status based on available spots and date
		let status: EventProduct['status'] = 'upcoming';
		if (availableSpots === 0) {
			status = 'soldout';
		} else if (stripeProduct.metadata.status) {
			status = stripeProduct.metadata.status as EventProduct['status'];
		}

		return {
			...baseProduct,
			slug: stripeProduct.metadata.slug || stripeProduct.id,
			date: stripeProduct.metadata.event_date || new Date().toISOString(),
			location: stripeProduct.metadata.location || 'TBA',
			capacity,
			availableSpots,
			organizer: stripeProduct.metadata.organizer || 'Euro Haus Events Team',
			tags: stripeProduct.metadata.tags ? JSON.parse(stripeProduct.metadata.tags) : [],
			status,
			agenda: stripeProduct.metadata.agenda ? JSON.parse(stripeProduct.metadata.agenda) : [],
			includes: stripeProduct.metadata.includes ? JSON.parse(stripeProduct.metadata.includes) : [],
			maxQuantity: availableSpots || 10,
			sponsors: stripeProduct.metadata.sponsors ? JSON.parse(stripeProduct.metadata.sponsors) : [],
			sponsorTiers: stripeProduct.metadata.sponsor_tiers ? JSON.parse(stripeProduct.metadata.sponsor_tiers) : [],
			hasTiers: stripeProduct.metadata.has_tiers === 'true', // Add this
			lowestPrice: stripeProduct.metadata.lowest_price ? parseFloat(stripeProduct.metadata.lowest_price) : undefined, // Add this
		};
	},

	async getEventWithPriceTiers(eventId: string): Promise<EventWithTiers | null> {
		try {
			// Fetch the event product
			const eventProduct = await this.getEventBySlug(eventId);
			if (!eventProduct) return null;

			// Fetch all prices for this product
			const response = await apiClient.get<{ prices: any[] }>(`/products/${eventProduct.id}/prices`);

			const priceTiers: TieredPrice[] = response.data.prices.map(price => ({
				id: price.id,
				priceId: price.id,
				name: price.nickname || price.metadata.tier_name || 'Standard',
				amount: price.unit_amount / 100,
				currency: price.currency,
				description: price.metadata.description,
				features: price.metadata.features ? JSON.parse(price.metadata.features) : [],
				maxQuantity: price.metadata.max_quantity ? parseInt(price.metadata.max_quantity) : undefined,
				soldOut: price.metadata.sold_out === 'true'
			}));

			return {
				...eventProduct,
				priceTiers: priceTiers.sort((a, b) => a.amount - b.amount)
			};
		} catch (error) {
			console.error('Failed to fetch event with price tiers:', error);
			return null;
		}
	},

	async getProductWithVariants(productId: string): Promise<ProductWithVariants | null> {
		try {
			// Fetch the main product
			const products = await this.getAllProducts();
			const mainProduct = products.find(p => p.id === productId);
			if (!mainProduct) return null;

			// Fetch all prices for this product
			const response = await apiClient.get<{ prices: any[] }>(`/products/${productId}/prices`);

			const variants: ProductVariant[] = response.data.prices.map(price => ({
				id: price.id,
				priceId: price.id,
				size: price.metadata.size,
				color: price.metadata.color,
				variant: price.metadata.variant || price.nickname || 'Standard',
				price: price.unit_amount / 100,
				inStock: price.metadata.in_stock !== 'false',
				images: price.metadata.images ? JSON.parse(price.metadata.images) : []
			}));

			return {
				...mainProduct,
				variants: variants.sort((a, b) => a.price - b.price)
			};
		} catch (error) {
			console.error('Failed to fetch product variants:', error);
			return null;
		}
	},
};
