import React, { createContext, useContext, useState, useEffect } from 'react';
import { toast } from 'sonner';

export interface CartItem {
	id: string;
	priceId?: string; // Stripe price ID for checkout
	title: string;
	description: string;
	price: number;
	quantity: number;
	imageUrl: string;
	maxQuantity?: number;
	type?: 'product' | 'event' | 'bundle'; // To distinguish between products and events
	eventDate?: string; // For event tickets
	metadata?: Record<string, any>;
}

interface CartContextType {
	items: CartItem[];
	addItem: (item: Omit<CartItem, 'quantity'> & { quantity?: number }) => void;
	removeItem: (id: string) => void;
	updateQuantity: (id: string, quantity: number) => void;
	clearCart: () => void;
	totalItems: number;
	subtotal: number;
	isLoading: boolean;
}

const CartContext = createContext<CartContextType | undefined>(undefined);

const CART_STORAGE_KEY = 'euro-haus-cart';

export function CartProvider({ children }: { children: React.ReactNode }) {
	const [items, setItems] = useState<CartItem[]>([]);
	const [isLoading, setIsLoading] = useState(true);

	// Load cart from localStorage on mount
	useEffect(() => {
		try {
			const savedCart = localStorage.getItem(CART_STORAGE_KEY);
			if (savedCart) {
				setItems(JSON.parse(savedCart));
			}
		} catch (error) {
			console.error('Failed to load cart from storage:', error);
		} finally {
			setIsLoading(false);
		}
	}, []);

	// Save cart to localStorage whenever it changes
	useEffect(() => {
		if (!isLoading) {
			localStorage.setItem(CART_STORAGE_KEY, JSON.stringify(items));
		}
	}, [items, isLoading]);

	const addItem = (newItem: Omit<CartItem, 'quantity'> & { quantity?: number }) => {
		setItems(currentItems => {
			const existingItem = currentItems.find(item => item.id === newItem.id);

			if (existingItem) {
				// Update quantity if item already exists
				const newQuantity = existingItem.quantity + (newItem.quantity || 1);
				const maxQuantity = existingItem.maxQuantity || 99;

				if (newQuantity > maxQuantity) {
					toast.error(`Maximum quantity (${maxQuantity}) reached for ${existingItem.title}`);
					return currentItems;
				}

				toast.success(`Updated ${existingItem.title} quantity to ${newQuantity}`);
				return currentItems.map(item =>
					item.id === newItem.id
						? { ...item, quantity: newQuantity }
						: item
				);
			} else {
				// Add new item
				toast.success(`Added ${newItem.title} to cart`);
				return [...currentItems, { ...newItem, quantity: newItem.quantity || 1 }];
			}
		});
	};

	const removeItem = (id: string) => {
		setItems(currentItems => {
			const item = currentItems.find(item => item.id === id);
			if (item) {
				toast.success(`Removed ${item.title} from cart`);
			}
			return currentItems.filter(item => item.id !== id);
		});
	};

	const updateQuantity = (id: string, quantity: number) => {
		if (quantity < 1) {
			removeItem(id);
			return;
		}

		setItems(currentItems =>
			currentItems.map(item => {
				if (item.id === id) {
					const maxQuantity = item.maxQuantity || 99;
					const newQuantity = Math.min(quantity, maxQuantity);

					if (quantity > maxQuantity) {
						toast.error(`Maximum quantity (${maxQuantity}) reached for ${item.title}`);
					}

					return { ...item, quantity: newQuantity };
				}
				return item;
			})
		);
	};

	const clearCart = () => {
		setItems([]);
		toast.success('Cart cleared');
	};

	const totalItems = items.reduce((sum, item) => sum + item.quantity, 0);
	const subtotal = items.reduce((sum, item) => sum + (item.price * item.quantity), 0);

	return (
		<CartContext.Provider
			value={{
				items,
				addItem,
				removeItem,
				updateQuantity,
				clearCart,
				totalItems,
				subtotal,
				isLoading,
			}}
		>
			{children}
		</CartContext.Provider>
	);
}

export function useCart() {
	const context = useContext(CartContext);
	if (!context) {
		throw new Error('useCart must be used within a CartProvider');
	}
	return context;
}
