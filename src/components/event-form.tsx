import { UseFormReturn, useFieldArray } from 'react-hook-form';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Plus, Trash2 } from 'lucide-react';
import {
	FormControl,
	FormField,
	FormItem,
	FormLabel,
	FormMessage,
} from '~/components/ui/form';
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from '~/components/ui/select';
import { FormData } from '~/lib/schemas/product-schema';

interface EventFormProps {
	form: UseFormReturn<FormData>;
	onGenerateSlug: () => void;
}

export function EventForm({ form, onGenerateSlug }: EventFormProps) {
	const tagsArray = useFieldArray({
		control: form.control,
		name: 'tags',
	});

	const agendaArray = useFieldArray({
		control: form.control,
		name: 'agenda',
	});

	const includesArray = useFieldArray({
		control: form.control,
		name: 'includes',
	});

	const sponsorsArray = useFieldArray({
		control: form.control,
		name: 'sponsors',
	});

	return (
		<>
			<div className="grid md:grid-cols-2 gap-4">
				<FormField
					control={form.control}
					name="eventDate"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Event Date</FormLabel>
							<FormControl>
								<Input {...field} type="date" onChange={(e) => {
									field.onChange(e);
									if (form.getValues('name')) onGenerateSlug();
								}} />
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
							<FormLabel>Event Time</FormLabel>
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
				name="slug"
				render={({ field }) => (
					<FormItem>
						<FormLabel>URL Slug</FormLabel>
						<div className="flex gap-2">
							<FormControl>
								<Input {...field} placeholder="porsche-club-june-2025" />
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
						<FormMessage />
					</FormItem>
				)}
			/>

			<FormField
				control={form.control}
				name="location"
				render={({ field }) => (
					<FormItem>
						<FormLabel>Location</FormLabel>
						<FormControl>
							<Input {...field} />
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
								<Input {...field} type="number" onChange={(e) => {
									field.onChange(e);
									form.setValue('availableSpots', e.target.value);
								}} />
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
								<Input {...field} type="number" />
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>
				<FormField
					control={form.control}
					name="status"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Status</FormLabel>
							<Select onValueChange={field.onChange} defaultValue={field.value}>
								<FormControl>
									<SelectTrigger>
										<SelectValue />
									</SelectTrigger>
								</FormControl>
								<SelectContent>
									<SelectItem value="upcoming">Upcoming</SelectItem>
									<SelectItem value="ongoing">Ongoing</SelectItem>
									<SelectItem value="completed">Completed</SelectItem>
									<SelectItem value="cancelled">Cancelled</SelectItem>
									<SelectItem value="soldout">Sold Out</SelectItem>
								</SelectContent>
							</Select>
							<FormMessage />
						</FormItem>
					)}
				/>
			</div>

			{/* Tags */}
			<div>
				<div className="flex justify-between items-center mb-2">
					<FormLabel>Tags</FormLabel>
					<Button
						type="button"
						size="sm"
						variant="outline"
						onClick={() => tagsArray.append({ value: '' })}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Tag
					</Button>
				</div>
				<div className="space-y-2">
					{tagsArray.fields.map((field, index) => (
						<FormField
							key={field.id}
							control={form.control}
							name={`tags.${index}.value`}
							render={({ field }) => (
								<FormItem>
									<div className="flex gap-2">
										<FormControl>
											<Input {...field} placeholder="e.g., Porsche, Track Day" />
										</FormControl>
										<Button
											type="button"
											size="icon"
											variant="outline"
											onClick={() => tagsArray.remove(index)}
											disabled={tagsArray.fields.length === 1}
										>
											<Trash2 className="h-4 w-4" />
										</Button>
									</div>
									<FormMessage />
								</FormItem>
							)}
						/>
					))}
				</div>
			</div>

			{/* Agenda */}
			<div>
				<div className="flex justify-between items-center mb-2">
					<FormLabel>Event Schedule</FormLabel>
					<Button
						type="button"
						size="sm"
						variant="outline"
						onClick={() => agendaArray.append({ time: '', activity: '' })}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Item
					</Button>
				</div>
				<div className="space-y-2">
					{agendaArray.fields.map((field, index) => (
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
											<Input {...field} placeholder="Registration & Welcome" />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>
							<Button
								type="button"
								size="icon"
								variant="outline"
								onClick={() => agendaArray.remove(index)}
								disabled={agendaArray.fields.length === 1}
							>
								<Trash2 className="h-4 w-4" />
							</Button>
						</div>
					))}
				</div>
			</div>

			{/* Includes */}
			<div>
				<div className="flex justify-between items-center mb-2">
					<FormLabel>What's Included</FormLabel>
					<Button
						type="button"
						size="sm"
						variant="outline"
						onClick={() => includesArray.append({ value: '' })}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Item
					</Button>
				</div>
				<div className="space-y-2">
					{includesArray.fields.map((field, index) => (
						<FormField
							key={field.id}
							control={form.control}
							name={`includes.${index}.value`}
							render={({ field }) => (
								<FormItem>
									<div className="flex gap-2">
										<FormControl>
											<Input {...field} placeholder="e.g., Lunch and refreshments" />
										</FormControl>
										<Button
											type="button"
											size="icon"
											variant="outline"
											onClick={() => includesArray.remove(index)}
											disabled={includesArray.fields.length === 1}
										>
											<Trash2 className="h-4 w-4" />
										</Button>
									</div>
									<FormMessage />
								</FormItem>
							)}
						/>
					))}
				</div>
			</div>

			{/* Sponsors */}
			<div>
				<div className="flex justify-between items-center mb-2">
					<FormLabel>Event Sponsors</FormLabel>
					<Button
						type="button"
						size="sm"
						variant="outline"
						onClick={() => sponsorsArray.append({ name: '', logoUrl: '', link: '' })}
					>
						<Plus className="h-4 w-4 mr-1" /> Add Sponsor
					</Button>
				</div>
				<div className="space-y-4">
					{sponsorsArray.fields.map((field, index) => (
						<div key={field.id} className="p-4 border rounded-lg space-y-3">
							<div className="flex justify-between items-start">
								<h4 className="text-sm font-medium">Sponsor {index + 1}</h4>
								<Button
									type="button"
									size="icon"
									variant="outline"
									onClick={() => sponsorsArray.remove(index)}
								>
									<Trash2 className="h-4 w-4" />
								</Button>
							</div>
							<FormField
								control={form.control}
								name={`sponsors.${index}.name`}
								render={({ field }) => (
									<FormItem>
										<FormLabel className="text-xs">Company Name</FormLabel>
										<FormControl>
											<Input {...field} placeholder="e.g., Porsche AG" />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>
							<FormField
								control={form.control}
								name={`sponsors.${index}.logoUrl`}
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
								name={`sponsors.${index}.link`}
								render={({ field }) => (
									<FormItem>
										<FormLabel className="text-xs">Website (optional)</FormLabel>
										<FormControl>
											<Input {...field} placeholder="https://example.com" />
										</FormControl>
										<FormMessage />
									</FormItem>
								)}
							/>
						</div>
					))}
				</div>
			</div>
		</>
	);
}
