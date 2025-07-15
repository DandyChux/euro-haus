import { useEffect } from 'react';
import { Card, CardContent } from '~/components/ui/card';
import { FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from '~/components/ui/form';
import { Input } from '~/components/ui/input';
import { Textarea } from '~/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { Checkbox } from '~/components/ui/checkbox';
import { Separator } from '~/components/ui/separator';
import { Button } from '~/components/ui/button';
import { useFieldArray, UseFormReturn } from 'react-hook-form';
import { Plus, Trash2 } from 'lucide-react';
import { FormData } from '~/lib/schemas/product-schema';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';

interface EventFormSectionProps {
	form: UseFormReturn<FormData>;
	isEditing: boolean;
	eventId?: string;
	onGenerateSlug: () => void;
}

export function EventFormSection({ form, isEditing, eventId, onGenerateSlug }: EventFormSectionProps) {
	const {
		fields: tierFields,
		append: appendTier,
		remove: removeTier,
		update: updateTier
	} = useFieldArray({
		control: form.control,
		name: "priceTiers",
	});

	const {
		fields: tagFields,
		append: appendTag,
		remove: removeTag
	} = useFieldArray({
		control: form.control,
		name: "tags",
	});

	const {
		fields: agendaFields,
		append: appendAgenda,
		remove: removeAgenda
	} = useFieldArray({
		control: form.control,
		name: "agenda",
	});

	const {
		fields: includesFields,
		append: appendIncludes,
		remove: removeIncludes
	} = useFieldArray({
		control: form.control,
		name: "includes",
	});

	const {
		fields: sponsorFields,
		append: appendSponsor,
		remove: removeSponsor
	} = useFieldArray({
		control: form.control,
		name: "sponsors",
	});

	const {
		fields: sponsorTierFields,
		append: appendSponsorTier,
		remove: removeSponsorTier,
		update: updateSponsorTier,
		move: moveSponsorTier
	} = useFieldArray({
		control: form.control,
		name: "sponsorTiers",
	});

	// Load price tiers when editing
	useEffect(() => {
		if (isEditing && eventId) {
			// Fetch event price tiers if editing
			const fetchEventPrices = async () => {
				try {
					const response = await apiClient.get(`/products/${eventId}/prices`);
					const prices = response.data.prices || [];

					// Convert prices to tier format
					const tiers = prices.map((price: any, index: number) => ({
						name: price.nickname || '',
						price: (price.unit_amount / 100).toFixed(2),
						description: price.metadata?.description || '',
						features: JSON.parse(price.metadata?.features || '[]'),
						maxQuantity: price.metadata?.max_quantity || '',
						sortOrder: parseInt(price.metadata?.sort_order || index.toString()),
					}));

					// Sort by sortOrder
					tiers.sort((a: any, b: any) => a.sortOrder - b.sortOrder);

					// Update form with loaded tiers
					if (tiers.length > 0) {
						form.setValue('hasTiers', true);
						form.setValue('priceTiers', tiers);
					}
				} catch (error) {
					console.error('Error fetching event price tiers:', error);
					toast.error('Failed to load event tiers');
				}
			};

			fetchEventPrices();
		}
	}, [isEditing, eventId, form]);

	// Helper function to add features to a tier
	const addFeatureToTier = (tierIndex: number) => {
		const currentTier = form.getValues(`priceTiers.${tierIndex}`);
		const updatedFeatures = [...(currentTier.features || []), ''];
		updateTier(tierIndex, { ...currentTier, features: updatedFeatures });
	};

	// Helper function to update a feature in a tier
	const updateFeatureInTier = (tierIndex: number, featureIndex: number, value: string) => {
		const currentTier = form.getValues(`priceTiers.${tierIndex}`);
		const updatedFeatures = [...(currentTier.features || [])];
		updatedFeatures[featureIndex] = value;
		updateTier(tierIndex, { ...currentTier, features: updatedFeatures });
	};

	// Helper function to remove a feature from a tier
	const removeFeatureFromTier = (tierIndex: number, featureIndex: number) => {
		const currentTier = form.getValues(`priceTiers.${tierIndex}`);
		const updatedFeatures = (currentTier.features || []).filter((_: any, i: number) => i !== featureIndex);
		updateTier(tierIndex, { ...currentTier, features: updatedFeatures });
	};

	// Helper functions for nested sponsor management
	const addSponsorToTier = (tierIndex: number) => {
		const currentTier = form.getValues(`sponsorTiers.${tierIndex}`);
		const updatedSponsors = [...(currentTier.sponsors || []), { name: '', logoUrl: '', link: '' }];
		updateSponsorTier(tierIndex, { ...currentTier, sponsors: updatedSponsors });
	};

	const removeSponsorFromTier = (tierIndex: number, sponsorIndex: number) => {
		const currentTier = form.getValues(`sponsorTiers.${tierIndex}`);
		const updatedSponsors = currentTier.sponsors.filter((_: any, i: number) => i !== sponsorIndex);
		updateSponsorTier(tierIndex, { ...currentTier, sponsors: updatedSponsors });
	};

	// Don't render if not an event
	if (form.watch('type') !== 'event') return null;

	return (
		<div className="space-y-6">
			<FormField
				control={form.control}
				name="slug"
				render={({ field }) => (
					<FormItem>
						<FormLabel>Event URL Slug</FormLabel>
						<div className="flex space-x-2">
							<FormControl>
								<Input {...field} placeholder="porsche-club-meetup-june-2025" />
							</FormControl>
							<Button
								type="button"
								variant="outline"
								onClick={onGenerateSlug}
								disabled={!form.getValues('name') || !form.getValues('eventDate')}
							>
								Generate
							</Button>
						</div>
						<FormDescription>Used in the event URL: /events/[slug]</FormDescription>
						<FormMessage />
					</FormItem>
				)}
			/>

			<div className="grid md:grid-cols-2 gap-4">
				<FormField
					control={form.control}
					name="eventDate"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Event Date</FormLabel>
							<FormControl>
								<Input
									{...field}
									type="date"
									onChange={(e) => {
										field.onChange(e);
										// Update slug when date changes
										if (form.getValues('name')) {
											setTimeout(onGenerateSlug, 100);
										}
									}}
								/>
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>

				<FormField
					control={form.control}
					name="eventTime"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Start Time</FormLabel>
							<FormControl>
								<Input {...field} type="time" />
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>
			</div>

			<FormField
				control={form.control}
				name="location"
				render={({ field }) => (
					<FormItem>
						<FormLabel>Location</FormLabel>
						<FormControl>
							<Input {...field} placeholder="Euro Haus Headquarters, Tampa, FL" />
						</FormControl>
						<FormMessage />
					</FormItem>
				)}
			/>

			<div className="grid md:grid-cols-3 gap-4">
				<FormField
					control={form.control}
					name="capacity"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Total Capacity</FormLabel>
							<FormControl>
								<Input
									{...field}
									type="number"
									min="1"
									onChange={(e) => {
										field.onChange(e);
										// Auto-update available spots if not set
										if (!form.getValues('availableSpots')) {
											form.setValue('availableSpots', e.target.value);
										}
									}}
								/>
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>

				<FormField
					control={form.control}
					name="availableSpots"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Available Spots</FormLabel>
							<FormControl>
								<Input {...field} type="number" min="0" />
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>

				<FormField
					control={form.control}
					name="organizer"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Organizer</FormLabel>
							<FormControl>
								<Input {...field} placeholder="Euro Haus Events Team" />
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>
			</div>

			<FormField
				control={form.control}
				name="status"
				render={({ field }) => (
					<FormItem>
						<FormLabel>Event Status</FormLabel>
						<Select
							value={field.value}
							onValueChange={field.onChange}
						>
							<FormControl>
								<SelectTrigger>
									<SelectValue placeholder="Select status" />
								</SelectTrigger>
							</FormControl>
							<SelectContent>
								<SelectItem value="upcoming">Upcoming</SelectItem>
								<SelectItem value="ongoing">Ongoing</SelectItem>
								<SelectItem value="completed">Completed</SelectItem>
								<SelectItem value="cancelled">Cancelled</SelectItem>
								<SelectItem value="sold-out">Sold Out</SelectItem>
							</SelectContent>
						</Select>
						<FormMessage />
					</FormItem>
				)}
			/>

			<Separator />

			{/* Tags */}
			<div className="bg-muted/50 p-4 rounded-md">
				<div className="flex justify-between items-center mb-3">
					<h3 className="text-lg font-medium">Event Tags</h3>
					<Button
						type="button"
						size="sm"
						onClick={() => appendTag({ value: '' })}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Tag
					</Button>
				</div>

				{tagFields.length === 0 ? (
					<div className="text-center py-4 text-muted-foreground">
						No tags added yet. Add tags to help categorize your event.
					</div>
				) : (
					<div className="space-y-2">
						{tagFields.map((field, index) => (
							<div key={field.id} className="flex items-end gap-2">
								<FormField
									control={form.control}
									name={`tags.${index}.value`}
									render={({ field }) => (
										<FormItem className="flex-1">
											<FormControl>
												<Input {...field} placeholder="e.g., BMW, Track Day, Networking" />
											</FormControl>
											<FormMessage />
										</FormItem>
									)}
								/>
								<Button
									type="button"
									variant="ghost"
									size="icon"
									onClick={() => removeTag(index)}
								>
									<Trash2 className="h-4 w-4" />
								</Button>
							</div>
						))}
					</div>
				)}
			</div>

			{/* Event Schedule/Agenda */}
			<div className="bg-muted/50 p-4 rounded-md">
				<div className="flex justify-between items-center mb-3">
					<h3 className="text-lg font-medium">Event Schedule</h3>
					<Button
						type="button"
						size="sm"
						onClick={() => appendAgenda({ time: '', activity: '' })}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Item
					</Button>
				</div>

				{agendaFields.length === 0 ? (
					<div className="text-center py-4 text-muted-foreground">
						No schedule items added yet. Add your event timeline.
					</div>
				) : (
					<div className="space-y-2">
						{agendaFields.map((field, index) => (
							<div key={field.id} className="flex gap-2">
								<FormField
									control={form.control}
									name={`agenda.${index}.time`}
									render={({ field }) => (
										<FormItem className="w-32">
											<FormControl>
												<Input {...field} placeholder="9:00 AM" />
											</FormControl>
											<FormMessage />
										</FormItem>
									)}
								/>
								<FormField
									control={form.control}
									name={`agenda.${index}.activity`}
									render={({ field }) => (
										<FormItem className="flex-1">
											<FormControl>
												<Input {...field} placeholder="Registration & Welcome Coffee" />
											</FormControl>
											<FormMessage />
										</FormItem>
									)}
								/>
								<Button
									type="button"
									size="icon"
									variant="ghost"
									onClick={() => removeAgenda(index)}
								>
									<Trash2 className="h-4 w-4" />
								</Button>
							</div>
						))}
					</div>
				)}
			</div>

			{/* What's Included */}
			<div className="bg-muted/50 p-4 rounded-md">
				<div className="flex justify-between items-center mb-3">
					<h3 className="text-lg font-medium">What's Included</h3>
					<Button
						type="button"
						size="sm"
						onClick={() => appendIncludes({ value: '' })}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Item
					</Button>
				</div>

				{includesFields.length === 0 ? (
					<div className="text-center py-4 text-muted-foreground">
						No items added yet. List what's included with the ticket.
					</div>
				) : (
					<div className="space-y-2">
						{includesFields.map((field, index) => (
							<div key={field.id} className="flex gap-2">
								<FormField
									control={form.control}
									key={field.id}
									name={`includes.${index}.value`}
									render={({ field }) => (
										<FormItem className="flex-1">
											<FormControl>
												<Input {...field} placeholder="e.g., Lunch and refreshments" />
											</FormControl>
											<FormMessage />
										</FormItem>
									)}
								/>
								<Button
									type="button"
									size="icon"
									variant="ghost"
									onClick={() => removeIncludes(index)}
								>
									<Trash2 className="h-4 w-4" />
								</Button>
							</div>
						))}
					</div>
				)}
			</div>

			{/* Sponsor Tiers */}
			<div className="bg-muted/50 p-4 rounded-md">
				<div className="flex justify-between items-center mb-3">
					<div>
						<h3 className="text-lg font-medium">Event Sponsors</h3>
						<p className="text-sm text-muted-foreground">Organize sponsors by tier (e.g., Platinum, Gold, Silver)</p>
					</div>
					<Button
						type="button"
						size="sm"
						onClick={() => appendSponsorTier({
							tierName: '',
							displayOrder: sponsorTierFields.length,
							sponsors: []
						})}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Tier
					</Button>
				</div>

				{sponsorTierFields.length === 0 ? (
					<div className="text-center py-4 text-muted-foreground">
						No sponsor tiers added yet. Add tiers to organize sponsors by level.
					</div>
				) : (
					<div className="space-y-6">
						{sponsorTierFields.map((tierField, tierIndex) => (
							<Card key={tierField.id} className="p-4">
								<div className="space-y-4">
									<div className="flex items-start justify-between">
										<div className="flex-1 space-y-3">
											<FormField
												control={form.control}
												name={`sponsorTiers.${tierIndex}.tierName`}
												render={({ field }) => (
													<FormItem>
														<FormLabel>Tier Name</FormLabel>
														<FormControl>
															<Input
																{...field}
																placeholder="e.g., Platinum Sponsors, Gold Sponsors"
															/>
														</FormControl>
														<FormMessage />
													</FormItem>
												)}
											/>
										</div>
										<div className="flex items-center gap-2 ml-4">
											<Button
												type="button"
												size="sm"
												variant="ghost"
												onClick={() => moveSponsorTier(tierIndex, Math.max(0, tierIndex - 1))}
												disabled={tierIndex === 0}
											>
												↑
											</Button>
											<Button
												type="button"
												size="sm"
												variant="ghost"
												onClick={() => moveSponsorTier(tierIndex, Math.min(sponsorTierFields.length - 1, tierIndex + 1))}
												disabled={tierIndex === sponsorTierFields.length - 1}
											>
												↓
											</Button>
											<Button
												type="button"
												size="sm"
												variant="ghost"
												onClick={() => removeSponsorTier(tierIndex)}
											>
												<Trash2 className="h-4 w-4" />
											</Button>
										</div>
									</div>

									{/* Sponsors within this tier */}
									<div className="space-y-3">
										<div className="flex justify-between items-center">
											<h4 className="text-sm font-medium text-muted-foreground">Sponsors in this tier</h4>
											<Button
												type="button"
												size="sm"
												variant="outline"
												onClick={() => addSponsorToTier(tierIndex)}
											>
												<Plus className="h-4 w-4 mr-1" /> Add Sponsor
											</Button>
										</div>

										{form.watch(`sponsorTiers.${tierIndex}.sponsors`)?.length === 0 ? (
											<div className="text-center py-3 text-sm text-muted-foreground border-2 border-dashed rounded-md">
												No sponsors in this tier yet
											</div>
										) : (
											<div className="grid gap-3">
												{form.watch(`sponsorTiers.${tierIndex}.sponsors`)?.map((_, sponsorIndex) => (
													<Card key={sponsorIndex} className="p-3 bg-background">
														<div className="space-y-3">
															<div className="flex justify-between items-start">
																<h5 className="text-sm font-medium">Sponsor {sponsorIndex + 1}</h5>
																<Button
																	type="button"
																	size="icon"
																	variant="ghost"
																	className="h-6 w-6"
																	onClick={() => removeSponsorFromTier(tierIndex, sponsorIndex)}
																>
																	<Trash2 className="h-3 w-3" />
																</Button>
															</div>
															<div className="grid gap-3">
																<FormField
																	control={form.control}
																	name={`sponsorTiers.${tierIndex}.sponsors.${sponsorIndex}.name`}
																	render={({ field }) => (
																		<FormItem>
																			<FormLabel className="text-xs">Company Name</FormLabel>
																			<FormControl>
																				<Input {...field} placeholder="e.g., Porsche USA" />
																			</FormControl>
																			<FormMessage />
																		</FormItem>
																	)}
																/>
																<FormField
																	control={form.control}
																	name={`sponsorTiers.${tierIndex}.sponsors.${sponsorIndex}.logoUrl`}
																	render={({ field }) => (
																		<FormItem>
																			<FormLabel className="text-xs">Logo URL</FormLabel>
																			<FormControl>
																				<Input {...field} placeholder="https://example.com/logo.png" />
																			</FormControl>
																			<FormMessage />
																		</FormItem>
																	)}
																/>
																<FormField
																	control={form.control}
																	name={`sponsorTiers.${tierIndex}.sponsors.${sponsorIndex}.link`}
																	render={({ field }) => (
																		<FormItem>
																			<FormLabel className="text-xs">Website (optional)</FormLabel>
																			<FormControl>
																				<Input {...field} placeholder="https://www.porsche.com" />
																			</FormControl>
																			<FormMessage />
																		</FormItem>
																	)}
																/>
															</div>
														</div>
													</Card>
												))}
											</div>
										)}
									</div>
								</div>
							</Card>
						))}
					</div>
				)}
			</div>

			<Separator />

			{/* Pricing Section */}
			<FormField
				control={form.control}
				name="hasTiers"
				render={({ field }) => (
					<FormItem className="flex items-center space-x-2">
						<FormControl>
							<Checkbox
								checked={field.value}
								onCheckedChange={(checked) => {
									field.onChange(checked);
									// Clear tiers if unchecked
									if (!checked) {
										form.setValue('priceTiers', []);
									}
								}}
							/>
						</FormControl>
						<FormLabel className="font-normal cursor-pointer">
							This event has multiple ticket tiers (VIP, General, Student, etc.)
						</FormLabel>
					</FormItem>
				)}
			/>

			{form.watch('hasTiers') ? (
				<div className="bg-muted/50 p-4 rounded-md">
					<div className="flex justify-between items-center mb-4">
						<h3 className="text-lg font-medium">Ticket Tiers</h3>
						<Button
							type="button"
							size="sm"
							onClick={() => appendTier({
								name: '',
								price: '',
								description: '',
								features: [],
								maxQuantity: '',
								sortOrder: tierFields.length,
							})}
						>
							<Plus className="h-4 w-4 mr-1" /> Add Tier
						</Button>
					</div>

					{tierFields.length === 0 ? (
						<div className="text-center py-8 text-muted-foreground">
							No ticket tiers added yet. Click "Add Tier" to create your first tier.
						</div>
					) : (
						<div className="space-y-4">
							{tierFields.map((field, index) => (
								<Card key={field.id} className="p-4">
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
															<Input {...field} placeholder="VIP Experience" />
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
															<Input {...field} placeholder="149.99" />
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
															placeholder="What's special about this tier..."
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
													onClick={() => addFeatureToTier(index)}
												>
													<Plus className="w-4 h-4" />
												</Button>
											</div>
											{form.watch(`priceTiers.${index}.features`)?.map((feature, featureIndex) => (
												<div key={featureIndex} className="flex gap-2">
													<Input
														value={feature}
														onChange={(e) => updateFeatureInTier(index, featureIndex, e.target.value)}
														placeholder="e.g., Meet & Greet, Premium Parking"
													/>
													<Button
														type="button"
														size="sm"
														variant="ghost"
														onClick={() => removeFeatureFromTier(index, featureIndex)}
													>
														<Trash2 className="w-4 h-4" />
													</Button>
												</div>
											))}
											{(!form.watch(`priceTiers.${index}.features`) || form.watch(`priceTiers.${index}.features`)?.length === 0) && (
												<div className="text-center py-2 text-sm text-muted-foreground">
													No features added. Click + to add features for this tier.
												</div>
											)}
										</div>
									</div>
								</Card>
							))}
						</div>
					)}
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
								<FormMessage />
							</FormItem>
						)}
					/>
				</div>
			)}
		</div>
	);
}
