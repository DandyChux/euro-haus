import React, { useState } from 'react';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Upload, X, Car, AlertCircle, Ticket, DollarSign, Plus, Trash2, Tag, CheckCircle } from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Textarea } from '~/components/ui/textarea';
import { Label } from '~/components/ui/label';
import { Separator } from '~/components/ui/separator';
import { Badge } from '~/components/ui/badge';
import {
	Form,
	FormControl,
	FormDescription,
	FormField,
	FormItem,
	FormLabel,
	FormMessage,
} from '~/components/ui/form';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { submissionService } from '~/lib/services/submission-service';
import type { SubmissionWithFiles } from '~/lib/services/submission-service';
import { Image } from './ui/image';
import { apiClient } from '~/lib/api';
import { toast } from 'sonner';

const formSchema = z.object({
	participantName: z.string().min(2, 'Name must be at least 2 characters'),
	participantEmail: z.string().email('Invalid email address'),
	participantPhone: z.string().optional(),
	vehicleYear: z.string().regex(/^\d{4}$/, 'Year must be 4 digits'),
	vehicleMake: z.string().min(2, 'Make is required'),
	vehicleModel: z.string().min(2, 'Model is required'),
	vehicleDescription: z.string().optional(),
	vehicleModifications: z.array(z.object({ value: z.string() })).optional(),
});

type FormData = z.infer<typeof formSchema>;

interface DiscountInfo {
	valid: boolean;
	code: string;
	couponName?: string;
	percentOff?: number | null;
	amountOff?: number | null;
	currency?: string;
}

interface VehicleSubmissionFormProps {
	eventId: string;
	eventSlug: string;
	eventName: string;
	ticketTier?: string;
	ticketPrice?: number;
	ticketQuantity?: number;
	onSuccess: (submissionId: string, discountCode?: string) => void;
	onCancel: () => void;
}

