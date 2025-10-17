import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState, useEffect, useCallback } from 'react';
import { useForm, UseFormReturn } from 'react-hook-form';
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
	RefreshCw,
	LayoutDashboard,
	Gift
} from 'lucide-react';
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
	FormData,
	ProductFormData,
	EventFormData,
	BundleFormData
} from '~/lib/schemas/product-schema';
import { ProductFormSection } from '~/components/admin/product-form-section';
import { EventFormSection } from '~/components/admin/event-form-section';
import { BundleFormSection } from '~/components/admin/bundle-form-section';
import { Image } from '~/components/ui/image';
import { fetchExternalMetadata } from '~/lib/utils';

// Stripe Product interface
interface StripeProduct {
	id: string;
	name: string;
	description: string | null;
	images: string[];
	metadata: Record<string, any>;
	default_price: {
		id: string;
		unit_amount: number;
		currency: string;
	} | null;
	prices?: Array<{
		id: string;
		unit_amount: number;
		currency: string;
		nickname: string | null;
		metadata: Record<string, string>;
	}>;
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
	const [isRefreshing, setIsRefreshing] = useState(false);
	const [searchQuery, setSearchQuery] = useState('');
	const [filterType, setFilterType] = useState<'all' | 'product' | 'bundle' | 'event'>('all');
	const [editingProduct, setEditingProduct] = useState<StripeProduct | null>(null);
	const [productType, setProductType] = useState<'product' | 'bundle' | 'event'>('product');
	const [productToDelete, setProductToDelete] = useState<{ id: string; name: string } | null>(null);
	const [isDeleting, setIsDeleting] = useState(false);

	const form = useForm<FormData>({
		resolver: zodResolver(formSchema),
		defaultValues: {
			type: 'product',
			name: '',
			description: '',
			imageUrl: '',
			featured: false,
			// Product specific with all required fields
			category: 'merchandise',
			hasVariants: false,
			price: '',
			compareAtPrice: '',
			inStock: true,
			isNew: false,
			maxQuantity: '10',
			variants: [],
			// Bundle specific fields will be set when switching to bundle type
			// Event specific fields will be set when switching to event type
		},
	});

