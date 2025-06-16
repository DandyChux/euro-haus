import { UseFormReturn } from 'react-hook-form';
import { Input } from '~/components/ui/input';
import { Checkbox } from '~/components/ui/checkbox';
import {
	FormControl,
	FormDescription,
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

interface ProductFormProps {
	form: UseFormReturn<FormData>;
}

export function ProductForm({ form }: ProductFormProps) {
	return (
		<>
			<div className="grid md:grid-cols-2 gap-4">
				<FormField
					control={form.control}
					name="category"
					render={({ field }) => (
						<FormItem>
							<FormLabel>Category</FormLabel>
							<Select onValueChange={field.onChange} defaultValue={field.value}>
								<FormControl>
									<SelectTrigger>
										<SelectValue />
									</SelectTrigger>
								</FormControl>
								<SelectContent>
									<SelectItem value="merchandise">Merchandise</SelectItem>
									<SelectItem value="apparel">Apparel</SelectItem>
									<SelectItem value="accessories">Accessories</SelectItem>
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
							<FormLabel>Compare at Price (Optional)</FormLabel>
							<FormControl>
								<Input {...field} placeholder="39.99" />
							</FormControl>
							<FormDescription>Original price for sales</FormDescription>
							<FormMessage />
						</FormItem>
					)}
				/>
			</div>

			<div className="flex space-x-6">
				<FormField
					control={form.control}
					name="inStock"
					render={({ field }) => (
						<FormItem className="flex items-center space-x-2">
							<FormControl>
								<Checkbox checked={field.value} onCheckedChange={field.onChange} />
							</FormControl>
							<FormLabel className="font-normal cursor-pointer">In Stock</FormLabel>
						</FormItem>
					)}
				/>
				<FormField
					control={form.control}
					name="isNew"
					render={({ field }) => (
						<FormItem className="flex items-center space-x-2">
							<FormControl>
								<Checkbox checked={field.value} onCheckedChange={field.onChange} />
							</FormControl>
							<FormLabel className="font-normal cursor-pointer">New Product</FormLabel>
						</FormItem>
					)}
				/>
			</div>
		</>
	);
}
