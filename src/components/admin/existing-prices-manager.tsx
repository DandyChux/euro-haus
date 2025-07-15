import React, { useState, useEffect } from 'react';
import { UseFormReturn } from 'react-hook-form';
import { Trash2, Edit2, Check, X, Star, StarOff, AlertTriangle, Car, Medal, Plus, Minus, List } from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Card } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';
import { FormData } from '~/lib/schemas/product-schema';
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from '~/components/ui/tooltip';
import { Alert, AlertDescription } from '../ui/alert';
import { Switch } from '~/components/ui/switch';
import { Label } from '~/components/ui/label';
import { Textarea } from '../ui/textarea';

interface ExistingPrice {
	id: string;
	nickname: string;
	unit_amount: number;
	currency: string;
	active: boolean;
	metadata: Record<string, any>;
	isDefault?: boolean;
}

interface ExistingPricesManagerProps {
	productId: string;
	productType: 'product' | 'event';
	form: UseFormReturn<FormData>;
}

export function ExistingPricesManager({ productId, productType, form }: ExistingPricesManagerProps) {
	const [prices, setPrices] = useState<ExistingPrice[]>([]);
	const [loading, setLoading] = useState(true);
	const [editingPrice, setEditingPrice] = useState<string | null>(null);
	const [editValues, setEditValues] = useState<Record<string, any>>({});
	const [defaultPriceId, setDefaultPriceId] = useState<string | null>(null);

	useEffect(() => {
		fetchPrices();
	}, [productId]);

	const fetchPrices = async () => {
		try {
			setLoading(true);
			const [pricesResponse, productResponse] = await Promise.all([
				apiClient.get(`/products/${productId}/prices`),
				apiClient.get(`/products/${productId}`)
			]);

			const fetchedPrices = pricesResponse.data.prices || [];
			const defaultPrice = productResponse.data.default_price?.id;

			const pricesWithDefault = fetchedPrices.map((price: any) => ({
				...price,
				isDefault: price.id === defaultPrice
			}));

			setPrices(pricesWithDefault);
			setDefaultPriceId(defaultPrice);
		} catch (error) {
			console.error('Failed to fetch prices:', error);
			toast.error('Failed to load existing prices');
		} finally {
			setLoading(false);
		}
	};

	const handleSetDefault = async (priceId: string, event?: React.MouseEvent<HTMLButtonElement>) => {
		event?.preventDefault();
		event?.stopPropagation();

		try {
			await apiClient.put(`/admin/set-default-price/${productId}`, { priceId });
			toast.success('Default price updated successfully');
			setTimeout(fetchPrices, 500);
		} catch (error) {
			console.error('Failed to set default price:', error);
			toast.error('Failed to set default price');
		}
	};

	const handleEdit = (priceId: string) => {
		const price = prices.find(p => p.id === priceId);
		if (price) {
			// Parse features if they exist
			let features = [];
			try {
				features = price.metadata?.features ? JSON.parse(price.metadata.features) : [];
			} catch (e) {
				features = [];
			}

			setEditingPrice(priceId);
			setEditValues({
				[priceId]: {
					nickname: price.nickname || '',
					description: price.metadata?.description || '',
					requiresVehicleSubmission: price.metadata?.requires_vehicle_submission === 'true',
					isMostPopular: price.metadata?.is_most_popular === 'true',
					features: features
				}
			});
		}
	};

	const handleCancelEdit = () => {
		setEditingPrice(null);
		setEditValues({});
	};

	const handleAddFeature = (priceId: string) => {
		const currentFeatures = editValues[priceId]?.features || [];
		setEditValues({
			...editValues,
			[priceId]: {
				...editValues[priceId],
				features: [...currentFeatures, '']
			}
		});
	};

	const handleRemoveFeature = (priceId: string, index: number) => {
		const currentFeatures = editValues[priceId]?.features || [];
		const newFeatures = currentFeatures.filter((_: any, i: any) => i !== index);
		setEditValues({
			...editValues,
			[priceId]: {
				...editValues[priceId],
				features: newFeatures
			}
		});
	};

	const handleFeatureChange = (priceId: string, index: number, value: string) => {
		const currentFeatures = editValues[priceId]?.features || [];
		const newFeatures = [...currentFeatures];
		newFeatures[index] = value;
		setEditValues({
			...editValues,
			[priceId]: {
				...editValues[priceId],
				features: newFeatures
			}
		});
	};

	const handleSaveEdit = async (priceId: string) => {
		try {
			const values = editValues[priceId];
			if (!values) return;

			// Filter out empty features
			const features = (values.features || []).filter((f: string) => f.trim() !== '');

			const currentPrice = prices.find(p => p.id === priceId);
			const metadata = {
				...currentPrice?.metadata,
				description: values.description || '',
				requires_vehicle_submission: String(values.requiresVehicleSubmission),
				is_most_popular: String(values.isMostPopular),
				features: JSON.stringify(features),
				updated_at: new Date().toISOString(),
			};

			await apiClient.put(`/admin/update-price/${priceId}`, {
				nickname: values.nickname,
				metadata,
			});

			toast.success('Price updated successfully');
			setEditingPrice(null);
			setEditValues({});
			setTimeout(fetchPrices, 2500);

		} catch (error) {
			console.error('Failed to update price:', error);
			toast.error('Failed to update price');
		}
	};

	const handleDelete = async (priceId: string) => {
		const price = prices.find(p => p.id === priceId);

		if (price?.isDefault) {
			toast.error('Cannot archive the default price. Please set another price as default first.');
			return;
		}

		if (!confirm('Are you sure you want to archive this price? This action cannot be undone.')) {
			return;
		}

		try {
			await apiClient.put(`/admin/archive-price/${priceId}`, { active: false });
			toast.success('Price archived successfully');
			fetchPrices();
		} catch (error) {
			console.error('Failed to archive price:', error);
			toast.error('Failed to archive price');
		}
	};

	const renderFeatures = (price: ExistingPrice) => {
		let features = [];
		try {
			features = price.metadata?.features ? JSON.parse(price.metadata.features) : [];
		} catch (e) {
			features = [];
		}

		if (features.length === 0) return null;

		return (
			<div className="mt-2 text-sm text-muted-foreground">
				<div className="flex items-center gap-1 mb-1">
					<List className="w-3 h-3" />
					<span className="font-medium">Features:</span>
				</div>
				<ul className="list-disc list-inside ml-4 space-y-0.5">
					{features.map((feature: string, index: number) => (
						<li key={index} className="text-xs">{feature}</li>
					))}
				</ul>
			</div>
		);
	};

	const renderDescription = (description: string | undefined) => {
		if (!description) return null;

		// Split by double line breaks to create paragraphs
		const paragraphs = description.split(/\n\n+/).filter(p => p.trim());

		return (
			<div className="text-sm text-muted-foreground mt-1 space-y-1">
				{paragraphs.map((paragraph, index) => (
					<p key={index} className="whitespace-pre-wrap">
						{paragraph.trim()}
					</p>
				))}
			</div>
		);
	};


	if (loading) {
		return <div className="text-sm text-muted-foreground">Loading existing prices...</div>;
	}

	if (prices.length === 0) {
		return null;
	}

	return (
		<div className="space-y-4">
			<h3 className="text-lg font-semibold">Existing Prices/Tiers</h3>
			<div className="space-y-2">
				{prices.map((price) => (
					<Card key={price.id} className="p-4">
						{editingPrice === price.id ? (
							<div className="space-y-4">
								<div className="grid grid-cols-2 gap-4">
									<Input
										value={editValues[price.id]?.nickname || ''}
										onChange={(e) => setEditValues({
											...editValues,
											[price.id]: { ...editValues[price.id], nickname: e.target.value }
										})}
										placeholder="Price name"
									/>
									<div className="flex items-center gap-2">
										<span className="text-sm text-muted-foreground">$</span>
										<span className="font-medium">{(price.unit_amount / 100).toFixed(2)}</span>
										<span className="text-sm text-muted-foreground">(price cannot be changed)</span>
									</div>
								</div>

								{productType === 'event' && (
									<>
										<div>
											<Label htmlFor={`description-${price.id}`} className="mb-2">
												Description (Optional)
											</Label>
											<Textarea
												id={`description-${price.id}`}
												value={editValues[price.id]?.description || ''}
												onChange={(e) => setEditValues(prev => ({
													...prev,
													[price.id]: { ...prev[price.id], description: e.target.value }
												}))}
												placeholder="Describe what's included in this tier..."
												rows={3}
											/>
										</div>

										{/* Features Management */}
										<div>
											<div className="flex items-center justify-between mb-2">
												<Label>Features (Optional)</Label>
												<Button
													type="button"
													size="sm"
													variant="outline"
													onClick={() => handleAddFeature(price.id)}
													className="h-8"
												>
													<Plus className="w-4 h-4 mr-1" /> Add Feature
												</Button>
											</div>
											<div className="space-y-2">
												{(editValues[price.id]?.features || []).map((feature: string, index: number) => (
													<div key={index} className="flex items-center gap-2">
														<Input
															value={feature}
															onChange={(e) => handleFeatureChange(price.id, index, e.target.value)}
															placeholder={`Feature ${index + 1}`}
															className="flex-1"
														/>
														<Button
															type="button"
															size="sm"
															variant="ghost"
															onClick={() => handleRemoveFeature(price.id, index)}
															className="h-9 w-9 p-0"
														>
															<Minus className="w-4 h-4" />
														</Button>
													</div>
												))}
												{(!editValues[price.id]?.features || editValues[price.id].features.length === 0) && (
													<p className="text-sm text-muted-foreground">No features added yet</p>
												)}
											</div>
										</div>

										<div className="flex flex-col space-y-3">
											<div className="flex items-center space-x-2">
												<Switch
													id={`requires-vehicle-${price.id}`}
													checked={editValues[price.id]?.requiresVehicleSubmission}
													onCheckedChange={(checked) => setEditValues({
														...editValues,
														[price.id]: { ...editValues[price.id], requiresVehicleSubmission: checked }
													})}
												/>
												<Label htmlFor={`requires-vehicle-${price.id}`}>
													Requires Vehicle Submission?
												</Label>
											</div>
											<div className="flex items-center space-x-2">
												<Switch
													id={`is-popular-${price.id}`}
													checked={editValues[price.id]?.isMostPopular}
													onCheckedChange={(checked) => setEditValues({
														...editValues,
														[price.id]: { ...editValues[price.id], isMostPopular: checked }
													})}
												/>
												<Label htmlFor={`is-popular-${price.id}`}>
													Most Popular Tier?
												</Label>
											</div>
										</div>
									</>
								)}

								<div className="flex gap-2">
									<Button type='button' size="sm" onClick={() => handleSaveEdit(price.id)}>
										<Check className="w-4 h-4 mr-1" /> Save
									</Button>
									<Button type='button' size="sm" variant="outline" onClick={handleCancelEdit}>
										<X className="w-4 h-4 mr-1" /> Cancel
									</Button>
								</div>
							</div>
						) : (
							<div className="flex items-center justify-between">
								<div className="flex-1">
									<div className="flex items-center gap-3">
										<div className="flex-1">
											<div className="font-medium flex items-center gap-2">
												{price.nickname || `${productType === 'event' ? 'Ticket' : 'Variant'} - $${(price.unit_amount / 100).toFixed(2)}`}
												{price.isDefault && (
													<Badge variant="default" className="text-xs">
														<Star className="w-3 h-3 mr-1" /> Default
													</Badge>
												)}
												{price.metadata?.is_most_popular === 'true' && (
													<Badge variant="secondary" className="text-xs">
														<Medal className="w-3 h-3 mr-1" /> Most Popular
													</Badge>
												)}
											</div>
											<div className="text-sm text-muted-foreground">
												${(price.unit_amount / 100).toFixed(2)} {price.currency.toUpperCase()}
												{productType === 'event' && price.metadata?.requires_vehicle_submission === 'true' && (
													<TooltipProvider>
														<Tooltip>
															<TooltipTrigger asChild>
																<Car className="w-4 h-4 ml-2 inline-block text-blue-500" />
															</TooltipTrigger>
															<TooltipContent>
																<p>Requires vehicle submission</p>
															</TooltipContent>
														</Tooltip>
													</TooltipProvider>
												)}
											</div>
											{price.metadata?.description && (
												renderDescription(price.metadata.description)
											)}
											{renderFeatures(price)}
										</div>
									</div>
								</div>
								<div className="flex items-center gap-2 ml-4">
									{!price.active && <Badge variant="secondary">Archived</Badge>}

									{price.active && !price.isDefault && (
										<TooltipProvider>
											<Tooltip>
												<TooltipTrigger asChild>
													<Button
														size="sm"
														type='button'
														variant="ghost"
														onClick={() => handleSetDefault(price.id)}
													>
														<StarOff className="w-4 h-4" />
													</Button>
												</TooltipTrigger>
												<TooltipContent>
													<p>Set as default price</p>
												</TooltipContent>
											</Tooltip>
										</TooltipProvider>
									)}

									<Button
										size="sm"
										type='button'
										variant="ghost"
										onClick={() => handleEdit(price.id)}
										disabled={!price.active}
									>
										<Edit2 className="w-4 h-4" />
									</Button>
									<Button
										size="sm"
										type='button'
										variant="ghost"
										onClick={() => handleDelete(price.id)}
										disabled={!price.active || price.isDefault}
									>
										<Trash2 className="w-4 h-4" />
									</Button>
								</div>
							</div>
						)}
					</Card>
				))}
			</div>

			{prices.some(p => p.isDefault && !p.active) && (
				<Alert variant="warning">
					<AlertTriangle className="h-4 w-4" />
					<AlertDescription>
						The default price is archived. Please set a new default price.
					</AlertDescription>
				</Alert>
			)}
		</div>
	);
}
