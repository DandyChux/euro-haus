import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Textarea } from '~/components/ui/textarea';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '~/components/ui/card';
import { Checkbox } from '~/components/ui/checkbox';
import { Separator } from '~/components/ui/separator';
import { Badge } from '~/components/ui/badge';
import { Skeleton } from '~/components/ui/skeleton';
import { toast } from 'sonner';
import { apiClient } from '~/lib/api';
import {
	Copy,
	Loader2,
	LogOut,
	Plus,
	Pencil,
	Trash2,
	Search,
	Package,
	Calendar,
	RefreshCw
} from 'lucide-react';
import { ProductForm } from '~/components/product-form';
import { EventForm } from '~/components/event-form';
import { ProtectedRoute } from '~/components/protected-route';
import { useAuth } from '~/lib/contexts/auth-context';
import {
	Form,
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
import {
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
} from '~/components/ui/tabs';
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from '~/components/ui/alert-dialog';
import {
	formSchema,
	FormData
} from '~/lib/schemas/product-schema';

// Stripe Product interface
interface StripeProduct {
	id: string;
	name: string;
	description: string | null;
	images: string[];
	metadata: Record<string, string>;
	active: boolean;
	default_price: {
		id: string;
		unit_amount: number;
		currency: string;
	} | null;
}

export const Route = createFileRoute('/admin/products')({
	component: AdminProductsPage,
});

function AdminProductsPage() {
	return (
		<ProtectedRoute>
			<AdminProductsContent />
		</ProtectedRoute>
	);
}

function AdminProductsContent() {
	const navigate = useNavigate();
	const { logout } = useAuth();
	const [activeTab, setActiveTab] = useState('manage');
	const [products, setProducts] = useState<StripeProduct[]>([]);
	const [filteredProducts, setFilteredProducts] = useState<StripeProduct[]>([]);
	const [isLoading, setIsLoading] = useState(false);
	const [isRefreshing, setIsRefreshing] = useState(false);
	const [searchQuery, setSearchQuery] = useState('');
	const [filterType, setFilterType] = useState<'all' | 'product' | 'event'>('all');
	const [editingProduct, setEditingProduct] = useState<StripeProduct | null>(null);
	const [productType, setProductType] = useState<'product' | 'event'>('product');
	const [productToDelete, setProductToDelete] = useState<{ id: string; name: string } | null>(null);
	const [isDeleting, setIsDeleting] = useState(false);

	const form = useForm<FormData>({
		resolver: zodResolver(formSchema),
		defaultValues: {
			type: 'product',
			name: '',
			description: '',
			price: '',
			imageUrl: '',
			featured: false,
			maxQuantity: '10',
			category: 'merchandise',
			inStock: true,
			isNew: false,
			compareAtPrice: '',
		} as FormData,
	});

	// Fetch products
	const fetchProducts = async () => {
		setIsRefreshing(true);
		try {
			const response = await apiClient.get('/products');
			const allProducts = response.data.products || [];
			setProducts(allProducts);
			filterProducts(allProducts, searchQuery, filterType);
		} catch (error) {
			console.error('Error fetching products:', error);
			toast.error('Failed to fetch products');
		} finally {
			setIsRefreshing(false);
		}
	};

	// Filter products based on search and type
	const filterProducts = (productList: StripeProduct[], query: string, type: string) => {
		let filtered = productList;

		// Filter by type
		if (type !== 'all') {
			filtered = filtered.filter(p => p.metadata.type === type);
		}

		// Filter by search query
		if (query) {
			filtered = filtered.filter(p =>
				p.name.toLowerCase().includes(query.toLowerCase()) ||
				(p.description && p.description.toLowerCase().includes(query.toLowerCase()))
			);
		}

		setFilteredProducts(filtered);
	};

	// Load products on mount
	useEffect(() => {
		fetchProducts();
	}, []);

	// Update filters
	useEffect(() => {
		filterProducts(products, searchQuery, filterType);
	}, [searchQuery, filterType, products]);

	// Generate slug from name
	const generateSlug = () => {
		const name = form.getValues('name');
		const date = form.getValues('eventDate');
		if (!name || !date) return;

		const dateObj = new Date(date);
		const monthYear = dateObj.toLocaleDateString('en-US', { month: 'long', year: 'numeric' }).toLowerCase();
		const slug = `${name.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-${monthYear.replace(' ', '-')}`;
		form.setValue('slug', slug);
	};

	// Load product for editing
	const loadProductForEdit = (product: StripeProduct) => {
		setEditingProduct(product);
		const isEvent = product.metadata.type === 'event';
		setProductType(isEvent ? 'event' : 'product');

		// Parse price from cents
		const price = product.default_price
			? (product.default_price.unit_amount / 100).toFixed(2)
			: '0.00';

		// Base form data
		const formData: any = {
			type: product.metadata.type || 'product',
			name: product.name,
			description: product.description || '',
			price,
			imageUrl: product.images[0] || '',
			featured: product.metadata.featured === 'true',
			maxQuantity: product.metadata.max_quantity || '10',
		};

		if (isEvent) {
			// Parse event-specific data
			const eventDate = product.metadata.event_date ? new Date(product.metadata.event_date) : new Date();
			formData.slug = product.metadata.slug || '';
			formData.eventDate = eventDate.toISOString().split('T')[0];
			formData.eventTime = eventDate.toTimeString().slice(0, 5);
			formData.location = product.metadata.location || '';
			formData.capacity = product.metadata.capacity || '100';
			formData.availableSpots = product.metadata.available_spots || '100';
			formData.organizer = product.metadata.organizer || '';
			formData.status = product.metadata.status || 'upcoming';

			// Parse arrays
			try {
				formData.tags = JSON.parse(product.metadata.tags || '[]').map((t: string) => ({ value: t }));
				formData.agenda = JSON.parse(product.metadata.agenda || '[]');
				formData.includes = JSON.parse(product.metadata.includes || '[]').map((i: string) => ({ value: i }));
			} catch {
				formData.tags = [{ value: '' }];
				formData.agenda = [{ time: '9:00 AM', activity: '' }];
				formData.includes = [{ value: '' }];
			}
		} else {
			// Product-specific data
			formData.category = product.metadata.category || 'merchandise';
			formData.inStock = product.metadata.in_stock !== 'false';
			formData.isNew = product.metadata.is_new === 'true';
			formData.compareAtPrice = product.metadata.compare_at_price || '';
		}

		form.reset(formData);
		setActiveTab('create');
	};

	// Submit form (create or update)
	const onSubmit = async (data: FormData) => {
		setIsLoading(true);

		try {
			// Prepare metadata
			const metadata: Record<string, string> = {
				type: data.type,
				featured: data.featured.toString(),
				max_quantity: data.maxQuantity,
			};

			if (data.type === 'product') {
				metadata.category = data.category;
				metadata.in_stock = data.inStock.toString();
				metadata.is_new = data.isNew.toString();
				if (data.compareAtPrice) {
					metadata.compare_at_price = data.compareAtPrice;
				}
			} else {
				// Event metadata
				const eventDateTime = `${data.eventDate}T${data.eventTime}:00Z`;
				metadata.slug = data.slug;
				metadata.event_date = eventDateTime;
				metadata.location = data.location;
				metadata.capacity = data.capacity;
				metadata.available_spots = data.availableSpots;
				metadata.organizer = data.organizer;
				metadata.status = data.status;
				metadata.tags = JSON.stringify(data.tags.map(t => t.value).filter(Boolean));
				metadata.agenda = JSON.stringify(data.agenda);
				metadata.includes = JSON.stringify(data.includes.map(i => i.value).filter(Boolean));
			}

			const requestData = {
				name: data.name,
				description: data.description,
				price: Math.round(parseFloat(data.price) * 100), // Convert to cents
				currency: 'usd',
				images: data.imageUrl ? [data.imageUrl] : [],
				metadata,
			};

			if (editingProduct) {
				// Update existing product
				await apiClient.put(`/admin/update-product/${editingProduct.id}`, requestData);
				toast.success(`${data.type === 'event' ? 'Event' : 'Product'} updated successfully!`);
			} else {
				// Create new product
				await apiClient.post('/admin/create-product', requestData);
				toast.success(`${data.type === 'event' ? 'Event' : 'Product'} created successfully!`);
			}

			// Reset form and refresh list
			form.reset();
			setEditingProduct(null);
			fetchProducts();
			setActiveTab('manage');
		} catch (error: unknown) {
			console.error('Error saving product:', error);
			const errorMessage = error && typeof error === 'object' && 'response' in error &&
				error.response && typeof error.response === 'object' && 'data' in error.response
				? String(error.response.data)
				: 'Failed to save product';
			toast.error(errorMessage);
		} finally {
			setIsLoading(false);
		}
	};

	// Handle delete confirmation
	const handleDeleteProduct = async () => {
		if (!productToDelete) return;

		setIsDeleting(true);
		try {
			await apiClient.delete(`/admin/delete-product/${productToDelete.id}`);
			toast.success('Product deleted successfully');
			fetchProducts();
			setProductToDelete(null);
		} catch (error) {
			console.error('Error deleting product:', error);
			toast.error('Failed to delete product');
		} finally {
			setIsDeleting(false);
		}
	};

	// Load templates
	const loadEventTemplate = () => {
		setEditingProduct(null);
		setProductType('event');
		form.reset({
			type: 'event',
			name: 'Porsche Club Meetup - Month Year',
			description: 'Join fellow Porsche enthusiasts for an exclusive gathering featuring rare models, technical talks, and driving experiences.',
			price: '149.99',
			imageUrl: 'https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/event-banner.jpg',
			featured: false,
			maxQuantity: '10',
			slug: '',
			eventDate: '',
			eventTime: '09:00',
			location: 'Euro Haus Headquarters, Orlando',
			capacity: '100',
			availableSpots: '100',
			organizer: 'Euro Haus Events Team',
			status: 'upcoming',
			tags: [
				{ value: 'Porsche' },
				{ value: 'Track Day' },
				{ value: 'Networking' },
			],
			agenda: [
				{ time: '9:00 AM', activity: 'Registration & Welcome Coffee' },
				{ time: '10:00 AM', activity: 'Technical Talk' },
				{ time: '11:30 AM', activity: 'Track Experience Sessions' },
				{ time: '1:00 PM', activity: 'Lunch & Networking' },
				{ time: '3:00 PM', activity: 'Awards Ceremony' },
			],
			includes: [
				{ value: 'Track time with professional instructors' },
				{ value: 'Lunch and refreshments' },
				{ value: 'Event merchandise' },
				{ value: 'Professional photography' },
				{ value: 'Certificate of participation' },
			],
		});
		toast.success('Event template loaded');
		setActiveTab('create');
	};

	const loadProductTemplate = () => {
		setEditingProduct(null);
		setProductType('product');
		form.reset({
			type: 'product',
			name: 'Euro Haus T-Shirt',
			description: 'Premium cotton t-shirt with embroidered Euro Haus logo',
			price: '29.99',
			imageUrl: 'https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/product.jpg',
			featured: false,
			maxQuantity: '10',
			category: 'apparel',
			inStock: true,
			isNew: false,
			compareAtPrice: '39.99',
		});
		toast.success('Product template loaded');
		setActiveTab('create');
	};

	const handleLogout = async () => {
		await logout();
		navigate({ to: '/admin/login' });
	};

	// Format price for display
	const formatPrice = (cents: number) => {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD',
		}).format(cents / 100);
	};

	return (
		<div className="min-h-screen bg-background">
			<div className="max-w-7xl mx-auto p-6">
				<div className="flex justify-between items-center mb-8">
					<div>
						<h1 className="text-3xl font-bold">Product Management</h1>
						<p className="text-muted-foreground">Manage your Stripe products and events</p>
					</div>
					<div className="flex gap-2">
						<Button variant="outline" onClick={() => navigate({ to: '/' })}>
							Back to Site
						</Button>
						<Button variant="ghost" size="icon" onClick={handleLogout} title="Logout">
							<LogOut className="h-4 w-4" />
						</Button>
					</div>
				</div>

				<Tabs value={activeTab} onValueChange={setActiveTab}>
					<TabsList className="mb-8">
						<TabsTrigger value="manage">
							<Package className="h-4 w-4 mr-2" />
							Manage Products
						</TabsTrigger>
						<TabsTrigger value="create">
							<Plus className="h-4 w-4 mr-2" />
							{editingProduct ? 'Edit' : 'Create'} Product
						</TabsTrigger>
					</TabsList>

					{/* Manage Products Tab */}
					<TabsContent value="manage">
						<Card>
							<CardHeader>
								<div className="flex justify-between items-start">
									<div>
										<CardTitle>All Products</CardTitle>
										<CardDescription>View and manage your products and events</CardDescription>
									</div>
									<Button
										variant="outline"
										size="sm"
										onClick={fetchProducts}
										disabled={isRefreshing}
									>
										{isRefreshing ? (
											<Loader2 className="h-4 w-4 animate-spin" />
										) : (
											<RefreshCw className="h-4 w-4" />
										)}
									</Button>
								</div>
							</CardHeader>
							<CardContent>
								{/* Filters */}
								<div className="flex gap-4 mb-6">
									<div className="flex-1">
										<div className="relative">
											<Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
											<Input
												placeholder="Search products..."
												value={searchQuery}
												onChange={(e) => setSearchQuery(e.target.value)}
												className="pl-10"
											/>
										</div>
									</div>
									<Select value={filterType} onValueChange={(value: any) => setFilterType(value)}>
										<SelectTrigger className="w-[180px]">
											<SelectValue placeholder="Filter by type" />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="all">All Products</SelectItem>
											<SelectItem value="product">Products Only</SelectItem>
											<SelectItem value="event">Events Only</SelectItem>
										</SelectContent>
									</Select>
								</div>

								{/* Product List */}
								{isRefreshing ? (
									<div className="space-y-4">
										{[...Array(3)].map((_, i) => (
											<Skeleton key={i} className="h-24 w-full" />
										))}
									</div>
								) : filteredProducts.length === 0 ? (
									<div className="text-center py-12">
										<p className="text-muted-foreground mb-4">No products found</p>
										<div className="flex gap-2 justify-center">
											<Button onClick={loadProductTemplate} variant="outline">
												<Plus className="h-4 w-4 mr-2" />
												Create Product
											</Button>
											<Button onClick={loadEventTemplate} variant="outline">
												<Calendar className="h-4 w-4 mr-2" />
												Create Event
											</Button>
										</div>
									</div>
								) : (
									<div className="space-y-4">
										{filteredProducts.map((product) => (
											<div
												key={product.id}
												className="border rounded-lg p-4 hover:shadow-md transition-shadow"
											>
												<div className="flex items-start gap-4">
													{product.images[0] && (
														<img
															src={product.images[0]}
															alt={product.name}
															className="w-20 h-20 object-cover rounded"
														/>
													)}
													<div className="flex-1">
														<div className="flex justify-between items-start">
															<div>
																<h3 className="font-semibold">{product.name}</h3>
																<p className="text-sm text-muted-foreground line-clamp-2">
																	{product.description}
																</p>
																<div className="flex gap-2 mt-2">
																	<Badge variant={product.metadata.type === 'event' ? 'default' : 'secondary'}>
																		{product.metadata.type === 'event' ? (
																			<Calendar className="h-3 w-3 mr-1" />
																		) : (
																			<Package className="h-3 w-3 mr-1" />
																		)}
																		{product.metadata.type || 'product'}
																	</Badge>
																	{product.metadata.featured === 'true' && (
																		<Badge variant="outline">Featured</Badge>
																	)}
																	{product.metadata.in_stock === 'false' && (
																		<Badge variant="destructive">Out of Stock</Badge>
																	)}
																</div>
															</div>
															<div className="text-right">
																<p className="font-semibold">
																	{product.default_price ? formatPrice(product.default_price.unit_amount) : 'No price'}
																</p>
																<div className="flex gap-2 mt-2">
																	<Button
																		size="sm"
																		variant="outline"
																		onClick={() => loadProductForEdit(product)}
																	>
																		<Pencil className="h-4 w-4" />
																	</Button>
																	<Button
																		size="sm"
																		variant="outline"
																		onClick={() => setProductToDelete({ id: product.id, name: product.name })}
																	>
																		<Trash2 className="h-4 w-4" />
																	</Button>
																</div>
															</div>
														</div>
													</div>
												</div>
											</div>
										))}
									</div>
								)}
							</CardContent>
						</Card>
					</TabsContent>

					{/* Create/Edit Product Tab */}
					<TabsContent value="create">
						{/* Quick Templates */}
						{!editingProduct && (
							<div className="grid md:grid-cols-2 gap-4 mb-8">
								<Card
									className="cursor-pointer hover:shadow-md transition-shadow"
									onClick={loadProductTemplate}
								>
									<CardHeader>
										<div className="flex items-center justify-between">
											<CardTitle className="text-lg">Product Template</CardTitle>
											<Copy className="h-4 w-4" />
										</div>
										<CardDescription>Load merchandise template</CardDescription>
									</CardHeader>
								</Card>
								<Card
									className="cursor-pointer hover:shadow-md transition-shadow"
									onClick={loadEventTemplate}
								>
									<CardHeader>
										<div className="flex items-center justify-between">
											<CardTitle className="text-lg">Event Template</CardTitle>
											<Copy className="h-4 w-4" />
										</div>
										<CardDescription>Load event ticket template</CardDescription>
									</CardHeader>
								</Card>
							</div>
						)}

						<Form {...form}>
							<form onSubmit={form.handleSubmit(onSubmit)}>
								<Card>
									<CardHeader>
										<CardTitle>
											{editingProduct ? `Edit ${editingProduct.metadata.type === 'event' ? 'Event' : 'Product'}` : 'Product Information'}
										</CardTitle>
										{editingProduct && (
											<CardDescription>
												Editing: {editingProduct.name}
											</CardDescription>
										)}
									</CardHeader>
									<CardContent className="space-y-6">
										{/* Product Type */}
										<FormField
											control={form.control}
											name="type"
											render={({ field }) => (
												<FormItem>
													<FormLabel>Product Type</FormLabel>
													<Select
														value={productType}
														onValueChange={(value: 'product' | 'event') => {
															setProductType(value);
															field.onChange(value);
															// Reset form with appropriate defaults
															if (value === 'event') {
																form.setValue('slug', '');
																form.setValue('eventDate', '');
																form.setValue('eventTime', '09:00');
																form.setValue('location', 'Euro Haus Headquarters, Orlando');
																form.setValue('capacity', '100');
																form.setValue('availableSpots', '100');
																form.setValue('organizer', 'Euro Haus Events Team');
																form.setValue('status', 'upcoming');
																form.setValue('tags', [{ value: '' }]);
																form.setValue('agenda', [{ time: '9:00 AM', activity: '' }]);
																form.setValue('includes', [{ value: '' }]);
															}
														}}
														disabled={!!editingProduct}
													>
														<FormControl>
															<SelectTrigger>
																<SelectValue />
															</SelectTrigger>
														</FormControl>
														<SelectContent>
															<SelectItem value="product">Regular Product</SelectItem>
															<SelectItem value="event">Event Ticket</SelectItem>
														</SelectContent>
													</Select>
													<FormMessage />
												</FormItem>
											)}
										/>

										{/* Basic Info */}
										<div className="grid md:grid-cols-2 gap-4">
											<FormField
												control={form.control}
												name="name"
												render={({ field }) => (
													<FormItem>
														<FormLabel>
															{productType === 'event' ? 'Event Name' : 'Product Name'}
														</FormLabel>
														<FormControl>
															<Input {...field} placeholder={productType === 'event' ? 'Porsche Club Meetup - June 2025' : 'Euro Haus T-Shirt'} />
														</FormControl>
														<FormMessage />
													</FormItem>
												)}
											/>
											<FormField
												control={form.control}
												name="price"
												render={({ field }) => (
													<FormItem>
														<FormLabel>Price (USD)</FormLabel>
														<FormControl>
															<Input {...field} placeholder="29.99" />
														</FormControl>
														<FormMessage />
													</FormItem>
												)}
											/>
										</div>

										<FormField
											control={form.control}
											name="description"
											render={({ field }) => (
												<FormItem>
													<FormLabel>Description</FormLabel>
													<FormControl>
														<Textarea {...field} rows={3} placeholder="Brief description of the product or event" />
													</FormControl>
													<FormMessage />
												</FormItem>
											)}
										/>

										<FormField
											control={form.control}
											name="imageUrl"
											render={({ field }) => (
												<FormItem>
													<FormLabel>Image URL</FormLabel>
													<FormControl>
														<Input {...field} type="url" placeholder="https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/..." />
													</FormControl>
													<FormDescription>Leave empty for placeholder image</FormDescription>
													<FormMessage />
												</FormItem>
											)}
										/>

										<Separator />

										{/* Type-specific forms */}
										{productType === 'product' ? (
											<ProductForm form={form} />
										) : (
											<EventForm form={form} onGenerateSlug={generateSlug} />
										)}

										<Separator />

										{/* Common metadata */}
										<div className="grid md:grid-cols-2 gap-4">
											<FormField
												control={form.control}
												name="maxQuantity"
												render={({ field }) => (
													<FormItem>
														<FormLabel>Max Quantity per Order</FormLabel>
														<FormControl>
															<Input {...field} type="number" />
														</FormControl>
														<FormMessage />
													</FormItem>
												)}
											/>
											<FormField
												control={form.control}
												name="featured"
												render={({ field }) => (
													<FormItem className="flex items-center space-x-2 mt-8">
														<FormControl>
															<Checkbox checked={field.value} onCheckedChange={field.onChange} />
														</FormControl>
														<FormLabel className="font-normal cursor-pointer">Featured on Homepage</FormLabel>
													</FormItem>
												)}
											/>
										</div>
									</CardContent>
									<CardFooter className="flex gap-2">
										{editingProduct && (
											<Button
												type="button"
												variant="outline"
												onClick={() => {
													setEditingProduct(null);
													form.reset();
												}}
											>
												Cancel Edit
											</Button>
										)}
										<Button
											type="submit"
											className="flex-1"
											size="lg"
											disabled={isLoading}
										>
											{isLoading ? (
												<>
													<Loader2 className="mr-2 h-4 w-4 animate-spin" />
													{editingProduct ? 'Updating...' : 'Creating...'}
												</>
											) : (
												`${editingProduct ? 'Update' : 'Create'} ${productType === 'event' ? 'Event' : 'Product'} in Stripe`
											)}
										</Button>
									</CardFooter>
								</Card>
							</form>
						</Form>
					</TabsContent>
				</Tabs>

				{/* Delete Confirmation Dialog */}
				<AlertDialog open={!!productToDelete} onOpenChange={(open) => !open && setProductToDelete(null)}>
					<AlertDialogContent>
						<AlertDialogHeader>
							<AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
							<AlertDialogDescription>
								This will permanently archive the product "{productToDelete?.name}" in Stripe.
								This action cannot be undone and the product will no longer be available for purchase.
							</AlertDialogDescription>
						</AlertDialogHeader>
						<AlertDialogFooter>
							<AlertDialogCancel>Cancel</AlertDialogCancel>
							<AlertDialogAction
								onClick={handleDeleteProduct}
								disabled={isDeleting}
								className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
							>
								{isDeleting ? (
									<>
										<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										Deleting...
									</>
								) : (
									'Delete Product'
								)}
							</AlertDialogAction>
						</AlertDialogFooter>
					</AlertDialogContent>
				</AlertDialog>
			</div>
		</div>
	);
}
