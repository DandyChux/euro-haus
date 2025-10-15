import { UseFormReturn } from 'react-hook-form';
import { useState, useEffect } from 'react';
import { BundleFormData, BundleItem } from '~/lib/schemas/product-schema';
import {
	FormControl,
	FormDescription,
	FormField,
	FormItem,
	FormLabel,
	FormMessage,
} from '~/components/ui/form';
import { Input } from '~/components/ui/input';
import { Button } from '~/components/ui/button';
import { Checkbox } from '~/components/ui/checkbox';
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from '~/components/ui/select';
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Plus, Trash2, Package, Calculator } from 'lucide-react';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';

interface StripeProduct {
	id: string;
	name: string;
	description: string | null;
	images: string[];
	metadata: Record<string, any>;
	default_price: {
		id: string;
		unit_amount: number;
		currency: string;
	} | null;
}

interface BundleFormSectionProps {
	form: UseFormReturn<BundleFormData>;
}

export function BundleFormSection({ form }: BundleFormSectionProps) {
	const [availableProducts, setAvailableProducts] = useState<StripeProduct[]>([]);
	const [isLoadingProducts, setIsLoadingProducts] = useState(false);
	const [selectedProductId, setSelectedProductId] = useState<string>('');
	const [quantity, setQuantity] = useState(1);

	const bundleItems = form.watch('bundleItems') || [];
	const discountType = form.watch('discountType');
	const discountValue = form.watch('discountValue');

	// Fetch available products
	useEffect(() => {
		const fetchProducts = async () => {
			setIsLoadingProducts(true);
			try {
				const response = await apiClient.get('/products');
				// Filter out events and bundles, only show regular products
				const products = (response.data.products || []).filter(
					(p: StripeProduct) =>
						p.metadata.type === 'product' &&
						p.default_price?.unit_amount
				);
				setAvailableProducts(products);
			} catch (error) {
				console.error('Error fetching products:', error);
				toast.error('Failed to fetch products');
			} finally {
				setIsLoadingProducts(false);
			}
		};

		fetchProducts();
	}, []);

	// Calculate total price of bundled items
	const calculateTotalPrice = () => {
		return bundleItems.reduce((total, item) => {
			return total + (item.price * item.quantity);
		}, 0);
	};

	// Calculate bundle price with discount
	const calculateBundlePrice = () => {
		const totalPrice = calculateTotalPrice();
		if (!discountValue || parseFloat(discountValue) === 0) return totalPrice;

		const discount = parseFloat(discountValue);

		if (discountType === 'percentage') {
			return totalPrice * (1 - discount / 100);
		} else {
			return Math.max(0, totalPrice - discount);
		}
	};

	// Calculate savings
	const calculateSavings = () => {
		const totalPrice = calculateTotalPrice();
		const bundlePrice = calculateBundlePrice();
		return totalPrice - bundlePrice;
	};

	// Auto-update price when discount changes
	useEffect(() => {
		if (bundleItems.length > 0) {
			const calculatedPrice = calculateBundlePrice();
			form.setValue('price', calculatedPrice.toFixed(2));
		}
	}, [bundleItems, discountType, discountValue]);

	// Add product to bundle
	const addProductToBundle = () => {
		if (!selectedProductId) {
			toast.error('Please select a product');
			return;
		}

		const product = availableProducts.find(p => p.id === selectedProductId);
		if (!product || !product.default_price) {
			toast.error('Invalid product selected');
			return;
		}

		// Check if product already in bundle
		const existingItem = bundleItems.find(item => item.productId === selectedProductId);
		if (existingItem) {
			toast.error('This product is already in the bundle');
			return;
		}

		const newItem: BundleItem = {
			productId: product.id,
			productName: product.name,
			quantity: quantity,
			price: product.default_price.unit_amount / 100, // Convert from cents
		};

		form.setValue('bundleItems', [...bundleItems, newItem]);
		setSelectedProductId('');
		setQuantity(1);
		toast.success('Product added to bundle');
	};

	// Remove product from bundle
	const removeProductFromBundle = (productId: string) => {
		const updatedItems = bundleItems.filter(item => item.productId !== productId);
		form.setValue('bundleItems', updatedItems);
		toast.success('Product removed from bundle');
	};

	// Update quantity for a bundle item
	const updateQuantity = (productId: string, newQuantity: number) => {
		if (newQuantity < 1) return;

		const updatedItems = bundleItems.map(item =>
			item.productId === productId
				? { ...item, quantity: newQuantity }
				: item
		);
		form.setValue('bundleItems', updatedItems);
	};

	return (
		<div className="space-y-6">
			{/* Bundle Items Section */}
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<Package className="h-5 w-5" />
						Bundle Contents
					</CardTitle>
					<CardDescription>
						Add products to this bundle. Select at least 2 products.
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					{/* Add Product Controls */}
					<div className="flex gap-2">
						<Select
							value={selectedProductId}
							onValueChange={setSelectedProductId}
							disabled={isLoadingProducts}
						>
							<SelectTrigger className="flex-1">
								<SelectValue placeholder="Select a product to add..." />
							</SelectTrigger>
							<SelectContent>
								{availableProducts.map((product) => (
									<SelectItem key={product.id} value={product.id}>
										{product.name} - ${(product.default_price?.unit_amount || 0) / 100}
									</SelectItem>
								))}
							</SelectContent>
						</Select>

						<Input
							type="number"
							min="1"
							value={quantity}
							onChange={(e) => setQuantity(parseInt(e.target.value) || 1)}
							className="w-20"
							placeholder="Qty"
						/>

						<Button
							type="button"
							onClick={addProductToBundle}
							disabled={!selectedProductId}
						>
							<Plus className="h-4 w-4 mr-2" />
							Add
						</Button>
					</div>

					{/* Bundle Items List */}
					{bundleItems.length > 0 ? (
						<div className="space-y-2">
							{bundleItems.map((item) => (
								<div
									key={item.productId}
									className="flex items-center justify-between p-3 border rounded-lg"
								>
									<div className="flex-1">
										<p className="font-medium">{item.productName}</p>
										<p className="text-sm text-muted-foreground">
											${item.price.toFixed(2)} each
										</p>
									</div>

									<div className="flex items-center gap-4">
										<div className="flex items-center gap-2">
											<Button
												type="button"
												variant="outline"
												size="sm"
												onClick={() => updateQuantity(item.productId, item.quantity - 1)}
												disabled={item.quantity <= 1}
											>
												-
											</Button>
											<span className="w-8 text-center">{item.quantity}</span>
											<Button
												type="button"
												variant="outline"
												size="sm"
												onClick={() => updateQuantity(item.productId, item.quantity + 1)}
											>
												+
											</Button>
										</div>

										<div className="text-right min-w-[80px]">
											<p className="font-semibold">
												${(item.price * item.quantity).toFixed(2)}
											</p>
										</div>

										<Button
											type="button"
											variant="ghost"
											size="sm"
											onClick={() => removeProductFromBundle(item.productId)}
										>
											<Trash2 className="h-4 w-4 text-destructive" />
										</Button>
									</div>
								</div>
							))}
						</div>
					) : (
						<div className="text-center py-8 text-muted-foreground">
							<Package className="h-12 w-12 mx-auto mb-2 opacity-50" />
							<p>No products added yet. Add at least 2 products to create a bundle.</p>
						</div>
					)}
				</CardContent>
			</Card>

			{/* Pricing Section */}
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<Calculator className="h-5 w-5" />
						Bundle Pricing
					</CardTitle>
					<CardDescription>
						Set the discount for this bundle
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="grid grid-cols-2 gap-4">
						<FormField
							control={form.control}
							name="discountType"
							render={({ field }) => (
								<FormItem>
									<FormLabel>Discount Type</FormLabel>
									<Select onValueChange={field.onChange} value={field.value}>
										<FormControl>
											<SelectTrigger>
												<SelectValue placeholder="Select discount type" />
											</SelectTrigger>
										</FormControl>
										<SelectContent>
											<SelectItem value="percentage">Percentage (%)</SelectItem>
											<SelectItem value="fixed">Fixed Amount ($)</SelectItem>
										</SelectContent>
									</Select>
									<FormMessage />
								</FormItem>
							)}
						/>

						<FormField
							control={form.control}
							name="discountValue"
							render={({ field }) => (
								<FormItem>
									<FormLabel>
										Discount Value {discountType === 'percentage' ? '(%)' : '($)'}
									</FormLabel>
									<FormControl>
										<Input
											type="number"
											step="0.01"
											min="0"
											placeholder={discountType === 'percentage' ? '10' : '5.00'}
											{...field}
										/>
									</FormControl>
									<FormMessage />
								</FormItem>
							)}
						/>
					</div>

					{bundleItems.length > 0 && (
						<div className="mt-4 p-4 bg-muted rounded-lg space-y-2">
							<div className="flex justify-between text-sm">
								<span>Total of individual items:</span>
								<span className="font-medium">${calculateTotalPrice().toFixed(2)}</span>
							</div>
							{discountValue && parseFloat(discountValue) > 0 && (
								<>
									<div className="flex justify-between text-sm text-green-600">
										<span>Savings:</span>
										<span className="font-medium">-${calculateSavings().toFixed(2)}</span>
									</div>
									<div className="flex justify-between text-lg font-bold pt-2 border-t">
										<span>Bundle Price:</span>
										<span>${calculateBundlePrice().toFixed(2)}</span>
									</div>
								</>
							)}
						</div>
					)}

					<FormField
						control={form.control}
						name="price"
						render={({ field }) => (
							<FormItem>
								<FormLabel>Final Bundle Price ($)</FormLabel>
								<FormControl>
									<Input
										type="number"
										step="0.01"
										min="0"
										placeholder="0.00"
										{...field}
										readOnly
										className="bg-muted"
									/>
								</FormControl>
								<FormDescription>
									This is calculated automatically based on your discount
								</FormDescription>
								<FormMessage />
							</FormItem>
						)}
					/>
				</CardContent>
			</Card>

			{/* Additional Settings */}
			<div className="grid grid-cols-2 gap-4">
				<FormField
					control={form.control}
					name="inStock"
					render={({ field }) => (
						<FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
							<FormControl>
								<Checkbox
									checked={field.value}
									onCheckedChange={field.onChange}
								/>
							</FormControl>
							<div className="space-y-1 leading-none">
								<FormLabel>In Stock</FormLabel>
								<FormDescription>
									Is this bundle currently available for purchase?
								</FormDescription>
							</div>
						</FormItem>
					)}
				/>

				<FormField
					control={form.control}
					name="maxQuantity"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Max Quantity Per Order</FormLabel>
							<FormControl>
								<Input
									type="number"
									min="0"
									placeholder="10"
									{...field}
								/>
							</FormControl>
							<FormDescription>
								Leave empty for unlimited
							</FormDescription>
							<FormMessage />
						</FormItem>
					)}
				/>
			</div>
		</div>
	);
}