export function VehicleSubmissionForm({
	eventId,
	eventSlug,
	eventName,
	ticketTier,
	ticketPrice,
	ticketQuantity,
	onSuccess,
	onCancel,
}: VehicleSubmissionFormProps) {
	const [images, setImages] = useState<File[]>([]);
	const [imagePreviews, setImagePreviews] = useState<string[]>([]);
	const [uploading, setUploading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [discountCode, setDiscountCode] = useState('');
	const [validatedDiscount, setValidatedDiscount] = useState<DiscountInfo | null>(null);
	const [isValidating, setIsValidating] = useState(false);
	const [discountError, setDiscountError] = useState<string | null>(null);

	const form = useForm<FormData>({
		resolver: zodResolver(formSchema),
		defaultValues: {
			participantName: '',
			participantEmail: '',
			participantPhone: '',
			vehicleYear: '',
			vehicleMake: '',
			vehicleModel: '',
			vehicleDescription: '',
			vehicleModifications: []
		},
	});

	const {
		fields: vehicleModifications,
		append: appendModification,
		remove: removeModification,
		update: updateModification
	} = useFieldArray({
		control: form.control,
		name: "vehicleModifications"
	})

	// Calculate discounted price
	const calculateDiscountedPrice = () => {
		if (!ticketPrice || !ticketQuantity) return 0;
		const subtotal = ticketPrice * ticketQuantity;

		if (!validatedDiscount) return subtotal;

		if (validatedDiscount.percentOff && validatedDiscount.percentOff > 0) {
			return subtotal * (1 - validatedDiscount.percentOff / 100);
		} else if (validatedDiscount.amountOff && validatedDiscount.amountOff > 0) {
			// Amount off is in cents, so divide by 100
			return Math.max(0, subtotal - (validatedDiscount.amountOff / 100));
		}

		return subtotal;
	};

	const getDiscountAmount = () => {
		if (!ticketPrice || !ticketQuantity || !validatedDiscount) return 0;
		const subtotal = ticketPrice * ticketQuantity;

		if (validatedDiscount.percentOff && validatedDiscount.percentOff > 0) {
			return subtotal * (validatedDiscount.percentOff / 100);
		} else if (validatedDiscount.amountOff && validatedDiscount.amountOff > 0) {
			// Amount off is in cents, so divide by 100
			return Math.min(subtotal, validatedDiscount.amountOff / 100);
		}

		return 0;
	};

	const getDiscountDisplay = () => {
		if (!validatedDiscount) return '';

		if (validatedDiscount.percentOff && validatedDiscount.percentOff > 0) {
			return `${validatedDiscount.percentOff}% OFF`;
		} else if (validatedDiscount.amountOff && validatedDiscount.amountOff > 0) {
			return `$${(validatedDiscount.amountOff / 100).toFixed(2)} OFF`;
		}

		return 'DISCOUNT APPLIED';
	};

	const handleValidateDiscount = async () => {
		if (!discountCode.trim()) return;

		setIsValidating(true);
		setDiscountError(null);

		try {
			// Call the existing /validate-promotion-code endpoint
			const response = await apiClient.post('/validate-promotion-code', {
				code: discountCode.toUpperCase()
			});

			if (response.data.valid) {
				const discountData: DiscountInfo = {
					valid: true,
					code: response.data.promotion_code.code,
					couponName: response.data.promotion_code.coupon_name,
					percentOff: response.data.discount.percent_off,
					amountOff: response.data.discount.amount_off,
					currency: response.data.discount.currency
				};

				setValidatedDiscount(discountData);
				toast.success(`Discount "${discountCode}" applied successfully!`);
			} else {
				setDiscountError(response.data.error || 'Invalid or expired discount code');
				setValidatedDiscount(null);
			}
		} catch (error) {
			console.error('Failed to validate discount:', error);
			setDiscountError('Invalid or expired discount code');
			setValidatedDiscount(null);
		} finally {
			setIsValidating(false);
		}
	};

	const handleRemoveDiscount = () => {
		setDiscountCode('');
		setValidatedDiscount(null);
		setDiscountError(null);
	};

	const handleDiscountCodeChange = (value: string) => {
		setDiscountCode(value.toUpperCase());
		// Clear validated discount if code changes
		if (validatedDiscount && value.toUpperCase() !== validatedDiscount.code) {
			setValidatedDiscount(null);
			setDiscountError(null);
		}
	};

	const handleImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const files = Array.from(e.target.files || []);

		// Validate images
		const validation = submissionService.validateImages(files);
		if (!validation.valid) {
			setError(validation.error ?? 'Invalid image format or size');
			return;
		}

		// Limit to 5 images
		if (images.length + files.length > 5) {
			setError('Maximum 5 images allowed');
			return;
		}

		setError(null);

		// Add new images
		const newImages = [...images, ...files];
		setImages(newImages);

		// Create previews
		files.forEach((file) => {
			const reader = new FileReader();
			reader.onloadend = () => {
				setImagePreviews((prev) => [...prev, reader.result as string]);
			};
			reader.readAsDataURL(file);
		});
	};

	const removeImage = (index: number) => {
		setImages((prev) => prev.filter((_, i) => i !== index));
		setImagePreviews((prev) => prev.filter((_, i) => i !== index));
	};

	const onSubmit = async (data: FormData) => {
		if (images.length === 0) {
			setError('Please upload at least one image of your vehicle');
			return;
		}

		setUploading(true);
		setError(null);

		// Transform the modifications from an object array to string array and filter empty values
		const vehicleModifications = data.vehicleModifications
			?.filter(mod => mod.value.trim() !== '')
			.map(mod => mod.value.trim());

		try {
			const submissionData: SubmissionWithFiles = {
				...data,
				eventId,
				eventSlug,
				images,
				ticketTier,
				ticketPrice,
				ticketQuantity,
				vehicleModifications
			};

			const submission = await submissionService.createSubmission(submissionData);
			// Pass the validated discount code, not just any typed code
			onSuccess(submission.id, validatedDiscount?.code);
		} catch (err) {
			setError(err instanceof Error ? err.message : 'Failed to submit vehicle information');
		} finally {
			setUploading(false);
		}
	};

	return (
		<div className="space-y-6">
			<div className="text-center">
				<Car className="mx-auto h-12 w-12 text-primary mb-4" />
				<h2 className="text-2xl font-bold">Vehicle Submission for {eventName}</h2>
				<p className="text-muted-foreground mt-2">
					Please provide details about the vehicle you wish to enter in this event.
				</p>
			</div>

			{/* Ticket Information */}
			{ticketTier && (
				<Card className="bg-primary/5 border-primary/20">
					<CardHeader className="pb-3">
						<CardTitle className="text-lg flex items-center gap-2">
							<Ticket className="w-5 h-5" />
							Selected Ticket
						</CardTitle>
					</CardHeader>
					<CardContent className="space-y-4">
						{/* Price Breakdown */}
						<div className="space-y-3">
							<div className="flex flex-wrap gap-4 items-center">
								<div>
									<p className="text-sm text-muted-foreground">Tier</p>
									<p className="font-semibold">{ticketTier}</p>
								</div>
								{ticketPrice && (
									<div>
										<p className="text-sm text-muted-foreground">Price per ticket</p>
										<p className="font-semibold flex items-center">
											<DollarSign className="w-4 h-4" />
											{ticketPrice.toFixed(2)}
										</p>
									</div>
								)}
								{ticketQuantity && (
									<div>
										<p className="text-sm text-muted-foreground">Quantity</p>
										<p className="font-semibold">{ticketQuantity} ticket{ticketQuantity > 1 ? 's' : ''}</p>
									</div>
								)}
							</div>

							{/* Price Summary with Discount */}
							{ticketPrice && ticketQuantity && (
								<div className="p-3 bg-background rounded-lg space-y-2">
									<div className="flex justify-between text-sm">
										<span className="text-muted-foreground">Subtotal</span>
										<span>${(ticketPrice * ticketQuantity).toFixed(2)}</span>
									</div>

									{validatedDiscount && getDiscountAmount() > 0 && (
										<div className="flex justify-between text-sm text-green-600">
											<span>Discount ({validatedDiscount.couponName || validatedDiscount.code})</span>
											<span>
												-${getDiscountAmount().toFixed(2)}
											</span>
										</div>
									)}

									<Separator />

									<div className="flex justify-between font-semibold text-lg">
										<span>Total</span>
										<span className="flex items-center">
											<DollarSign className="w-4 h-4" />
											{calculateDiscountedPrice().toFixed(2)}
										</span>
									</div>
								</div>
							)}
						</div>

						<Separator />

						{/* Discount Code */}
						<div className="space-y-2">
							<Label htmlFor="discount-code" className="text-sm font-medium flex items-center gap-2">
								<Tag className="w-4 h-4" />
								Promotion Code (Optional)
							</Label>

							{!validatedDiscount ? (
								<>
									<div className="flex gap-2">
										<Input
											id="discount-code"
											placeholder="Enter promotion code"
											value={discountCode}
											onChange={(e) => handleDiscountCodeChange(e.target.value)}
											className="flex-1"
											disabled={uploading || isValidating}
											onKeyDown={(e) => {
												if (e.key === 'Enter') {
													e.preventDefault();
													handleValidateDiscount();
												}
											}}
										/>
										<Button
											variant="outline"
											size="default"
											type="button"
											disabled={!discountCode.trim() || isValidating}
											onClick={handleValidateDiscount}
										>
											{isValidating ? 'Validating...' : 'Apply'}
										</Button>
									</div>

									{discountError && (
										<Alert variant="destructive" className="py-2">
											<AlertCircle className="h-4 w-4" />
											<AlertDescription>{discountError}</AlertDescription>
										</Alert>
									)}
								</>
							) : (
								<div className="flex items-center justify-between p-3 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800">
									<div className="flex items-center gap-2">
										<CheckCircle className="h-4 w-4 text-green-600" />
										<span className="font-medium text-green-700 dark:text-green-400">
											{validatedDiscount.code}
										</span>
										{validatedDiscount.couponName && (
											<span className="text-sm text-muted-foreground">
												({validatedDiscount.couponName})
											</span>
										)}
										<Badge variant="secondary" className="text-xs">
											{getDiscountDisplay()}
										</Badge>
									</div>
									<Button
										variant="ghost"
										size="sm"
										onClick={handleRemoveDiscount}
										className="text-muted-foreground hover:text-destructive"
									>
										<X className="h-4 w-4" />
									</Button>
								</div>
							)}

							{!validatedDiscount && (
								<p className="text-xs text-muted-foreground">
									Enter a promotion code if you have one. The discount will be applied at checkout.
								</p>
							)}
						</div>
					</CardContent>
				</Card>
			)}

			{error && (
				<Alert variant="destructive">
					<AlertCircle className="h-4 w-4" />
					<AlertDescription>{error}</AlertDescription>
				</Alert>
			)}

			<Form {...form}>
				<form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
					{/* Personal Information */}
					<div className="space-y-4">
						<h3 className="text-lg font-semibold">Your Information</h3>

						<FormField
							control={form.control}
							name="participantName"
							render={({ field }) => (
								<FormItem>
									<FormLabel>Full Name</FormLabel>
									<FormControl>
										<Input {...field} placeholder="John Doe" />
									</FormControl>
									<FormMessage />
								</FormItem>
							)}
						/>

						<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
							<FormField
								control={form.control}
								name="participantEmail"
								render={({ field }) => (
									<FormItem>
										<FormLabel>Email</FormLabel>
										<FormControl>
											<Input {...field} type="email" placeholder="john@example.com" />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>

							<FormField
								control={form.control}
								name="participantPhone"
								render={({ field }) => (
									<FormItem>
										<FormLabel>Phone (Optional)</FormLabel>
										<FormControl>
											<Input {...field} type="tel" placeholder="+1 (555) 123-4567" />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>
						</div>
					</div>

					{/* Vehicle Information */}
					<div className="space-y-4">
						<h3 className="text-lg font-semibold">Vehicle Information</h3>

						<div className="grid grid-cols-1 md:grid-cols-3 gap-4">
							<FormField
								control={form.control}
								name="vehicleYear"
								render={({ field }) => (
									<FormItem>
										<FormLabel>Year</FormLabel>
										<FormControl>
											<Input {...field} placeholder="2023" maxLength={4} />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>

							<FormField
								control={form.control}
								name="vehicleMake"
								render={({ field }) => (
									<FormItem>
										<FormLabel>Make</FormLabel>
										<FormControl>
											<Input {...field} placeholder="Porsche" />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>

							<FormField
								control={form.control}
								name="vehicleModel"
								render={({ field }) => (
									<FormItem>
										<FormLabel>Model</FormLabel>
										<FormControl>
											<Input {...field} placeholder="911 GT3" />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>
						</div>

						<FormField
							control={form.control}
							name="vehicleDescription"
							render={({ field }) => (
								<FormItem>
									<FormLabel>Vehicle Description (Optional)</FormLabel>
									<FormControl>
										<Textarea
											{...field}
											placeholder="Tell us about your vehicle's history, special features, etc."
											rows={3}
										/>
									</FormControl>
									<FormDescription>
										Share what makes your vehicle special
									</FormDescription>
									<FormMessage />
								</FormItem>
							)}
						/>

						{/* Vehicle Modifications */}
						<div className='bg-muted/50 p-4 rounded-md'>
							<div className='flex justify-between items-center mb-3'>
								<h3 className='text-lg font-medium'>Vehicle Modifications</h3>
								<Button
									type='button'
									size='sm'
									onClick={() => appendModification({ value: '' })}
								>
									<Plus className="h-4 w-4 mr-1" /> Add Modification
								</Button>
							</div>

							{vehicleModifications.length === 0 ? (
								<div className="text-center py-4 text-muted-foreground">
									No modifications added yet. Add modifications to showcase your build.
								</div>
							) : (
								<div className="space-y-2">
									{vehicleModifications.map((field, index) => (
										<div key={field.id} className="flex items-end gap-2">
											<FormField
												control={form.control}
												name={`vehicleModifications.${index}.value`}
												render={({ field }) => (
													<FormItem className="flex-1">
														<FormControl>
															<Input {...field} placeholder="e.g., Turbocharged, Modified Suspension" />
														</FormControl>
														<FormMessage />
													</FormItem>
												)}
											/>
											<Button
												type="button"
												variant="ghost"
												size="icon"
												onClick={() => removeModification(index)}
											>
												<Trash2 className="h-4 w-4" />
											</Button>
										</div>
									))}
								</div>
							)}
						</div>
					</div>

					{/* Image Upload */}
					<div className="space-y-4">
						<h3 className="text-lg font-semibold">Vehicle Images</h3>
						<p className="text-sm text-muted-foreground">
							Upload up to 5 images of your vehicle. Include exterior, interior, and engine bay shots if possible.
						</p>

						<div className="space-y-4">
							<Label htmlFor="image-upload" className="cursor-pointer">
								<div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center hover:border-gray-400 transition-colors">
									<Upload className="mx-auto h-12 w-12 text-gray-400" />
									<p className="mt-2 text-sm text-gray-600">
										Click to upload images or drag and drop
									</p>
									<p className="text-xs text-gray-500">
										JPEG, PNG or WebP, max 10MB each
									</p>
								</div>
								<input
									id="image-upload"
									type="file"
									multiple
									accept="image/jpeg,image/png,image/webp"
									onChange={handleImageChange}
									className="hidden"
									disabled={uploading}
								/>
							</Label>

							{/* Image Previews */}
							{imagePreviews.length > 0 && (
								<div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
									{imagePreviews.map((preview, index) => (
										<div key={index} className="relative group">
											<Image
												src={preview}
												alt={`Vehicle ${index + 1}`}
												className="w-full h-24 object-cover rounded-lg"
											/>
											<button
												type="button"
												onClick={() => removeImage(index)}
												className="absolute top-1 right-1 bg-red-500 text-white rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity"
											>
												<X className="h-4 w-4" />
											</button>
										</div>
									))}
								</div>
							)}
						</div>
					</div>

					<div className="flex gap-4">
						<Button
							type="button"
							variant="outline"
							onClick={onCancel}
							disabled={uploading}
						>
							Cancel
						</Button>
						<Button type="submit" disabled={uploading || images.length === 0}>
							{uploading ? 'Submitting...' : 'Continue to Checkout'}
						</Button>
					</div>
				</form>
			</Form>
		</div>
	);
}
