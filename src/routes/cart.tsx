import { createFileRoute, Link } from '@tanstack/react-router';
import { Button } from '~/components/ui/button';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '~/components/ui/card';
import { Separator } from '~/components/ui/separator';
import { Input } from '~/components/ui/input';
import { Image } from '~/components/ui/image';
import { Badge } from '~/components/ui/badge';
import { Skeleton } from '~/components/ui/skeleton';
import { useCart } from '~/lib/contexts/cart-context';
import { ShoppingBag, Trash2, Plus, Minus, ArrowRight, Package, Truck, Shield } from 'lucide-react';
import { useState } from 'react';
import { loadStripe } from '@stripe/stripe-js';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';

export const Route = createFileRoute('/cart')({
	component: CartPage,
});

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY || '');

function CartItemSkeleton() {
	return (
		<div className="flex gap-4 p-4 border rounded-lg">
			<Skeleton className="w-24 h-24 rounded-md" />
			<div className="flex-1 space-y-2">
				<Skeleton className="h-5 w-3/4" />
				<Skeleton className="h-4 w-1/2" />
				<Skeleton className="h-4 w-24" />
			</div>
			<div className="flex flex-col items-end space-y-2">
				<Skeleton className="h-4 w-20" />
				<Skeleton className="h-8 w-24" />
			</div>
		</div>
	);
}

