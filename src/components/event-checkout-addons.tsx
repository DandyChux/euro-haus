import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Checkbox } from '~/components/ui/checkbox';
import { Badge } from '~/components/ui/badge';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Plus, Minus, Package } from 'lucide-react';

interface LinkedProduct {
	id: string;
	name: string;
	description?: string;
	images?: string[];
	price: {
		id: string;
		unit_amount: number;
		currency: string;
	};
}

interface EventCheckoutAddonsProps {
	linkedProducts: LinkedProduct[];
	includedProducts?: any[];
	onAddonsChange: (addons: SelectedAddon[]) => void;
}

interface SelectedAddon {
	productId: string;
	priceId: string;
	quantity: number;
	name: string;
	unitPrice: number;
}

export function EventCheckoutAddons({
	linkedProducts,
	includedProducts = [],
	onAddonsChange
}: EventCheckoutAddonsProps) {
	const [selectedAddons, setSelectedAddons] = useState<Record<string, SelectedAddon>>({});

	const handleAddonToggle = (product: LinkedProduct, selected: boolean) => {
		const newAddons = { ...selectedAddons };

		if (selected) {
			newAddons[product.id] = {
				productId: product.id,
				priceId: product.price.id,
				quantity: 1,
				name: product.name,
				unitPrice: product.price.unit_amount
			};
		} else {
			delete newAddons[product.id];
		}

		setSelectedAddons(newAddons);
		onAddonsChange(Object.values(newAddons));
	};

	const handleQuantityChange = (productId: string, delta: number) => {
		const newAddons = { ...selectedAddons };
		if (newAddons[productId]) {
			const newQuantity = Math.max(1, newAddons[productId].quantity + delta);
			newAddons[productId].quantity = newQuantity;
			setSelectedAddons(newAddons);
			onAddonsChange(Object.values(newAddons));
		}
	};

	const calculateTotal = () => {
		return Object.values(selectedAddons).reduce((total, addon) => {
			return total + (addon.unitPrice * addon.quantity);
		}, 0);
	};

	if (linkedProducts.length === 0 && includedProducts.length === 0) {
		return null;
	}

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2">
					<Package className="w-5 h-5" />
					Event Add-ons & Merchandise
				</CardTitle>
			</CardHeader>
			<CardContent className="space-y-4">
				{/* Included Products */}
				{includedProducts.length > 0 && (
					<div className="space-y-2">
						<h3 className="text-sm font-semibold text-muted-foreground">
							Included with your ticket:
						</h3>
						<div className="space-y-2">
							{includedProducts.map(product => (
								<div key={product.id} className="flex items-center justify-between p-3 bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg">
									<div className="flex items-center gap-3">
										<div className="w-2 h-2 bg-green-500 rounded-full" />
										<span className="font-medium">{product.name}</span>
										{product.quantity > 1 && (
											<Badge variant="secondary">
												×{product.quantity}
											</Badge>
										)}
									</div>
									<Badge variant="outline" className="bg-green-100 dark:bg-green-900/50">
										Included
									</Badge>
								</div>
							))}
						</div>
					</div>
				)}

				{/* Optional Add-ons */}
				{linkedProducts.length > 0 && (
					<div className="space-y-2">
						<h3 className="text-sm font-semibold text-muted-foreground">
							Available add-ons:
						</h3>
						<div className="space-y-2">
							{linkedProducts.map(product => {
								const isSelected = !!selectedAddons[product.id];
								const addon = selectedAddons[product.id];

								return (
									<div key={product.id} className="border rounded-lg p-3">
										<div className="flex items-start justify-between">
											<div className="flex items-start gap-3">
												<Checkbox
													id={`addon-${product.id}`}
													checked={isSelected}
													onCheckedChange={(checked) =>
														handleAddonToggle(product, checked as boolean)
													}
												/>
												<label
													htmlFor={`addon-${product.id}`}
													className="flex-1 cursor-pointer"
												>
													<div className="flex items-start gap-3">
														{product.images?.[0] && (
															<img
																src={product.images[0]}
																alt={product.name}
																className="w-16 h-16 object-cover rounded"
															/>
														)}
														<div className="flex-1">
															<div className="font-medium">{product.name}</div>
															{product.description && (
																<p className="text-sm text-muted-foreground mt-1">
																	{product.description}
																</p>
															)}
															<div className="text-sm font-semibold mt-2">
																${(product.price.unit_amount / 100).toFixed(2)}
															</div>
														</div>
													</div>
												</label>
											</div>

											{isSelected && (
												<div className="flex items-center gap-2">
													<Button
														variant="outline"
														size="sm"
														onClick={() => handleQuantityChange(product.id, -1)}
														disabled={addon.quantity <= 1}
													>
														<Minus className="w-3 h-3" />
													</Button>
													<span className="min-w-[30px] text-center font-medium">
														{addon.quantity}
													</span>
													<Button
														variant="outline"
														size="sm"
														onClick={() => handleQuantityChange(product.id, 1)}
													>
														<Plus className="w-3 h-3" />
													</Button>
												</div>
											)}
										</div>
									</div>
								);
							})}
						</div>
					</div>
				)}

				{/* Total for add-ons */}
				{Object.keys(selectedAddons).length > 0 && (
					<div className="pt-4 border-t">
						<div className="flex justify-between items-center">
							<span className="font-medium">Add-ons Total:</span>
							<span className="text-lg font-bold">
								${(calculateTotal() / 100).toFixed(2)}
							</span>
						</div>
					</div>
				)}
			</CardContent>
		</Card>
	);
}
