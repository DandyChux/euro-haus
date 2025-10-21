// Updated euro-haus/src/routes/cart.tsx
import { createFileRoute, Link } from '@tanstack/react-router';
import { Button } from '~/components/ui/button';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '~/components/ui/card';
import { Separator } from '~/components/ui/separator';
import { Input } from '~/components/ui/input';
import { Image } from '~/components/ui/image';
import { Badge } from '~/components/ui/badge';
import { Skeleton } from '~/components/ui/skeleton';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { RadioGroup, RadioGroupItem } from '~/components/ui/radio-group';
import { Label } from '~/components/ui/label';
import { useCart } from '~/lib/contexts/cart-context';
import { ShoppingBag, Trash2, Plus, Minus, ArrowRight, Package, Truck, Shield, MapPin, Clock } from 'lucide-react';
import { useState, useEffect, useCallback } from 'react';
import { loadStripe } from '@stripe/stripe-js';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';

export const Route = createFileRoute('/cart')({
	component: CartPage,
});

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY || '');

interface ShippingRate {
	id: string;
	display_name: string;
	amount: number;
	currency: string;
	metadata?: {
		delivery_days?: string;
		eligible?: string;
	};
}

interface TaxCalculationRequest {
	line_items: Array<{ amount: number; reference: string; tax_code: string }>;
	currency: string;
	shipping_amount: number;
	address?: {
		country: string;
		state: string;
		postal_code: string;
	};
}

