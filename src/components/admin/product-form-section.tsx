import { useEffect } from 'react';
import { Card, CardContent } from '~/components/ui/card';
import { FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from '~/components/ui/form';
import { Input } from '~/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { Checkbox } from '~/components/ui/checkbox';
import { Separator } from '~/components/ui/separator';
import { Button } from '~/components/ui/button';
import { UseFormReturn, useFieldArray } from 'react-hook-form';
import { Plus, Trash2, GripVertical } from 'lucide-react';
import { FormData } from '~/lib/schemas/product-schema';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';

interface ProductFormSectionProps {
	form: UseFormReturn<FormData>;
	isEditing: boolean;
	productId?: string;
}

export function ProductFormSection({ form, isEditing, productId }: ProductFormSectionProps) {
	const {
		fields: variantFields,
		append: appendVariant,
		remove: removeVariant,
		move: moveVariant,
		update: updateVariant
	} = useFieldArray({
		control: form.control,
		name: "variants",
	});

	// Load variants or existing prices when editing
	useEffect(() => {
		if (isEditing && productId) {
			// Fetch product variants/prices if editing
			const fetchProductPrices = async () => {
				try {
					const response = await apiClient.get(`/products/${productId}/prices`);
					const prices = response.data.prices || [];

					// Convert prices to variant format
					const variants = prices.map((price: any, index: number) => ({
						variantName: price.nickname || '',
						size: price.metadata?.size || '',
						color: price.metadata?.color || '',
						price: (price.unit_amount / 100).toFixed(2),
						sku: price.metadata?.sku || '',
						inStock: price.metadata?.in_stock !== 'false',
						sortOrder: parseInt(price.metadata?.sort_order || index.toString()),
					}));

					// Sort by sortOrder
					variants.sort((a: any, b: any) => a.sortOrder - b.sortOrder);

					// Update form with loaded variants
					if (variants.length > 0) {
						form.setValue('hasVariants', true);
						form.setValue('variants', variants);
					}
				} catch (error) {
					console.error('Error fetching product prices:', error);
					toast.error('Failed to load product variants')
				}
			};

			fetchProductPrices();
		}
	}, [isEditing, productId, form, toast]);

	const watchCategory = form.watch('category');
	const isApparel = watchCategory === 'apparel';

	const addVariant = () => {
		appendVariant({
			variantName: '',
			size: '',
			color: '',
			price: '',
			sku: '',
			inStock: true,
			sortOrder: variantFields.length,
		});
	};

	const moveVariantPosition = (index: number, direction: 'up' | 'down') => {
		const newIndex = direction === 'up' ? index - 1 : index + 1;

		if (newIndex < 0 || newIndex >= variantFields.length) return;

		moveVariant(index, newIndex);

		// Update sort order after moving
		variantFields.forEach((_, i) => {
			updateVariant(i, { ...form.getValues(`variants.${i}`), sortOrder: i });
		});
	};

	// Don't render if not a product
	if (form.watch('type') !== 'product') return null;

	return (
		<div className="space-y-6">
			<div className="grid md:grid-cols-2 gap-4">
				<FormField
					control={form.control}
					name="category"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Product Category</FormLabel>
							<Select
								value={field.value}
								onValueChange={field.onChange}
							>
								<FormControl>
									<SelectTrigger>
										<SelectValue placeholder="Select category" />
									</SelectTrigger>
								</FormControl>
								<SelectContent>
									<SelectItem value="merchandise">Merchandise</SelectItem>
									<SelectItem value="apparel">Apparel</SelectItem>
									<SelectItem value="accessories">Accessories</SelectItem>
									<SelectItem value="parts">Car Parts</SelectItem>
									<SelectItem value="collectibles">Collectibles</SelectItem>
								</SelectContent>
							</Select>
							<FormMessage />
						</FormItem>
					)}
				/>
				<FormField
					control={form.control}
					name="compareAtPrice"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Compare At Price (Optional)</FormLabel>
							<FormControl>
								<Input {...field} placeholder="39.99" />
							</FormControl>
							<FormDescription>Original price for sale items</FormDescription>
							<FormMessage />
						</FormItem>
					)}
				/>
			</div>

			<div className="grid md:grid-cols-2 gap-4">
				<FormField
					control={form.control}
					name="inStock"
					render={({ field }) => (
						<FormItem className="flex items-center space-x-2 mt-2">
							<FormControl>
								<Checkbox
									checked={field.value}
									onCheckedChange={field.onChange}
								/>
							</FormControl>
							<FormLabel className="font-normal cursor-pointer">
								In Stock
							</FormLabel>
						</FormItem>
					)}
				/>

				<FormField
					control={form.control}
					name="isNew"
					render={({ field }) => (
						<FormItem className="flex items-center space-x-2 mt-2">
							<FormControl>
								<Checkbox
									checked={field.value}
									onCheckedChange={field.onChange}
								/>
							</FormControl>
							<FormLabel className="font-normal cursor-pointer">
								New Product
							</FormLabel>
						</FormItem>
					)}
				/>
			</div>

			<Separator />

			<FormField
				control={form.control}
				name="hasVariants"
				render={({ field }) => (
					<FormItem className="flex items-center space-x-2">
						<FormControl>
							<Checkbox
								checked={field.value}
								onCheckedChange={(checked) => {
									field.onChange(checked);
									// Clear variants if unchecked
									if (!checked) {
										form.setValue('variants', []);
									}
								}}
							/>
						</FormControl>
						<FormLabel className="font-normal cursor-pointer">
							This product has multiple variants (sizes, colors, etc.)
						</FormLabel>
					</FormItem>
				)}
			/>

			{form.watch('hasVariants') ? (
				<div className="bg-muted/50 p-4 rounded-md space-y-4">
					<div className="flex justify-between items-center">
						<h3 className="text-lg font-semibold">Product Variants</h3>
						<Button type="button" onClick={addVariant} size="sm">
							<Plus className="w-4 h-4 mr-1" /> Add Variant
						</Button>
					</div>

					{variantFields.length === 0 ? (
						<div className="text-center py-8 text-muted-foreground">
							No variants added yet. Click "Add Variant" to create your first variant.
						</div>
					) : (
						<div className="space-y-4">
							{variantFields.map((field, index) => (
								<Card key={field.id} className="p-4">
									<div className="flex items-start justify-between mb-4">
										<div className="flex items-center gap-2">
											<GripVertical className="w-4 h-4 text-gray-400 cursor-move" />
											<h4 className="font-medium">Variant {index + 1}</h4>
										</div>
										<div className="flex gap-1">
											<Button
												type="button"
												size="sm"
												variant="ghost"
												onClick={() => moveVariantPosition(index, 'up')}
												disabled={index === 0}
											>
												↑
											</Button>
											<Button
												type="button"
												size="sm"
												variant="ghost"
												onClick={() => moveVariantPosition(index, 'down')}
												disabled={index === variantFields.length - 1}
											>
												↓
											</Button>
											<Button
												type="button"
												size="sm"
												variant="ghost"
												onClick={() => removeVariant(index)}
											>
												<Trash2 className="w-4 h-4" />
											</Button>
										</div>
									</div>

									<div className="grid md:grid-cols-2 gap-4">
										<FormField
											control={form.control}
											name={`variants.${index}.variantName`}
											render={({ field }) => (
												<FormItem>
													<FormLabel>Variant Name</FormLabel>
													<FormControl>
														<Input {...field} placeholder="e.g., Black T-Shirt - Large" />
													</FormControl>
													<FormMessage />
												</FormItem>
											)}
										/>

										<FormField
											control={form.control}
											name={`variants.${index}.price`}
											render={({ field }) => (
												<FormItem>
													<FormLabel>Price</FormLabel>
													<FormControl>
														<Input {...field} placeholder="29.99" />
													</FormControl>
													<FormMessage />
												</FormItem>
											)}
										/>

										{isApparel && (
											<>
												<FormField
													control={form.control}
													name={`variants.${index}.size`}
													render={({ field }) => (
														<FormItem>
															<FormLabel>Size</FormLabel>
															<FormControl>
																<Input {...field} placeholder="S, M, L, XL" />
															</FormControl>
														</FormItem>
													)}
												/>
												<FormField
													control={form.control}
													name={`variants.${index}.color`}
													render={({ field }) => (
														<FormItem>
															<FormLabel>Color</FormLabel>
															<FormControl>
																<Input {...field} placeholder="Black, White, etc." />
															</FormControl>
														</FormItem>
													)}
												/>
											</>
										)}

										<FormField
											control={form.control}
											name={`variants.${index}.sku`}
											render={({ field }) => (
												<FormItem>
													<FormLabel>SKU (Optional)</FormLabel>
													<FormControl>
														<Input {...field} placeholder="PROD-VAR-001" />
													</FormControl>
												</FormItem>
											)}
										/>

										<FormField
											control={form.control}
											name={`variants.${index}.inStock`}
											render={({ field }) => (
												<FormItem className="flex items-center space-x-2">
													<FormControl>
														<Checkbox checked={field.value} onCheckedChange={field.onChange} />
													</FormControl>
													<FormLabel className="font-normal cursor-pointer">In Stock</FormLabel>
												</FormItem>
											)}
										/>
									</div>
								</Card>
							))}
						</div>
					)}
				</div>
			) : (
				// Single price for products without variants
				<div className="grid md:grid-cols-2 gap-4">
					<FormField
						control={form.control}
						name="price"
						render={({ field }) => (
							<FormItem>
								<FormLabel>Price</FormLabel>
								<FormControl>
									<Input {...field} placeholder="29.99" />
								</FormControl>
								<FormMessage />
							</FormItem>
						)}
					/>
					<FormField
						control={form.control}
						name="maxQuantity"
						render={({ field }) => (
							<FormItem>
								<FormLabel>Max Quantity per Order</FormLabel>
								<FormControl>
									<Input {...field} placeholder="10" />
								</FormControl>
								<FormMessage />
							</FormItem>
						)}
					/>
				</div>
			)}
		</div>
	);
}
