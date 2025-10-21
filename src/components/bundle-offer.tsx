import React, { useState, useMemo, useEffect } from 'react';
import { Button } from '~/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { Badge } from '~/components/ui/badge';
import { Separator } from '~/components/ui/separator';
import { Alert, AlertDescription } from '~/components/ui/alert';
import {
	BundleProduct,
	BundleItem,
	BundleItemWithProduct,
	Product,
	ProductWithVariants,
	ProductVariant,
} from '~/lib/services/stripe-service';
import { ShoppingCart, Loader2, Package, AlertCircle, Check, X } from 'lucide-react';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';
import { cn } from '~/lib/utils';

// Type guard for products with variants
function productHasVariants(product: Product | ProductWithVariants): product is ProductWithVariants {
	return 'variants' in product && Array.isArray(product.variants);
}

// Type guard to check if bundle items have product data
function isBundleItemWithProduct(item: BundleItem | BundleItemWithProduct): item is BundleItemWithProduct {
	return 'product' in item;
}

export interface SelectedVariant {
	productId: string;
	variant: ProductVariant | null;
	priceId: string | null;
}

interface BundleOfferInteractiveProps {
	bundle: BundleProduct;
	onAddToCart?: (bundle: BundleProduct, selections: SelectedVariant[]) => void;
}