function CartItemSkeleton() {
	return (
		<div className="flex flex-col sm:flex-row gap-3 sm:gap-4 p-3 sm:p-4 border rounded-lg">
			<Skeleton className="w-full sm:w-24 h-24 rounded-md" />
			<div className="flex-1 space-y-2">
				<Skeleton className="h-5 w-3/4" />
				<Skeleton className="h-4 w-1/2" />
				<Skeleton className="h-4 w-24" />
			</div>
			<div className="flex flex-row sm:flex-col items-center sm:items-end justify-between sm:justify-start space-y-0 sm:space-y-2 mt-3 sm:mt-0">
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
	const [isCalculatingTax, setIsCalculatingTax] = useState(false);
	const [taxAmount, setTaxAmount] = useState<number>(0);
	const [shippingRates, setShippingRates] = useState<ShippingRate[]>([]);
	const [selectedShippingRate, setSelectedShippingRate] = useState<string>('');
	const [isLoadingShippingRates, setIsLoadingShippingRates] = useState(false);

	// Simple address collection for tax calculation
	const [shippingAddress, setShippingAddress] = useState({
		country: 'US',
		state: '',
		postal_code: ''
	});

	// Fetch shipping rates when cart changes
	const fetchShippingRates = useCallback(async () => {
		try {
			setIsLoadingShippingRates(true);
			const response = await apiClient.get('/shipping-rates', {
				params: {
					country: shippingAddress.country,
					subtotal: Math.round(subtotal * 100) // Convert to cents
				}
			});

			const rates = response.data as ShippingRate[];
			setShippingRates(rates);

			// Auto-select the first (best) rate
			if (rates.length > 0 && !selectedShippingRate) {
				setSelectedShippingRate(rates[0].id);
			}
		} catch (error) {
			console.error('Error fetching shipping rates:', error);
			// Fallback rates
			const fallbackRates: ShippingRate[] = subtotal >= 75
				? [
					{ id: 'free_standard', display_name: 'FREE Standard Shipping (5-7 business days)', amount: 0, currency: 'usd' },
					{ id: 'express', display_name: 'Express Shipping (2-3 business days)', amount: 1999, currency: 'usd' }
				]
				: [
					{ id: 'standard', display_name: 'Standard Shipping (5-7 business days)', amount: 999, currency: 'usd' },
					{ id: 'express', display_name: 'Express Shipping (2-3 business days)', amount: 1999, currency: 'usd' }
				];
			setShippingRates(fallbackRates);
			if (!selectedShippingRate && fallbackRates.length > 0) {
				setSelectedShippingRate(fallbackRates[0].id);
			}
		} finally {
			setIsLoadingShippingRates(false);
		}
	}, [shippingAddress.country, subtotal, selectedShippingRate]);

	// Calculate tax when cart or address changes
	const calculateTax = useCallback(async () => {
		if (items.length === 0) {
			setTaxAmount(0);
			return;
		}

		// Get selected shipping amount
		const selectedRate = shippingRates.find(rate => rate.id === selectedShippingRate);
		const shippingAmount = selectedRate?.amount || 0;

		try {
			setIsCalculatingTax(true);

			// Prepare line items for tax calculation
			const lineItems = items.map(item => ({
				amount: Math.round(item.price * item.quantity * 100),
				reference: item.id,
				tax_code: item.type === 'event' ? 'txcd_10000000' : 'txcd_99999999'
			}));

			const requestData: TaxCalculationRequest = {
				line_items: lineItems,
				currency: 'usd',
				shipping_amount: shippingAmount // Include shipping in tax calculation
			};

			// Only include address if we have enough information
			if (shippingAddress.postal_code && shippingAddress.state) {
				requestData.address = {
					country: shippingAddress.country,
					state: shippingAddress.state,
					postal_code: shippingAddress.postal_code
				};
			}

			const response = await apiClient.post('/calculate-tax-shipping', requestData);

			// Use only the tax amount from the response, not shipping (we handle that separately)
			setTaxAmount(response.data.tax_amount / 100); // Convert from cents
		} catch (error) {
			console.error('Error calculating tax:', error);
			// Don't fall back to a fixed percentage - show 0 if we can't calculate
			// This prevents showing incorrect estimates
			setTaxAmount(0);
		} finally {
			setIsCalculatingTax(false);
		}
	}, [items, shippingRates, selectedShippingRate, shippingAddress]);

	useEffect(() => {
		if (items.length > 0) {
			fetchShippingRates();
		}
	}, [items.length, subtotal, fetchShippingRates]);

	useEffect(() => {
		if (items.length > 0 && shippingAddress.postal_code && shippingAddress.state) {
			calculateTax();
		}
	}, [items.length, shippingAddress.postal_code, shippingAddress.state, selectedShippingRate, calculateTax]);

	// Calculate totals based on current selections
	const selectedShippingAmount = shippingRates.find(rate => rate.id === selectedShippingRate)?.amount || 0;
	const shipping = selectedShippingAmount / 100; // Convert from cents
	const tax = taxAmount;
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
						// Add tax code for each item
						tax_code: item.type === 'event' ? 'txcd_10000000' : 'txcd_99999999'
					};
				}
			});

			// Find selected shipping rate details
			const selectedRate = shippingRates.find(rate => rate.id === selectedShippingRate);

			// Create checkout session with automatic tax and selected shipping
			const response = await apiClient.post('/create-checkout-session', {
				line_items: lineItems,
				mode: 'payment',
				success_url: `${window.location.origin}/checkout/success?session_id={CHECKOUT_SESSION_ID}`,
				cancel_url: `${window.location.origin}/cart`,
				allow_promotion_codes: true,
				metadata: {
					cart_items: JSON.stringify(items.map(item => ({
						id: item.id,
						title: item.title,
						quantity: item.quantity,
						type: item.type,
					}))),
					...(selectedShippingRate && selectedRate && {
						selected_shipping_rate: selectedShippingRate,
						shipping_amount: selectedRate.amount.toString()
					})
				},
				// Pass pre-selected shipping info if available
				...(shippingAddress.postal_code && shippingAddress.state && {
					customer_address: {
						country: shippingAddress.country,
						state: shippingAddress.state,
						postal_code: shippingAddress.postal_code
					}
				})
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
				<div className="max-w-7xl mx-auto px-4 py-8 sm:py-12">
					<Skeleton className="h-8 sm:h-10 w-32 sm:w-48 mb-6 sm:mb-8" />
					<div className="grid lg:grid-cols-3 gap-6 lg:gap-8">
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
			<div className="min-h-screen bg-background flex items-center justify-center px-4">
				<div className="text-center space-y-4 sm:space-y-6 p-6 sm:p-8 max-w-md w-full">
					<div className="mx-auto w-20 h-20 sm:w-24 sm:h-24 bg-muted rounded-full flex items-center justify-center">
						<ShoppingBag className="w-10 h-10 sm:w-12 sm:h-12 text-muted-foreground" />
					</div>
					<h1 className="text-2xl sm:text-3xl font-bold">Your cart is empty</h1>
					<p className="text-sm sm:text-base text-muted-foreground">
						Looks like you haven't added anything to your cart yet.
						Start shopping to find your perfect Euro Haus gear!
					</p>
					<div className="flex flex-col sm:flex-row gap-3 sm:gap-4 justify-center">
						<Button asChild className="w-full sm:w-auto">
							<Link to="/catalog">
								Browse Products
								<ArrowRight className="ml-2 h-4 w-4" />
							</Link>
						</Button>
						<Button asChild variant="outline" className="w-full sm:w-auto">
							<Link to="/events">View Events</Link>
						</Button>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-background">
			<div className="max-w-7xl mx-auto px-4 py-6 sm:py-8 lg:py-12">
				{/* Header */}
				<div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6 sm:mb-8">
					<h1 className="text-2xl sm:text-3xl font-bold">
						Shopping Cart
						<span className="text-base sm:text-lg font-normal text-muted-foreground ml-2">
							({items.length} {items.length === 1 ? 'item' : 'items'})
						</span>
					</h1>
					<Button
						variant="ghost"
						size="sm"
						onClick={clearCart}
						className="text-muted-foreground hover:text-foreground self-start sm:self-auto"
					>
						Clear Cart
					</Button>
				</div>

				<div className="grid lg:grid-cols-3 gap-6 lg:gap-8">
					{/* Cart Items */}
					<div className="lg:col-span-2 space-y-4">
						{items.map((item) => (
							<Card key={item.id} className="overflow-hidden">
								<CardContent className="p-3 sm:p-4">
									<div className="flex flex-col sm:flex-row gap-3 sm:gap-4">
										{/* Product Image */}
										<div className="relative w-full sm:w-24 h-32 sm:h-24 flex-shrink-0">
											<Image
												src={item.imageUrl}
												alt={item.title}
												className="w-full h-full object-cover rounded-md"
											/>
										</div>

										{/* Product Details */}
										<div className="flex-1 min-w-0 space-y-1">
											<h3 className="font-semibold text-base sm:text-lg line-clamp-2">{item.title}</h3>
											<p className="text-xs sm:text-sm text-muted-foreground line-clamp-2">{item.description}</p>
											<p className="text-lg font-bold">${item.price.toFixed(2)}</p>
										</div>

										{/* Quantity and Actions - Mobile */}
										<div className="flex items-center justify-between sm:hidden mt-2">
											<div className="flex items-center gap-2">
												<Button
													variant="outline"
													size="icon"
													className="h-7 w-7"
													onClick={() => updateQuantity(item.id, item.quantity - 1)}
													disabled={item.quantity <= 1}
												>
													<Minus className="h-3 w-3" />
												</Button>
												<span className="w-8 text-center text-sm">{item.quantity}</span>
												<Button
													variant="outline"
													size="icon"
													className="h-7 w-7"
													onClick={() => updateQuantity(item.id, item.quantity + 1)}
													disabled={item.quantity >= (item.maxQuantity || 99)}
												>
													<Plus className="h-3 w-3" />
												</Button>
											</div>
											<Button
												variant="ghost"
												size="sm"
												onClick={() => removeItem(item.id)}
												className="text-muted-foreground hover:text-destructive"
											>
												Remove
											</Button>
										</div>

										{/* Quantity and Actions - Desktop */}
										<div className="hidden sm:flex flex-col items-end justify-between">
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
						<div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 mt-6 sm:mt-8">
							<div className="flex items-center gap-3 p-3 sm:p-4 bg-muted/50 rounded-lg">
								<Truck className="h-5 w-5 text-primary flex-shrink-0" />
								<div>
									<p className="font-medium text-sm">Free Shipping</p>
									<p className="text-xs text-muted-foreground">On orders over $75</p>
								</div>
							</div>
							<div className="flex items-center gap-3 p-3 sm:p-4 bg-muted/50 rounded-lg">
								<Package className="h-5 w-5 text-primary flex-shrink-0" />
								<div>
									<p className="font-medium text-sm">Easy Returns</p>
									<p className="text-xs text-muted-foreground">30-day return policy</p>
								</div>
							</div>
							<div className="flex items-center gap-3 p-3 sm:p-4 bg-muted/50 rounded-lg">
								<Shield className="h-5 w-5 text-primary flex-shrink-0" />
								<div>
									<p className="font-medium text-sm">Secure Checkout</p>
									<p className="text-xs text-muted-foreground">Powered by Stripe</p>
								</div>
							</div>
						</div>
					</div>

					{/* Order Summary */}
					<div className="lg:col-span-1">
						<Card className="sticky top-4 lg:top-24">
							<CardHeader className='pb-3 sm:pb-6'>
								<CardTitle className='text-lg sm:text-xl'>Order Summary</CardTitle>
							</CardHeader>
							<CardContent className="space-y-4">
								{/* Shipping Address for Tax Calculation */}
								<div className="space-y-2">
									<div className="flex items-center gap-2 text-sm font-medium">
										<MapPin className="h-4 w-4" />
										<span>Delivery Location</span>
									</div>
									<div className="grid grid-cols-2 gap-2">
										<Select
											value={shippingAddress.state}
											onValueChange={(value) => setShippingAddress({ ...shippingAddress, state: value })}
										>
											<SelectTrigger className='h-9 text-sm'>
												<SelectValue placeholder="State" />
											</SelectTrigger>
											<SelectContent>
												<SelectItem value="AL">Alabama</SelectItem>
												<SelectItem value="AK">Alaska</SelectItem>
												<SelectItem value="AZ">Arizona</SelectItem>
												<SelectItem value="AR">Arkansas</SelectItem>
												<SelectItem value="CA">California</SelectItem>
												<SelectItem value="CO">Colorado</SelectItem>
												<SelectItem value="CT">Connecticut</SelectItem>
												<SelectItem value="DE">Delaware</SelectItem>
												<SelectItem value="FL">Florida</SelectItem>
												<SelectItem value="GA">Georgia</SelectItem>
												<SelectItem value="HI">Hawaii</SelectItem>
												<SelectItem value="ID">Idaho</SelectItem>
												<SelectItem value="IL">Illinois</SelectItem>
												<SelectItem value="IN">Indiana</SelectItem>
												<SelectItem value="IA">Iowa</SelectItem>
												<SelectItem value="KS">Kansas</SelectItem>
												<SelectItem value="KY">Kentucky</SelectItem>
												<SelectItem value="LA">Louisiana</SelectItem>
												<SelectItem value="ME">Maine</SelectItem>
												<SelectItem value="MD">Maryland</SelectItem>
												<SelectItem value="MA">Massachusetts</SelectItem>
												<SelectItem value="MI">Michigan</SelectItem>
												<SelectItem value="MN">Minnesota</SelectItem>
												<SelectItem value="MS">Mississippi</SelectItem>
												<SelectItem value="MO">Missouri</SelectItem>
												<SelectItem value="MT">Montana</SelectItem>
												<SelectItem value="NE">Nebraska</SelectItem>
												<SelectItem value="NV">Nevada</SelectItem>
												<SelectItem value="NH">New Hampshire</SelectItem>
												<SelectItem value="NJ">New Jersey</SelectItem>
												<SelectItem value="NM">New Mexico</SelectItem>
												<SelectItem value="NY">New York</SelectItem>
												<SelectItem value="NC">North Carolina</SelectItem>
												<SelectItem value="ND">North Dakota</SelectItem>
												<SelectItem value="OH">Ohio</SelectItem>
												<SelectItem value="OK">Oklahoma</SelectItem>
												<SelectItem value="OR">Oregon</SelectItem>
												<SelectItem value="PA">Pennsylvania</SelectItem>
												<SelectItem value="RI">Rhode Island</SelectItem>
												<SelectItem value="SC">South Carolina</SelectItem>
												<SelectItem value="SD">South Dakota</SelectItem>
												<SelectItem value="TN">Tennessee</SelectItem>
												<SelectItem value="TX">Texas</SelectItem>
												<SelectItem value="UT">Utah</SelectItem>
												<SelectItem value="VT">Vermont</SelectItem>
												<SelectItem value="VA">Virginia</SelectItem>
												<SelectItem value="WA">Washington</SelectItem>
												<SelectItem value="WV">West Virginia</SelectItem>
												<SelectItem value="WI">Wisconsin</SelectItem>
												<SelectItem value="WY">Wyoming</SelectItem>
											</SelectContent>
										</Select>
										<Input
											placeholder="ZIP Code"
											value={shippingAddress.postal_code}
											onChange={(e) => setShippingAddress({ ...shippingAddress, postal_code: e.target.value.replace(/\D/g, '').slice(0, 5) })}
											maxLength={5}
											className='h-9 text-sm'
										/>
									</div>
									{!shippingAddress.state || !shippingAddress.postal_code ? (
										<p className="text-xs text-muted-foreground">
											Enter location for accurate tax & shipping
										</p>
									) : null}
								</div>

								{/* Shipping Method Selection */}
								{shippingRates.length > 0 && (
									<div className="space-y-2">
										<div className="flex items-center gap-2 text-sm font-medium">
											<Truck className="h-4 w-4" />
											<span>Shipping Method</span>
											<span className="text-xs font-normal text-muted-foreground">(Preview - select at checkout)</span>
										</div>
										{isLoadingShippingRates ? (
											<Skeleton className="h-20 w-full" />
										) : (
											<RadioGroup value={selectedShippingRate} onValueChange={setSelectedShippingRate} className="space-y-2">
												{shippingRates.map(rate => (
													<div key={rate.id} className="flex items-start space-x-2 p-2 sm:p-3 border rounded-md hover:bg-muted/50">
														<RadioGroupItem value={rate.id} id={rate.id} className="mt-0.5" />
														<Label htmlFor={rate.id} className="flex-1 cursor-pointer">
															<div className="flex justify-between items-start gap-2">
																<div className="flex-1 min-w-0">
																	<p className="font-medium text-xs sm:text-sm break-words">
																		{rate.display_name}
																	</p>
																	{rate.metadata?.delivery_days && (
																		<p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
																			<Clock className="h-3 w-3 flex-shrink-0" />
																			<span>{rate.metadata.delivery_days} days</span>
																		</p>
																	)}
																</div>
																<p className="font-bold text-xs sm:text-sm flex-shrink-0">
																	{rate.amount === 0 ? 'FREE' : `$${(rate.amount / 100).toFixed(2)}`}
																</p>
															</div>
														</Label>
													</div>
												))}
											</RadioGroup>
										)}
										{subtotal < 75 && (
											<p className="text-xs text-muted-foreground">
												Add ${(75 - subtotal).toFixed(2)} more for free shipping!
											</p>
										)}
									</div>
								)}

								{/* Promo Code */}
								<div className="flex gap-2">
									<Input
										placeholder="Promo code"
										value={promoCode}
										onChange={(e) => setPromoCode(e.target.value)}
										className="h-9 text-sm"
									/>
									<Button variant="outline" size="sm" disabled>
										Apply
									</Button>
								</div>

								<Separator />

								{/* Price Breakdown */}
								<div className="space-y-2">
									<div className="flex justify-between text-sm">
										<span>Subtotal</span>
										<span className="font-medium">${subtotal.toFixed(2)}</span>
									</div>
									<div className="flex justify-between text-sm">
										<span>Shipping</span>
										{isLoadingShippingRates ? (
											<Skeleton className="h-4 w-16" />
										) : (
											<span className="font-medium">{shipping === 0 ? 'FREE' : `$${shipping.toFixed(2)}`}</span>
										)}
									</div>
									<div className="flex justify-between font-bold text-base sm:text-lg">
										<span>Estimated Total</span>
										{isCalculatingTax || isLoadingShippingRates ? (
											<Skeleton className="h-5 sm:h-6 w-20" />
										) : (
											<span>${selectedShippingRate ? total.toFixed(2) : subtotal.toFixed(2)}</span>
										)}
									</div>
									<p className="text-xs text-muted-foreground">
										{selectedShippingRate
											? `Estimated with ${shippingRates.find(r => r.id === selectedShippingRate)?.display_name}. Final shipping selected at checkout.`
											: 'Shipping and tax calculated at checkout'
										}
									</p>
									{tax === 0 && shippingAddress.state && shippingAddress.postal_code && (
										<p className="text-xs text-muted-foreground italic">
											No tax in {shippingAddress.state}
										</p>
									)}
									{shipping === 0 && (
										<Badge variant="secondary" className="w-full justify-center text-xs">
											✨ You qualify for free shipping!
										</Badge>
									)}
									<Separator />
									<div className="flex justify-between font-bold text-base sm:text-lg">
										<span>Total</span>
										{isCalculatingTax || isLoadingShippingRates ? (
											<Skeleton className="h-5 sm:h-6 w-20" />
										) : (
											<span>${total.toFixed(2)}</span>
										)}
									</div>
									{!shippingAddress.state || !shippingAddress.postal_code ? (
										<p className="text-xs text-muted-foreground">
											Final tax calculated at checkout
										</p>
									) : (
										<p className="text-xs text-muted-foreground">
											Tax for {shippingAddress.state}. Final amount may vary.
										</p>
									)}
								</div>
							</CardContent>
							<CardFooter className="flex flex-col gap-2 pt-3 sm:pt-6">
								<Button
									className="w-full"
									size="default"
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