function CartPage() {
	const { items, removeItem, updateQuantity, subtotal, clearCart, isLoading } = useCart();
	const [isCheckingOut, setIsCheckingOut] = useState(false);
	const [promoCode, setPromoCode] = useState('');

	// Calculate pricing
	const shipping = subtotal > 75 ? 0 : 9.99;
	const tax = subtotal * 0.08; // 8% tax rate
	const total = subtotal + shipping + tax;

	const handleCheckout = async () => {
		if (items.length === 0) {
			toast.error('Your cart is empty');
			return;
		}

		try {
			setIsCheckingOut(true);

			// Create line items for Stripe checkout
			const lineItems = items.map(item => {
				if (item.priceId) {
					// Use existing price ID if available
					return {
						price: item.priceId,
						quantity: item.quantity,
					};
				} else {
					// Create price data on the fly (fallback)
					return {
						price_data: {
							currency: 'usd',
							product_data: {
								name: item.title,
								description: item.description,
								images: [item.imageUrl],
								metadata: {
									type: item.type || 'product',
									...(item.eventDate && { event_date: item.eventDate }),
								},
							},
							unit_amount: Math.round(item.price * 100),
						},
						quantity: item.quantity,
					};
				}
			});

			// Create checkout session
			const response = await apiClient.post('/create-checkout-session', {
				line_items: lineItems,
				mode: 'payment',
				success_url: `${window.location.origin}/checkout/success?session_id={CHECKOUT_SESSION_ID}`,
				cancel_url: `${window.location.origin}/cart`,
				metadata: {
					cart_items: JSON.stringify(items.map(item => ({
						id: item.id,
						title: item.title,
						quantity: item.quantity,
						type: item.type,
					}))),
				},
			});

			// Redirect to Stripe checkout
			if (response.data.url) {
				window.location.href = response.data.url;
			} else if (response.data.session_id && window.Stripe) {
				const stripe = await stripePromise;
				await stripe?.redirectToCheckout({ sessionId: response.data.session_id });
			}
		} catch (error) {
			console.error('Checkout error:', error);
			toast.error('Failed to initiate checkout. Please try again.');
		} finally {
			setIsCheckingOut(false);
		}
	};

	if (isLoading) {
		return (
			<div className="min-h-screen bg-background">
				<div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
					<Skeleton className="h-10 w-48 mb-8" />
					<div className="grid lg:grid-cols-3 gap-8">
						<div className="lg:col-span-2 space-y-4">
							{[1, 2, 3].map((i) => (
								<CartItemSkeleton key={i} />
							))}
						</div>
						<div className="lg:col-span-1">
							<Card>
								<CardHeader>
									<Skeleton className="h-6 w-32" />
								</CardHeader>
								<CardContent className="space-y-4">
									<Skeleton className="h-20 w-full" />
									<Skeleton className="h-12 w-full" />
								</CardContent>
							</Card>
						</div>
					</div>
				</div>
			</div>
		);
	}

	if (items.length === 0) {
		return (
			<div className="min-h-screen bg-background flex items-center justify-center">
				<div className="text-center space-y-6 p-8">
					<div className="mx-auto w-24 h-24 bg-muted rounded-full flex items-center justify-center">
						<ShoppingBag className="w-12 h-12 text-muted-foreground" />
					</div>
					<h1 className="text-3xl font-bold">Your cart is empty</h1>
					<p className="text-muted-foreground max-w-md mx-auto">
						Looks like you haven't added anything to your cart yet.
						Start shopping to find your perfect Euro Haus gear!
					</p>
					<div className="flex gap-4 justify-center">
						<Button asChild>
							<Link to="/catalog">
								Browse Products
								<ArrowRight className="ml-2 h-4 w-4" />
							</Link>
						</Button>
						<Button asChild variant="outline">
							<Link to="/events">View Events</Link>
						</Button>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-background">
			<div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
				{/* Header */}
				<div className="flex items-center justify-between mb-8">
					<h1 className="text-3xl font-bold">Shopping Cart ({items.length} {items.length === 1 ? 'item' : 'items'})</h1>
					<Button
						variant="ghost"
						size="sm"
						onClick={clearCart}
						className="text-muted-foreground hover:text-foreground"
					>
						Clear Cart
					</Button>
				</div>

				<div className="grid lg:grid-cols-3 gap-8">
					{/* Cart Items */}
					<div className="lg:col-span-2 space-y-4">
						{items.map((item) => (
							<Card key={item.id} className="overflow-hidden">
								<CardContent className="p-4">
									<div className="flex gap-4">
										{/* Product Image */}
										<div className="relative w-24 h-24 flex-shrink-0">
											<Image
												src={item.imageUrl}
												alt={item.title}
												className="w-full h-full object-cover rounded-md"
											/>
										</div>

										{/* Product Details */}
										<div className="flex-1 min-w-0">
											<h3 className="font-semibold text-lg truncate">{item.title}</h3>
											<p className="text-sm text-muted-foreground line-clamp-2">{item.description}</p>
											<p className="text-lg font-bold mt-2">${item.price.toFixed(2)}</p>
										</div>

										{/* Quantity and Actions */}
										<div className="flex flex-col items-end justify-between">
											<Button
												variant="ghost"
												size="icon"
												onClick={() => removeItem(item.id)}
												className="text-muted-foreground hover:text-destructive"
											>
												<Trash2 className="h-4 w-4" />
											</Button>

											<div className="flex items-center gap-2">
												<Button
													variant="outline"
													size="icon"
													className="h-8 w-8"
													onClick={() => updateQuantity(item.id, item.quantity - 1)}
													disabled={item.quantity <= 1}
												>
													<Minus className="h-3 w-3" />
												</Button>
												<Input
													type="number"
													value={item.quantity}
													onChange={(e) => updateQuantity(item.id, parseInt(e.target.value) || 1)}
													className="w-16 h-8 text-center"
													min="1"
													max={item.maxQuantity || 99}
												/>
												<Button
													variant="outline"
													size="icon"
													className="h-8 w-8"
													onClick={() => updateQuantity(item.id, item.quantity + 1)}
													disabled={item.quantity >= (item.maxQuantity || 99)}
												>
													<Plus className="h-3 w-3" />
												</Button>
											</div>
										</div>
									</div>
								</CardContent>
							</Card>
						))}

						{/* Features */}
						<div className="grid sm:grid-cols-3 gap-4 mt-8">
							<div className="flex items-center gap-3 p-4 bg-muted/50 rounded-lg">
								<Truck className="h-5 w-5 text-primary" />
								<div>
									<p className="font-medium text-sm">Free Shipping</p>
									<p className="text-xs text-muted-foreground">On orders over $75</p>
								</div>
							</div>
							<div className="flex items-center gap-3 p-4 bg-muted/50 rounded-lg">
								<Package className="h-5 w-5 text-primary" />
								<div>
									<p className="font-medium text-sm">Easy Returns</p>
									<p className="text-xs text-muted-foreground">30-day return policy</p>
								</div>
							</div>
							<div className="flex items-center gap-3 p-4 bg-muted/50 rounded-lg">
								<Shield className="h-5 w-5 text-primary" />
								<div>
									<p className="font-medium text-sm">Secure Checkout</p>
									<p className="text-xs text-muted-foreground">Powered by Stripe</p>
								</div>
							</div>
						</div>
					</div>

					{/* Order Summary */}
					<div className="lg:col-span-1">
						<Card className="sticky top-24">
							<CardHeader>
								<CardTitle>Order Summary</CardTitle>
							</CardHeader>
							<CardContent className="space-y-4">
								{/* Promo Code */}
								<div className="flex gap-2">
									<Input
										placeholder="Promo code"
										value={promoCode}
										onChange={(e) => setPromoCode(e.target.value)}
									/>
									<Button variant="outline" size="default">
										Apply
									</Button>
								</div>

								<Separator />

								{/* Price Breakdown */}
								<div className="space-y-2">
									<div className="flex justify-between text-sm">
										<span>Subtotal</span>
										<span>${subtotal.toFixed(2)}</span>
									</div>
									<div className="flex justify-between text-sm">
										<span>Shipping</span>
										<span>{shipping === 0 ? 'FREE' : `$${shipping.toFixed(2)}`}</span>
									</div>
									<div className="flex justify-between text-sm">
										<span>Tax</span>
										<span>${tax.toFixed(2)}</span>
									</div>
									{shipping === 0 && (
										<Badge variant="secondary" className="w-full justify-center">
											You qualify for free shipping!
										</Badge>
									)}
									<Separator />
									<div className="flex justify-between font-bold text-lg">
										<span>Total</span>
										<span>${total.toFixed(2)}</span>
									</div>
								</div>
							</CardContent>
							<CardFooter className="flex flex-col gap-2">
								<Button
									className="w-full"
									size="lg"
									onClick={handleCheckout}
									disabled={isCheckingOut}
								>
									{isCheckingOut ? 'Processing...' : 'Proceed to Checkout'}
									<ArrowRight className="ml-2 h-4 w-4" />
								</Button>
								<Button
									variant="outline"
									className="w-full"
									asChild
								>
									<Link to="/catalog">Continue Shopping</Link>
								</Button>
								<p className="text-xs text-center text-muted-foreground mt-2">
									Secure checkout powered by Stripe
								</p>
							</CardFooter>
						</Card>
					</div>
				</div>
			</div>
		</div>
	);
}
