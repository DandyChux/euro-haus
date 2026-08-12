import { browser } from "$app/environment";
import { toast } from "svelte-sonner";
import type { CartItem } from "$lib/schemas/session";

const STORAGE_KEY = "euro-haus-cart";
const DEFAULT_MAX_QUANTITY = 99;

export type AddToCartInput = Omit<CartItem, "key" | "quantity"> & {
	quantity?: number;
};

type PersistedCartItem = Partial<CartItem> &
	Pick<CartItem, "id" | "title" | "price">;

function createCartKey(item: Pick<CartItem, "id" | "price_id">) {
	return `${item.id}:${item.price_id ?? "default"}`;
}

function clampQuantity(quantity: number, maxQuantity = DEFAULT_MAX_QUANTITY) {
	return Math.max(1, Math.min(quantity, maxQuantity));
}

function normalizeCartItem(item: PersistedCartItem): CartItem {
	const maxQuantity = item.max_quantity ?? DEFAULT_MAX_QUANTITY;
	const quantity = clampQuantity(item.quantity ?? 1, maxQuantity);

	return {
		key: item.key ?? createCartKey(item),
		id: item.id,
		price_id: item.price_id,
		title: item.title,
		description: item.description ?? "",
		price: item.price,
		quantity,
		imageUrl: item.imageUrl,
		max_quantity: item.max_quantity,
		type: item.type,
		event_date: item.event_date,
	};
}

function loadItems(): CartItem[] {
	if (!browser) return [];

	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return [];

		const parsed = JSON.parse(raw);
		if (!Array.isArray(parsed)) return [];

		return parsed
			.filter(
				(item): item is PersistedCartItem =>
					typeof item === "object" &&
					item !== null &&
					typeof item.id === "string" &&
					typeof item.title === "string" &&
					typeof item.price === "number",
			)
			.map(normalizeCartItem);
	} catch (error) {
		console.error("Failed to load cart from storage:", error);
		return [];
	}
}

export const cart = $state({
	items: loadItems(),
	isLoading: false,
});

function persistCart() {
	if (!browser || cart.isLoading) return;
	localStorage.setItem(STORAGE_KEY, JSON.stringify(cart.items));
}

function findItem(identifier: string) {
	return cart.items.find(
		(item) => item.key === identifier || item.id === identifier,
	);
}

export function addToCart(input: AddToCartInput) {
	const key = createCartKey(input);
	const existingItem = cart.items.find((item) => item.key === key);
	const quantityToAdd = Math.max(1, input.quantity ?? 1);

	if (existingItem) {
		const maxQuantity =
			existingItem.max_quantity ??
			input.max_quantity ??
			DEFAULT_MAX_QUANTITY;
		const nextQuantity = existingItem.quantity + quantityToAdd;

		if (nextQuantity > maxQuantity) {
			toast.error(
				`Maximum quantity (${maxQuantity}) reached for ${existingItem.title}`,
			);
			return existingItem;
		}

		existingItem.quantity = nextQuantity;
		persistCart();
		toast.success(
			`Updated ${existingItem.title} quantity to ${nextQuantity}`,
		);
		return existingItem;
	}

	const maxQuantity = input.max_quantity ?? DEFAULT_MAX_QUANTITY;
	const initialQuantity = clampQuantity(quantityToAdd, maxQuantity);

	if (quantityToAdd > maxQuantity) {
		toast.error(
			`Maximum quantity (${maxQuantity}) reached for ${input.title}`,
		);
	}

	const item: CartItem = {
		...input,
		key,
		quantity: initialQuantity,
	};

	cart.items.push(item);
	persistCart();
	toast.success(`Added ${item.title} to cart`);
	return item;
}

export function removeFromCart(identifier: string) {
	const index = cart.items.findIndex(
		(item) => item.key === identifier || item.id === identifier,
	);

	if (index === -1) return;

	const [item] = cart.items.splice(index, 1);
	persistCart();
	toast.success(`Removed ${item.title} from cart`);
}

export function updateCartItemQuantity(identifier: string, quantity: number) {
	const item = findItem(identifier);
	if (!item) return;

	if (quantity < 1) {
		removeFromCart(identifier);
		return;
	}

	const maxQuantity = item.max_quantity ?? DEFAULT_MAX_QUANTITY;
	if (quantity > maxQuantity) {
		toast.error(
			`Maximum quantity (${maxQuantity}) reached for ${item.title}`,
		);
	}

	item.quantity = clampQuantity(quantity, maxQuantity);
	persistCart();
	return item;
}

export function clearCart() {
	if (cart.items.length === 0) return;
	cart.items.length = 0;
	persistCart();
	toast.success("Cart cleared");
}

export function cartTotalItems() {
	return cart.items.reduce((sum, item) => sum + item.quantity, 0);
}

export function cartSubtotal() {
	return cart.items.reduce(
		(sum, item) => sum + item.price * item.quantity,
		0,
	);
}

export const addItem = addToCart;
export const removeItem = removeFromCart;
export const updateQuantity = updateCartItemQuantity;
