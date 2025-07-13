import React from 'react';
import { UseFormReturn } from 'react-hook-form';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Textarea } from '~/components/ui/textarea';
import { Card } from '~/components/ui/card';
import { Checkbox } from '~/components/ui/checkbox';
import {
	FormControl,
	FormField,
	FormItem,
	FormLabel,
	FormMessage,
} from '~/components/ui/form';
import { FormData } from '~/lib/schemas/product-schema';

interface EventTiersFormProps {
	form: UseFormReturn<FormData>;
}

export function EventTiersForm({ form }: EventTiersFormProps) {
	const watchHasTiers = form.watch('hasTiers');

	const addTier = () => {
		const currentTiers = form.getValues('priceTiers') || [];
		form.setValue('priceTiers', [
			...currentTiers,
			{
				name: '',
				price: '',
				description: '',
				features: [],
				maxQuantity: undefined,
				sortOrder: currentTiers.length,
			},
		]);
	};

	const removeTier = (index: number) => {
		const currentTiers = form.getValues('priceTiers') || [];
		form.setValue('priceTiers', currentTiers.filter((_, i) => i !== index));
	};

	const addFeature = (tierIndex: number) => {
		const currentTiers = form.getValues('priceTiers') || [];
		const tier = currentTiers[tierIndex];
		if (!tier.features) tier.features = [];
		tier.features.push('');
		form.setValue('priceTiers', [...currentTiers]);
	};

	const updateFeature = (tierIndex: number, featureIndex: number, value: string) => {
		const currentTiers = form.getValues('priceTiers') || [];
		const tier = currentTiers[tierIndex];
		if (tier.features) {
			tier.features[featureIndex] = value;
			form.setValue('priceTiers', [...currentTiers]);
		}
	};

	const removeFeature = (tierIndex: number, featureIndex: number) => {
		const currentTiers = form.getValues('priceTiers') || [];
		const tier = currentTiers[tierIndex];
		if (tier.features) {
			tier.features.splice(featureIndex, 1);
			form.setValue('priceTiers', [...currentTiers]);
		}
	};

	if (form.watch('type') !== 'event') return null;

	return (
		<div className="space-y-4">
			<FormField
				control={form.control}
				name="hasTiers"
				render={({ field }) => (
					<FormItem className="flex items-center space-x-2">
						<FormControl>
							<Checkbox
								checked={field.value}
								onCheckedChange={field.onChange}
							/>
						</FormControl>
						<FormLabel className="font-normal cursor-pointer">
							This event has multiple ticket tiers (VIP, General, Student, etc.)
						</FormLabel>
					</FormItem>
				)}
			/>

			{watchHasTiers ? (
				<div className="space-y-4">
					<div className="flex justify-between items-center">
						<h3 className="text-lg font-semibold">Ticket Tiers</h3>
						<Button type="button" onClick={addTier} size="sm">
							<Plus className="w-4 h-4 mr-1" /> Add Tier
						</Button>
					</div>

					{form.watch('priceTiers')?.map((tier, index) => (
						<Card key={index} className="p-4">
							<div className="flex items-start justify-between mb-4">
								<h4 className="font-medium">Tier {index + 1}</h4>
								<Button
									type="button"
									size="sm"
									variant="ghost"
									onClick={() => removeTier(index)}
								>
									<Trash2 className="w-4 h-4" />
								</Button>
							</div>

							<div className="space-y-4">
								<div className="grid md:grid-cols-2 gap-4">
									<FormField
										control={form.control}
										name={`priceTiers.${index}.name`}
										render={({ field }) => (
											<FormItem>
												<FormLabel>Tier Name</FormLabel>
												<FormControl>
													<Input {...field} placeholder="VIP, General Admission, etc." />
												</FormControl>
												<FormMessage />
											</FormItem>
										)}
									/>

									<FormField
										control={form.control}
										name={`priceTiers.${index}.price`}
										render={({ field }) => (
											<FormItem>
												<FormLabel>Price</FormLabel>
												<FormControl>
													<Input {...field} placeholder="99.99" />
												</FormControl>
												<FormMessage />
											</FormItem>
										)}
									/>
								</div>

								<FormField
									control={form.control}
									name={`priceTiers.${index}.description`}
									render={({ field }) => (
										<FormItem>
											<FormLabel>Description (Optional)</FormLabel>
											<FormControl>
												<Textarea
													{...field}
													placeholder="What's included in this tier..."
													rows={2}
												/>
											</FormControl>
										</FormItem>
									)}
								/>

								<FormField
									control={form.control}
									name={`priceTiers.${index}.maxQuantity`}
									render={({ field }) => (
										<FormItem>
											<FormLabel>Max Tickets Available (Optional)</FormLabel>
											<FormControl>
												<Input {...field} placeholder="50" />
											</FormControl>
											<FormMessage />
										</FormItem>
									)}
								/>

								<div className="space-y-2">
									<div className="flex justify-between items-center">
										<FormLabel>Features</FormLabel>
										<Button
											type="button"
											size="sm"
											variant="ghost"
											onClick={() => addFeature(index)}
										>
											<Plus className="w-4 h-4" />
										</Button>
									</div>
									{tier.features?.map((feature, featureIndex) => (
										<div key={featureIndex} className="flex gap-2">
											<Input
												value={feature}
												onChange={(e) => updateFeature(index, featureIndex, e.target.value)}
												placeholder="e.g., Meet & Greet, Premium Seating"
											/>
											<Button
												type="button"
												size="sm"
												variant="ghost"
												onClick={() => removeFeature(index, featureIndex)}
											>
												<Trash2 className="w-4 h-4" />
											</Button>
										</div>
									))}
								</div>
							</div>
						</Card>
					))}
				</div>
			) : (
				// Single price for events without tiers
				<div className="grid md:grid-cols-2 gap-4">
					<FormField
						control={form.control}
						name="price"
						render={({ field }) => (
							<FormItem>
								<FormLabel>Ticket Price</FormLabel>
								<FormControl>
									<Input {...field} placeholder="49.99" />
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
								<FormLabel>Max Tickets per Order</FormLabel>
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
