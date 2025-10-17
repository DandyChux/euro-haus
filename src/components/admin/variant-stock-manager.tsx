import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Input } from '~/components/ui/input';
import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';
import { Package, Save, AlertCircle } from 'lucide-react';

interface VariantStockManagerProps {
	productId: string;
}

interface VariantPrice {
	id: string;
	nickname: string | null;
	unit_amount: number;
	metadata: {
		variant?: string;
		size?: string;
		color?: string;
		in_stock?: string;
		stock_quantity?: string;
		[key: string]: string | undefined;
	};
}

export function VariantStockManager({ productId }: VariantStockManagerProps) {
	const [prices, setPrices] = useState<VariantPrice[]>([]);
	const [stockQuantities, setStockQuantities] = useState<Record<string, string>>({});
	const [isLoading, setIsLoading] = useState(true);
	const [isSaving, setIsSaving] = useState(false);
	const [hasChanges, setHasChanges] = useState(false);

	// Fetch existing prices/variants
	useEffect(() => {
		const fetchPrices = async () => {
			try {
				const response = await apiClient.get(`/admin/product-prices/${productId}`);
				const priceData = response.data.prices || [];
				setPrices(priceData);

				// Initialize stock quantities from metadata
				const initialStock: Record<string, string> = {};
				priceData.forEach((price: VariantPrice) => {
					initialStock[price.id] = price.metadata.stock_quantity || '';
				});
				setStockQuantities(initialStock);
			} catch (error) {
				console.error('Failed to fetch variant prices:', error);
				toast.error('Failed to load variant stock levels');
			} finally {
				setIsLoading(false);
			}
		};

		fetchPrices();
	}, [productId]);

	// Handle stock quantity change
	const handleStockChange = (priceId: string, value: string) => {
		// Only allow numbers
		if (value && !/^\d+$/.test(value)) return;

		setStockQuantities(prev => ({
			...prev,
			[priceId]: value
		}));
		setHasChanges(true);
	};

	// Save all changes
	const saveChanges = async () => {
		setIsSaving(true);
		try {
			// Update each price's metadata with new stock quantity
			const updatePromises = prices.map(async (price) => {
				const newQuantity = stockQuantities[price.id];
				const currentQuantity = price.metadata.stock_quantity || '';

				// Only update if changed
				if (newQuantity !== currentQuantity) {
					await apiClient.put(`/admin/update-price-metadata/${price.id}`, {
						metadata: {
							...price.metadata,
							stock_quantity: newQuantity,
							in_stock: newQuantity === '0' ? 'false' : 'true'
						}
					});
				}
			});

			await Promise.all(updatePromises);
			toast.success('Stock levels updated successfully');
			setHasChanges(false);

			// Refresh prices to get updated metadata
			const response = await apiClient.get(`/admin/product-prices/${productId}`);
			setPrices(response.data.prices || []);
		} catch (error) {
			console.error('Failed to update stock levels:', error);
			toast.error('Failed to update stock levels');
		} finally {
			setIsSaving(false);
		}
	};

	if (isLoading) {
		return <div className="animate-pulse bg-muted h-32 rounded-lg" />;
	}

	if (prices.length === 0) {
		return null;
	}

	// Calculate total stock
	const totalStock = Object.values(stockQuantities).reduce((sum, qty) => {
		const num = parseInt(qty) || 0;
		return sum + num;
	}, 0);

	return (
		<Card>
			<CardHeader>
				<CardTitle className="flex items-center gap-2">
					<Package className="h-5 w-5" />
					Variant Stock Levels
				</CardTitle>
			</CardHeader>
			<CardContent>
				<div className="space-y-4">
					<div className="flex items-center justify-between p-3 bg-muted rounded-lg">
						<span className="font-medium">Total Stock Across All Variants:</span>
						<Badge variant={totalStock > 0 ? "default" : "destructive"}>
							{totalStock} units
						</Badge>
					</div>

					<div className="space-y-2">
						{prices.map((price) => {
							const quantity = stockQuantities[price.id] || '';
							const numQuantity = parseInt(quantity) || 0;

							return (
								<div key={price.id} className="flex items-center gap-4 p-3 border rounded-lg">
									<div className="flex-1">
										<p className="font-medium">
											{price.metadata.variant || price.nickname || 'Default'}
										</p>
										{(price.metadata.size || price.metadata.color) && (
											<p className="text-sm text-muted-foreground">
												{price.metadata.size && `Size: ${price.metadata.size}`}
												{price.metadata.size && price.metadata.color && ' • '}
												{price.metadata.color && `Color: ${price.metadata.color}`}
											</p>
										)}
									</div>

									<div className="flex items-center gap-2">
										<Input
											type="number"
											min="0"
											value={quantity}
											onChange={(e) => handleStockChange(price.id, e.target.value)}
											className="w-24"
											placeholder="∞"
										/>
										<Badge
											variant={numQuantity === 0 ? "destructive" : numQuantity < 5 ? "secondary" : "default"}
										>
											{numQuantity === 0 ? 'Out of Stock' : numQuantity < 5 ? 'Low Stock' : 'In Stock'}
										</Badge>
									</div>
								</div>
							);
						})}
					</div>

					{hasChanges && (
						<Alert>
							<AlertCircle className="h-4 w-4" />
							<AlertDescription>
								You have unsaved changes to stock levels
							</AlertDescription>
						</Alert>
					)}

					<Button
						onClick={saveChanges}
						disabled={!hasChanges || isSaving}
						className="w-full"
					>
						<Save className="h-4 w-4 mr-2" />
						{isSaving ? 'Saving...' : 'Save Stock Levels'}
					</Button>
				</div>
			</CardContent>
		</Card>
	);
}
