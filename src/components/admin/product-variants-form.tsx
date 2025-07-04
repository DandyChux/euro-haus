import React from 'react';
import { UseFormReturn } from 'react-hook-form';
import { Plus, Trash2, GripVertical } from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Checkbox } from '~/components/ui/checkbox';
import { Card } from '~/components/ui/card';
import {
	FormControl,
	FormField,
	FormItem,
	FormLabel,
	FormMessage,
} from '~/components/ui/form';
import { FormData, ProductVariant } from '~/lib/schemas/product-schema';

interface ProductVariantsFormProps {
	form: UseFormReturn<FormData>;
}

export function ProductVariantsForm({ form }: ProductVariantsFormProps) {
	const watchHasVariants = form.watch('hasVariants');
	const watchCategory = form.watch('category');
	const isApparel = watchCategory === 'apparel';

	const addVariant = () => {
		const currentVariants = form.getValues('variants') || [];
		form.setValue('variants', [
			...currentVariants,
			{
				variantName: '',
				size: '',
				color: '',
				price: '',
				sku: '',
				inStock: true,
				sortOrder: currentVariants.length,
			},
		]);
	};

	const removeVariant = (index: number) => {
		const currentVariants = form.getValues('variants') || [];
		form.setValue('variants', currentVariants.filter((_, i) => i !== index));
	};

	const moveVariant = (index: number, direction: 'up' | 'down') => {
		const currentVariants = form.getValues('variants') || [];
		const newIndex = direction === 'up' ? index - 1 : index + 1;

		if (newIndex < 0 || newIndex >= currentVariants.length) return;

		const newVariants = [...currentVariants];
		[newVariants[index], newVariants[newIndex]] = [newVariants[newIndex], newVariants[index]];

		// Update sort order
		newVariants.forEach((variant, i) => {
			variant.sortOrder = i;
		});

		form.setValue('variants', newVariants);
	};

	if (form.watch('type') !== 'product') return null;

	return (
		<div className="space-y-4">
			<FormField
				control={form.control}
				name="hasVariants"
				render={({ field }) => (
					<FormItem className="flex items-center space-x-2">
						<FormControl>
							<Checkbox
								checked={field.value}
								onCheckedChange={field.onChange}
							/>
						</FormControl>
						<FormLabel className="font-normal cursor-pointer">
							This product has multiple variants (sizes, colors, etc.)
						</FormLabel>
					</FormItem>
				)}
			/>

			{watchHasVariants ? (
				<div className="space-y-4">
					<div className="flex justify-between items-center">
						<h3 className="text-lg font-semibold">Product Variants</h3>
						<Button type="button" onClick={addVariant} size="sm">
							<Plus className="w-4 h-4 mr-1" /> Add Variant
						</Button>
					</div>

					{form.watch('variants')?.map((variant, index) => (
						<Card key={index} className="p-4">
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
										onClick={() => moveVariant(index, 'up')}
										disabled={index === 0}
									>
										↑
									</Button>
									<Button
										type="button"
										size="sm"
										variant="ghost"
										onClick={() => moveVariant(index, 'down')}
										disabled={index === (form.watch('variants')?.length || 0) - 1}
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
							</FormItem>
						)}
					/>
				</div>
			)}
		</div>
	);
}