	// Fetch products
	const fetchProducts = useCallback(async () => {
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
	}, [searchQuery, filterType]);

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
	}, [fetchProducts]);

	// Update filters
	useEffect(() => {
		filterProducts(products, searchQuery, filterType);
	}, [searchQuery, filterType, products]);

	// Generate slug from name
	const generateSlug = () => {
		const formType = form.getValues('type');
		if (formType !== 'event') return;

		const name = form.getValues('name');
		const eventData = form.getValues() as EventFormData;
		const date = eventData.eventDate;

		if (!name || !date) return;

		const dateObj = new Date(date);
		const monthYear = dateObj.toLocaleDateString('en-US', { month: 'long', year: 'numeric' }).toLowerCase();
		const slug = `${name.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-${monthYear.replace(' ', '-')}`;
		form.setValue('slug', slug);
	};

	// Load product for editing
	const loadProductForEdit = async (product: StripeProduct) => {
		setEditingProduct(product);
		const isEvent = product.metadata.type === 'event';
		setProductType(isEvent ? 'event' : 'product');

		form.clearErrors();

		// Parse price from cents
		const price = product.default_price
			? (product.default_price.unit_amount / 100).toFixed(2)
			: '0.00';

		if (isEvent) {
			// Fetch external metadata if needed
			let processedMetadata = { ...product.metadata };

			// Check if any fields are stored externally
			const hasExternalData = ['sponsor_tiers', 'sponsors', 'agenda', 'includes'].some(
				field => product.metadata[`${field}_external`] === 'true'
			);

			if (hasExternalData) {
				try {
					processedMetadata = await fetchExternalMetadata(product.metadata);
				} catch (error) {
					console.error('Error fetching external metadata:', error);
					toast.error('Warning: Some event data could not be loaded from external storage');
				}
			}

			// Event-specific data
			const eventDate = product.metadata.event_date ? new Date(product.metadata.event_date) : new Date();
			const eventFormData: EventFormData = {
				type: 'event',
				name: product.name,
				description: product.description || '',
				imageUrl: product.images[0] || '',
				featured: processedMetadata.featured === 'true',
				slug: processedMetadata.slug || '',
				eventDate: eventDate.toISOString().split('T')[0],
				eventTime: eventDate.toTimeString().slice(0, 5),
				location: processedMetadata.location || 'Euro Haus Headquarters, Tampa',
				capacity: processedMetadata.capacity || '100',
				availableSpots: processedMetadata.available_spots || processedMetadata.capacity || '100',
				organizer: processedMetadata.organizer || 'Euro Haus Events Team',
				status: (processedMetadata.status as EventFormData['status']) || 'upcoming',
				hasTiers: processedMetadata.has_tiers === 'true',
				price: price,
				maxQuantity: processedMetadata.max_quantity || '10',
				priceTiers: [],
				tags: [],
				agenda: [],
				includes: [],
				sponsors: [],
				sponsorTiers: [],
			};

			// Parse arrays from JSON strings
			try {
				// Tags
				const tagData = processedMetadata.tags;
				if (typeof tagData === 'string') {
					const tagStrings = JSON.parse(tagData || '[]');
					eventFormData.tags = Array.isArray(tagStrings)
						? tagStrings.map((tag: string) => ({ value: tag }))
						: [];
				} else if (Array.isArray(tagData)) {
					eventFormData.tags = tagData.map((tag: string) => ({ value: tag }));
				}

				// Agenda
				const agendaData = processedMetadata.agenda;
				if (typeof agendaData === 'string') {
					eventFormData.agenda = JSON.parse(agendaData || '[]');
				} else if (Array.isArray(agendaData)) {
					eventFormData.agenda = agendaData;
				}

				// Includes
				const includesData = processedMetadata.includes;
				if (typeof includesData === 'string') {
					const includesStrings = JSON.parse(includesData || '[]');
					eventFormData.includes = Array.isArray(includesStrings)
						? includesStrings.map((item: string) => ({ value: item }))
						: [];
				} else if (Array.isArray(includesData)) {
					eventFormData.includes = includesData.map((item: string) => ({ value: item }));
				}

				// Sponsor Tiers - handle both string and object formats
				const sponsorTiersData = processedMetadata.sponsor_tiers;
				if (sponsorTiersData) {
					let parsedTiers;
					if (typeof sponsorTiersData === 'string') {
						parsedTiers = JSON.parse(sponsorTiersData);
					} else {
						parsedTiers = sponsorTiersData;
					}

					if (Array.isArray(parsedTiers) && parsedTiers.length > 0) {
						eventFormData.sponsorTiers = parsedTiers.map((tier: any) => ({
							tierName: tier.tierName,
							displayOrder: tier.displayOrder ?? 0,
							sponsors: tier.sponsors || []
						}));
					}
				} else {
					// Legacy sponsors format
					const sponsorsData = processedMetadata.sponsors;
					if (sponsorsData) {
						let parsedSponsors;
						if (typeof sponsorsData === 'string') {
							parsedSponsors = JSON.parse(sponsorsData || '[]');
						} else {
							parsedSponsors = sponsorsData;
						}

						if (Array.isArray(parsedSponsors) && parsedSponsors.length > 0) {
							eventFormData.sponsorTiers = [{
								tierName: 'Sponsors',
								displayOrder: 0,
								sponsors: parsedSponsors
							}];
						}
					}
				}
			} catch (error) {
				console.error('Error parsing event metadata:', error);
				toast.error('Warning: Some event data could not be parsed');
			}

			// Explicitly set type first
			form.setValue('type', 'event', { shouldValidate: true });

			// Use setValue for each field to ensure proper form state
			Object.entries(eventFormData).forEach(([key, value]) => {
				form.setValue(key as any, value, {
					shouldValidate: true,
					shouldDirty: false,
					shouldTouch: false
				});
			});
		} else {
			// Product-specific data
			const productFormData: ProductFormData = {
				type: 'product',
				name: product.name,
				description: product.description || '',
				imageUrl: product.images[0] || '',
				featured: product.metadata.featured === 'true',
				category: (product.metadata.category as ProductFormData['category']) || 'merchandise',
				hasVariants: product.metadata.has_variants === 'true',
				price: price,
				compareAtPrice: product.metadata.compare_at_price || '',
				inStock: product.metadata.in_stock !== 'false',
				isNew: product.metadata.is_new === 'true',
				maxQuantity: product.metadata.max_quantity || '10',
				variants: [],
			};

			// Explicitly set type first
			form.setValue('type', 'product', { shouldValidate: true });

			// Then set all other fields
			Object.entries(productFormData).forEach(([key, value]) => {
				form.setValue(key as any, value, { shouldValidate: true });
			});
		}

		setActiveTab('create');
	};

	// Submit form (create or update)
	const onSubmit = async (data: FormData) => {

		try {
			// Prepare base metadata
			const metadata: Record<string, string> = {
				type: data.type,
				featured: data.featured.toString(),
			};

			// Base request data
			// eslint-disable-next-line prefer-const
			let requestData: Record<string, unknown> = {
				name: data.name,
				description: data.description,
				images: data.imageUrl ? [data.imageUrl] : [],
				metadata,
			};

			if (data.type === 'product') {
				// Product metadata
				metadata.category = data.category;
				metadata.in_stock = data.inStock?.toString() || '0';
				metadata.is_new = data.isNew.toString();
				if (data.compareAtPrice) {
					metadata.compare_at_price = data.compareAtPrice;
				}

				// Handle variants
				if (data.hasVariants && data.variants && data.variants.length > 0) {
					metadata.has_variants = 'true';
					metadata.max_quantity = data.maxQuantity || '10';

					// Don't include a default price for products with variants
					requestData.default_price = null;

					// Create/update product first
					let productId: string;
					if (editingProduct) {
						await apiClient.put(`/admin/update-product/${editingProduct.id}`, requestData);
						productId = editingProduct.id;
					} else {
						const productResponse = await apiClient.post('/admin/create-product', requestData);
						productId = productResponse.data.productID || productResponse.data.product_id;
					}

					// Then create prices for each variant and find the lowest price
					let lowestPriceId: string | null = null;
					let lowestPrice = Infinity;

					for (const variant of data.variants) {
						const variantPriceData = {
							product: productId,
							unit_amount: Math.round(parseFloat(variant.price) * 100),
							currency: 'usd',
							nickname: variant.variantName,
							metadata: {
								variant: variant.variantName,
								size: variant.size || '',
								color: variant.color || '',
								sku: variant.sku || '',
								in_stock: variant.inStock.toString(),
								stock_quantity: variant.stockQuantity || '',
								sort_order: variant.sortOrder.toString(),
							},
						};

						const priceResponse = await apiClient.post('/admin/create-price', variantPriceData);
						const newPriceId = priceResponse.data.id;

						if (variantPriceData.unit_amount < lowestPrice) {
							lowestPrice = variantPriceData.unit_amount;
							lowestPriceId = newPriceId;
						}
					}

					// Set the lowest price as the default for the product
					if (lowestPriceId) {
						console.log('LOWEST PRICE ID', lowestPriceId)
						await apiClient.post('/admin/set-default-price', {
							productId: productId,
							priceId: lowestPriceId,
						});
					}
				} else {
					// Single price product
					metadata.has_variants = 'false';
					metadata.max_quantity = data.maxQuantity || '0';
					requestData.price = Math.round(parseFloat(data.price || '0') * 100);
					requestData.currency = 'usd';

					if (editingProduct) {
						await apiClient.put(`/admin/update-product/${editingProduct.id}`, requestData);
					} else {
						await apiClient.post('/admin/create-product', requestData);
					}
				}
			} else if (data.type === 'bundle') {
				// Bundle metadata
				metadata.in_stock = data.inStock?.toString() || 'false';
				metadata.bundle_items = JSON.stringify(data.bundleItems);
				metadata.discount_type = data.discountType;
				metadata.discount_value = data.discountValue;
				metadata.max_quantity = data.maxQuantity || '0';

				// Calculate total value for reference
				const totalValue = data.bundleItems.reduce((sum, item) =>
					sum + (item.price * item.quantity), 0
				);
				metadata.total_value = totalValue.toString();

				// Single price for bundle
				requestData.price = Math.round(parseFloat(data.price || '0') * 100);
				requestData.currency = 'usd';

				if (editingProduct) {
					await apiClient.put(`/admin/update-product/${editingProduct.id}`, requestData);
				} else {
					await apiClient.post('/admin/create-product', requestData);
				}
			} else {
				// Event metadata
				const eventDateTime = `${data.eventDate}T${data.eventTime}:00Z`;
				metadata.slug = data.slug;
				metadata.event_date = eventDateTime;
				metadata.location = data.location;
				metadata.capacity = data.capacity;
				metadata.organizer = data.organizer;
				metadata.status = data.status;
				metadata.tags = JSON.stringify(data.tags?.map(t => t.value).filter(Boolean));
				metadata.agenda = JSON.stringify(data.agenda);
				metadata.includes = JSON.stringify(data.includes?.map(i => i.value).filter(Boolean));

				// Save sponsor tiers
				if (data.sponsorTiers && data.sponsorTiers.length > 0) {
					metadata.sponsor_tiers = JSON.stringify(data.sponsorTiers);
					// Clear legacy sponsors field
					metadata.sponsors = JSON.stringify([]);
				} else {
					// Fallback to legacy format if no tiers
					metadata.sponsors = JSON.stringify(data.sponsors || []);
				}

				// In the onSubmit function, update tier handling for events
				if (data.hasTiers && data.priceTiers && data.priceTiers.length > 0) {
					metadata.has_tiers = 'true';
					metadata.available_spots = data.capacity;

					// Don't include a default price for events with tiers
					requestData.default_price = null;

					// Create/update product first
					let productId: string;
					if (editingProduct) {
						await apiClient.put(`/admin/update-product/${editingProduct.id}`, requestData);
						productId = editingProduct.id;
					} else {
						const productResponse = await apiClient.post('/admin/create-product', requestData);
						productId = productResponse.data.productID || productResponse.data.product_id;
					}

					// Then create NEW price tiers only
					for (const tier of data.priceTiers) {
						// Skip if this tier already exists (shouldn't happen with new UI)
						if ((tier as any).id) continue;

						const tierPriceData = {
							product: productId,
							unit_amount: Math.round(parseFloat(tier.price) * 100),
							currency: 'usd',
							nickname: tier.name,
							metadata: {
								tier_name: tier.name,
								description: tier.description || '',
								features: JSON.stringify(tier.features || []),
								max_quantity: tier.maxQuantity || '',
								sort_order: tier.sortOrder.toString(),
							},
						};

						await apiClient.post('/admin/create-price', tierPriceData);
					}
				} else {
					// Single price event
					metadata.has_tiers = 'false';
					metadata.max_quantity = data.maxQuantity || '';
					metadata.available_spots = data.availableSpots || '';
					requestData.price = Math.round(parseFloat(data.price || '') * 100);
					requestData.currency = 'usd';

					if (editingProduct) {
						await apiClient.put(`/admin/update-product/${editingProduct.id}`, requestData);
					} else {
						await apiClient.post('/admin/create-product', requestData);
					}
				}
			}

			toast.success(`${data.type === 'event' ? 'Event' : 'Product'} ${editingProduct ? 'updated' : 'created'} successfully!`);

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
			location: 'Euro Haus Headquarters, Tampa',
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
			sponsors: [
				{ name: 'Porsche USA', logoUrl: 'https://euro-haus.nyc3.cdn.digitaloceanspaces.com/graphics/porsche-logo.png', link: 'https://www.porsche.com' },
				{ name: 'Michelin', logoUrl: 'https://euro-haus.nyc3.cdn.digitaloceanspaces.com/graphics/michelin-logo.png', link: 'https://www.michelin.com' },
			],
			sponsorTiers: [],
			hasTiers: false,
			priceTiers: []
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
			hasVariants: false,
			compareAtPrice: '',
			inStock: true,
			isNew: true,
			variants: [],
		});
		toast.success('Product template loaded');
		setActiveTab('create');
	};

	const loadBundleTemplate = () => {
		setEditingProduct(null);
		setProductType('bundle');
		form.reset({
			type: 'bundle',
			name: 'Euro Haus Bundle',
			description: 'Premium cotton t-shirt with embroidered Euro Haus logo',
			price: '29.99',
			imageUrl: 'https://euro-haus.nyc3.cdn.digitaloceanspaces.com/images/product.jpg',
			featured: false,
			maxQuantity: '10',
			inStock: true,
			bundleItems: [],
			discountType: 'percentage',
			discountValue: '',
		});
		toast.success('Bundle template loaded');
		setActiveTab('create');
	}

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
						<Button variant="outline" onClick={() => navigate({ to: '/admin' })}>
							<LayoutDashboard className="h-4 w-4 mr-2" />
							Dashboard
						</Button>
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
									<Select value={filterType} onValueChange={(value: 'all' | 'product' | 'event') => setFilterType(value)}>
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
														<Image
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
							<div className="grid md:grid-cols-3 gap-4 mb-8">
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
								<Card
									className="cursor-pointer hover:shadow-md transition-shadow"
									onClick={loadBundleTemplate}
								>
									<CardHeader>
										<div className="flex items-center justify-between">
											<CardTitle className="text-lg">Bundle Template</CardTitle>
											<Copy className="h-4 w-4" />
										</div>
										<CardDescription>Load bundle template</CardDescription>
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
														value={field.value}
														onValueChange={(value: 'product' | 'bundle' | 'event') => {
															setProductType(value);
															field.onChange(value);

															// Reset form with appropriate defaults
															if (value === 'event') {
																const currentValues = form.getValues();
																const eventDefaults: EventFormData = {
																	type: 'event',
																	name: currentValues.name || '',
																	description: currentValues.description || '',
																	imageUrl: currentValues.imageUrl || '',
																	featured: currentValues.featured || false,
																	slug: '',
																	eventDate: '',
																	eventTime: '09:00',
																	location: 'Euro Haus Headquarters, Tampa',
																	capacity: '100',
																	organizer: 'Euro Haus Events Team',
																	status: 'upcoming',
																	hasTiers: false,
																	price: '',
																	maxQuantity: '10',
																	priceTiers: [],
																	tags: [{ value: '' }],
																	agenda: [{ time: '9:00 AM', activity: '' }],
																	includes: [{ value: '' }],
																	sponsors: [],
																	availableSpots: '',
																};
																form.reset(eventDefaults);
															} else if (value === 'bundle') {
																const bundleDefaults: BundleFormData = {
																	type: 'bundle',
																	bundleItems: [],
																	discountType: 'percentage',
																	discountValue: '',
																	name: '',
																	description: '',
																	imageUrl: '',
																	featured: false,
																	price: '',
																	inStock: true
																};
																form.reset(bundleDefaults);
															} else {
																const currentValues = form.getValues();
																const productDefaults: ProductFormData = {
																	type: 'product',
																	name: currentValues.name || '',
																	description: currentValues.description || '',
																	imageUrl: currentValues.imageUrl || '',
																	featured: currentValues.featured || false,
																	category: 'merchandise',
																	hasVariants: false,
																	price: '',
																	compareAtPrice: '',
																	inStock: true,
																	isNew: false,
																	maxQuantity: '10',
																	variants: [],
																};
																form.reset(productDefaults);
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
															<SelectItem value="product">
																<div className="flex items-center gap-2">
																	<Package className="h-4 w-4" />
																	Regular Product
																</div>
															</SelectItem>
															<SelectItem value="bundle">
																<div className="flex items-center gap-2">
																	<Gift className="h-4 w-4" />
																	Product Bundle
																</div>
															</SelectItem>
															<SelectItem value="event">
																<div className="flex items-center gap-2">
																	<Calendar className="h-4 w-4" />
																	Event
																</div>
															</SelectItem>
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
										<div className="space-y-6">
											{productType === 'product' && (
												<ProductFormSection
													form={form}
													isEditing={!!editingProduct}
													productId={editingProduct?.id}
												/>
											)}
											{productType === 'bundle' && (
												<BundleFormSection
													form={form as UseFormReturn<BundleFormData>}
												/>
											)}
											{productType === 'event' && (
												<EventFormSection
													form={form}
													isEditing={!!editingProduct}
													eventId={editingProduct?.id}
													onGenerateSlug={generateSlug}
												/>
											)}
										</div>

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
											className="flex-1 disabled:bg-gray-500 disabled:text-white cursor-pointer disabled:cursor-not-allowed"
											size="lg"
											disabled={form.formState.isSubmitting}
											onClick={() => {
												if (!form.formState.isValid) {
													console.log('Attempting to submit with validation errors:', form.formState.errors);
												}
											}}
										>
											{form.formState.isSubmitting ? (
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