export function BundleOfferInteractive({ bundle, onAddToCart }: BundleOfferInteractiveProps) {
	const [selectedVariants, setSelectedVariants] = useState<Record<string, SelectedVariant>>({});
	const [isProcessing, setIsProcessing] = useState(false);

	// Check if all bundle items have product data
	const allItemsHaveProductData = bundle.bundleItems.every(isBundleItemWithProduct);
	const bundleItems = allItemsHaveProductData ? (bundle.bundleItems as BundleItemWithProduct[]) : [];

	// Initialize selections - MUST be called unconditionally
	useEffect(() => {
		if (!allItemsHaveProductData) return;

		const initialSelections: Record<string, SelectedVariant> = {};

		bundleItems.forEach(item => {
			const product = item.product;
			const hasVariants = productHasVariants(product);

			if (hasVariants && product.variants.length > 0) {
				// Product has variants - user must select
				initialSelections[item.productId] = {
					productId: item.productId,
					variant: null,
					priceId: null,
				};
			} else {
				// No variants - use default price
				initialSelections[item.productId] = {
					productId: item.productId,
					variant: null,
					priceId: product.priceId || null,
				};
			}
		});

		setSelectedVariants(initialSelections);
	}, [allItemsHaveProductData, bundleItems]);

	// Handle variant selection
	const handleVariantSelect = (productId: string, variantId: string) => {
		const item = bundleItems.find(i => i.productId === productId);
		if (!item) return;

		const product = item.product;
		if (!productHasVariants(product)) return;

		const variant = product.variants.find(v => v.id === variantId);
		if (!variant) return;

		setSelectedVariants(prev => ({
			...prev,
			[productId]: {
				productId,
				variant,
				priceId: variant.priceId,
			}
		}));
	};

	// Check if all required selections are made - MUST be called unconditionally
	const allSelectionsValid = useMemo(() => {
		if (!allItemsHaveProductData) return false;

		return bundleItems.every(item => {
			const selection = selectedVariants[item.productId];
			if (!selection) return false;

			const product = item.product;
			// If product has variants, a variant must be selected
			if (productHasVariants(product) && product.variants.length > 0) {
				return selection.variant !== null && selection.priceId !== null;
			}

			// Otherwise, just need a priceId
			return selection.priceId !== null;
		});
	}, [selectedVariants, bundleItems, allItemsHaveProductData]);

	// Check if all selections are in stock - MUST be called unconditionally
	const allInStock = useMemo(() => {
		if (!allItemsHaveProductData) return false;

		return bundleItems.every(item => {
			const selection = selectedVariants[item.productId];
			if (!selection) return false;

			if (selection.variant) {
				return selection.variant.inStock;
			}

			return item.product.inStock !== false;
		});
	}, [selectedVariants, bundleItems, allItemsHaveProductData]);

	// Handle checkout
	const handleBundleCheckout = async () => {
		if (!allSelectionsValid) {
			toast.error("Please select options for all items in the bundle");
			return;
		}

		if (!allInStock) {
			toast.error("Some selected items are out of stock");
			return;
		}

		setIsProcessing(true);

		try {
			// Build line items with selected variants
			const lineItems = bundleItems.map(item => {
				const selection = selectedVariants[item.productId];
				return {
					price: selection!.priceId,
					quantity: item.quantity,
				};
			});

			// Create checkout session with bundle discount
			const response = await apiClient.post('/create-checkout-session', {
				line_items: lineItems,
				success_url: `${window.location.origin}/checkout/success`,
				cancel_url: window.location.href,
				bundleDiscount: {
					name: bundle.title,
					type: bundle.discountType,
					value: bundle.discountValue.toString(),
				},
				metadata: {
					bundle_id: bundle.id,
					bundle_name: bundle.title,
				}
			});

			// Redirect to Stripe Checkout
			const stripe = window.Stripe?.(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);
			if (stripe && response.data.sessionId) {
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			} else {
				throw new Error("Unable to redirect to checkout");
			}
		} catch (error) {
			console.error("Bundle checkout failed:", error);
			toast.error("Failed to start checkout. Please try again.");
		} finally {
			setIsProcessing(false);
		}
	};

	// Handle add to cart (if provided)
	const handleAddToCart = () => {
		if (!allSelectionsValid) {
			toast.error("Please select options for all items in the bundle");
			return;
		}

		if (!allInStock) {
			toast.error("Some selected options are out of stock");
			return;
		}

		if (onAddToCart) {
			const selections = Object.values(selectedVariants);
			onAddToCart(bundle, selections);
		}
	};

	// Format savings percentage
	const savingsPercentage = bundle.totalValue > 0
		? Math.round((bundle.savings / bundle.totalValue) * 100)
		: 0;

	// Show simplified view if no product data
	if (!allItemsHaveProductData) {
		return (
			<Card className="overflow-hidden border-2 border-primary/20">
				<CardHeader className="bg-gradient-to-r from-primary/10 to-primary/5">
					<div className="flex items-start justify-between">
						<div>
							<CardTitle className="text-xl flex items-center gap-2">
								<Package className="h-5 w-5" />
								{bundle.title}
							</CardTitle>
							<p className="text-sm text-muted-foreground mt-1">
								Save ${bundle.savings.toFixed(2)} when you buy these together
							</p>
						</div>
						<Badge className="bg-green-600 text-white">
							BUNDLE DEAL
						</Badge>
					</div>
				</CardHeader>
				<CardContent className="pt-6 space-y-4">
					<div className="space-y-2">
						{(bundle.bundleItems as BundleItem[]).map((item) => (
							<div key={item.productId} className="flex justify-between">
								<span>{item.productName} {item.quantity > 1 && `(x${item.quantity})`}</span>
								<span className="text-muted-foreground">${(item.price * item.quantity).toFixed(2)}</span>
							</div>
						))}
					</div>
					<Separator />
					<div className="flex justify-between text-lg font-bold">
						<span>Bundle Price:</span>
						<span className="text-primary">${bundle.price.toFixed(2)}</span>
					</div>
					<Alert>
						<AlertCircle className="h-4 w-4" />
						<AlertDescription>
							To select sizes and options, please visit each product page individually.
						</AlertDescription>
					</Alert>
				</CardContent>
			</Card>
		);
	}

	return (
		<Card className="overflow-hidden border-2 border-primary/20">
			<CardHeader className="bg-gradient-to-r from-primary/10 to-primary/5">
				<div className="flex items-start justify-between">
					<div>
						<CardTitle className="text-xl flex items-center gap-2">
							<Package className="h-5 w-5" />
							{bundle.title}
						</CardTitle>
						<p className="text-sm text-muted-foreground mt-1">
							Save ${bundle.savings.toFixed(2)} ({savingsPercentage}% off) when you buy these together
						</p>
					</div>
					<Badge className="bg-green-600 text-white">
						BUNDLE DEAL
					</Badge>
				</div>
			</CardHeader>

			<CardContent className="pt-6 space-y-6">
				{/* Bundle Items */}
				<div className="space-y-4">
					{bundleItems.map((item, index) => {
						const product = item.product;
						const hasVariants = productHasVariants(product);
						const selection = selectedVariants[item.productId];
						const isSelected = selection && (hasVariants ? selection.variant !== null : true);

						return (
							<div key={item.productId}>
								<div className={cn(
									"p-4 rounded-lg border transition-all",
									isSelected ? "border-primary bg-primary/5" : "border-muted"
								)}>
									<div className="flex flex-col sm:flex-row gap-4">
										{/* Product Info */}
										<div className="flex-1">
											<div className="flex items-start justify-between">
												<div>
													<h4 className="font-semibold flex items-center gap-2">
														{item.productName}
														{item.quantity > 1 && (
															<Badge variant="secondary" className="text-xs">
																×{item.quantity}
															</Badge>
														)}
													</h4>
													<p className="text-sm text-muted-foreground mt-1">
														Regular price: ${item.price.toFixed(2)} each
													</p>
												</div>
												{isSelected && (
													<Check className="h-5 w-5 text-primary" />
												)}
											</div>

											{/* Variant Selection */}
											{hasVariants && product.variants.length > 0 && (
												<div className="mt-3">
													<label className="text-sm font-medium mb-2 block">
														Select {product.category === 'apparel' ? 'Size' : 'Option'}:
													</label>

													{/* Size/Variant Grid for Apparel */}
													{product.category === 'apparel' ? (
														<div className="grid grid-cols-4 sm:grid-cols-6 gap-2">
															{product.variants.map((variant) => (
																<Button
																	key={variant.id}
																	type="button"
																	variant={selection?.variant?.id === variant.id ? "default" : "outline"}
																	size="sm"
																	className={cn(
																		"text-xs",
																		!variant.inStock && "opacity-50 cursor-not-allowed"
																	)}
																	onClick={() => variant.inStock && handleVariantSelect(item.productId, variant.id)}
																	disabled={!variant.inStock}
																>
																	{variant.size || variant.variant}
																	{!variant.inStock && (
																		<X className="h-3 w-3 ml-1" />
																	)}
																</Button>
															))}
														</div>
													) : (
														// Dropdown for other products
														<Select
															value={selection?.variant?.id || ''}
															onValueChange={(value) => handleVariantSelect(item.productId, value)}
														>
															<SelectTrigger className="w-full sm:w-[200px]">
																<SelectValue placeholder="Choose an option..." />
															</SelectTrigger>
															<SelectContent>
																{product.variants.map((variant) => (
																	<SelectItem
																		key={variant.id}
																		value={variant.id}
																		disabled={!variant.inStock}
																	>
																		<div className="flex items-center justify-between w-full">
																			<span>{variant.variant}</span>
																			{!variant.inStock && (
																				<span className="text-xs text-muted-foreground ml-2">Out of stock</span>
																			)}
																		</div>
																	</SelectItem>
																))}
															</SelectContent>
														</Select>
													)}

													{/* Show selected variant info */}
													{selection?.variant && (
														<p className="text-xs text-muted-foreground mt-2">
															Selected: {selection.variant.variant}
															{selection.variant.color && ` - ${selection.variant.color}`}
														</p>
													)}
												</div>
											)}

											{/* No variants - just show as included */}
											{!hasVariants && (
												<div className="mt-2">
													<Badge variant="outline" className="text-xs">
														<Check className="h-3 w-3 mr-1" />
														Included
													</Badge>
												</div>
											)}
										</div>

										{/* Item Total */}
										<div className="text-right">
											<p className="text-sm text-muted-foreground">Item value:</p>
											<p className="font-semibold">
												${(item.price * item.quantity).toFixed(2)}
											</p>
										</div>
									</div>
								</div>

								{index < bundleItems.length - 1 && (
									<Separator className="my-4" />
								)}
							</div>
						);
					})}
				</div>

				{/* Validation Messages */}
				{!allSelectionsValid && (
					<Alert>
						<AlertCircle className="h-4 w-4" />
						<AlertDescription>
							Please select options for all items before proceeding
						</AlertDescription>
					</Alert>
				)}

				{allSelectionsValid && !allInStock && (
					<Alert variant="destructive">
						<AlertCircle className="h-4 w-4" />
						<AlertDescription>
							Some selected options are out of stock. Please choose different options.
						</AlertDescription>
					</Alert>
				)}

				{/* Pricing Summary */}
				<div className="bg-muted/50 rounded-lg p-4 space-y-2">
					<div className="flex justify-between text-sm">
						<span>Total value (if bought separately):</span>
						<span className="line-through text-muted-foreground">
							${bundle.totalValue.toFixed(2)}
						</span>
					</div>
					<div className="flex justify-between text-sm text-green-600 font-medium">
						<span>Bundle discount:</span>
						<span>-${bundle.savings.toFixed(2)}</span>
					</div>
					<Separator />
					<div className="flex justify-between text-lg font-bold">
						<span>Bundle price:</span>
						<span className="text-primary">${bundle.price.toFixed(2)}</span>
					</div>
				</div>

				{/* Actions */}
				<div className="flex flex-col sm:flex-row gap-3">
					<Button
						size="lg"
						className="flex-1"
						onClick={handleBundleCheckout}
						disabled={!allSelectionsValid || !allInStock || isProcessing}
					>
						{isProcessing ? (
							<>
								<Loader2 className="mr-2 h-5 w-5 animate-spin" />
								Processing...
							</>
						) : (
							<>
								<ShoppingCart className="mr-2 h-5 w-5" />
								Buy Bundle Now
							</>
						)}
					</Button>

					{onAddToCart && (
						<Button
							size="lg"
							variant="outline"
							onClick={handleAddToCart}
							disabled={!allSelectionsValid || !allInStock || isProcessing}
						>
							Add Bundle to Cart
						</Button>
					)}
				</div>

				{/* Additional Info */}
				<div className="text-xs text-muted-foreground space-y-1">
					<p>• Bundle discount applied at checkout</p>
					<p>• All items ship together</p>
					<p>• Cannot be combined with other offers</p>
				</div>
			</CardContent>
		</Card>
	);
}
