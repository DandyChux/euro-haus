import React, { useState } from 'react';
import { useForm, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Upload, X, Car, AlertCircle, Ticket, DollarSign, Plus, Trash2 } from 'lucide-react';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Textarea } from '~/components/ui/textarea';
import { Label } from '~/components/ui/label';
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

interface VehicleSubmissionFormProps {
	eventId: string;
	eventSlug: string;
	eventName: string;
	ticketTier?: string;
	ticketPrice?: number;
	ticketQuantity?: number;
	onSuccess: (submissionId: string) => void;
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
			onSuccess(submission.id);
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
					<CardContent>
						<div className="flex flex-wrap gap-4 items-center">
							<div>
								<p className="text-sm text-muted-foreground">Tier</p>
								<p className="font-semibold">{ticketTier}</p>
							</div>
							{ticketPrice && (
								<div>
									<p className="text-sm text-muted-foreground">Price</p>
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
							{ticketPrice && ticketQuantity && (
								<div className="ml-auto">
									<p className="text-sm text-muted-foreground">Total</p>
									<p className="font-semibold flex items-center">
										<DollarSign className="w-4 h-4" />
										{(ticketPrice * ticketQuantity).toFixed(2)}
									</p>
								</div>
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
									No tags added yet. Add tags to help categorize your event.
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
							{uploading ? 'Submitting...' : 'Submit for Review'}
						</Button>
					</div>
				</form>
			</Form>
		</div>
	);
}
