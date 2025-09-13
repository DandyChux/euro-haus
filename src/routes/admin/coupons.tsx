import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Skeleton } from '~/components/ui/skeleton';
import { toast } from 'sonner';
import { apiClient } from '~/lib/api';
import {
	Loader2,
	LogOut,
	Plus,
	Trash2,
	Search,
	LayoutDashboard,
	Percent,
	DollarSign,
	Copy,
	Tag,
	Calendar,
	Package,
	CheckCircle,
	XCircle,
	QrCode
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
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from '~/components/ui/dialog';
import { Separator } from '~/components/ui/separator';
import { format } from 'date-fns';
import { Checkbox } from '~/components/ui/checkbox';
import { Label } from '~/components/ui/label';

// Coupon creation schema
const couponFormSchema = z.object({
	name: z.string().min(1, 'Name is required'),
	type: z.enum(['percent', 'fixed']),
	percentOff: z.string().optional(),
	amountOff: z.string().optional(),
	currency: z.string().optional(),
	duration: z.enum(['once', 'repeating', 'forever']),
	durationInMonths: z.string().optional(),
	maxRedemptions: z.string().optional(),
	redeemBy: z.string().optional(),
	applyToProducts: z.array(z.string()).optional(),
	restrictToNewCustomers: z.boolean().optional(),
});

// Promotion code schema
const promoCodeFormSchema = z.object({
	couponId: z.string().min(1, 'Please select a coupon'),
	code: z.string().min(1, 'Code is required').regex(/^[A-Z0-9]+$/, 'Code must be uppercase letters and numbers only'),
	maxRedemptions: z.string().optional(),
	expiresAt: z.string().optional(),
	firstTimeOnly: z.boolean().optional(),
	minimumAmount: z.string().optional(),
});

type CouponFormData = z.infer<typeof couponFormSchema>;
type PromoCodeFormData = z.infer<typeof promoCodeFormSchema>;

interface Coupon {
	id: string;
	name: string;
	percent_off: number | null;
	amount_off: number | null;
	currency: string | null;
	duration: string;
	duration_in_months: number | null;
	max_redemptions: number | null;
	times_redeemed: number;
	redeem_by: number | null;
	valid: boolean;
	created: number;
	metadata: Record<string, any>;
	applies_to?: {
		products: string[];
	};
}

interface PromotionCode {
	id: string;
	code: string;
	coupon: {
		id: string;
		name: string;
	};
	active: boolean;
	max_redemptions: number | null;
	times_redeemed: number;
	expires_at: number | null;
	created: number;
	restrictions?: {
		first_time_transaction: boolean;
		minimum_amount: number | null;
		minimum_amount_currency: string | null;
	};
}

interface Product {
	id: string;
	name: string;
	metadata: {
		type?: string;
	};
}

export const Route = createFileRoute('/admin/coupons')({
	component: AdminCouponsPage,
});

function AdminCouponsPage() {
	return (
		<ProtectedRoute>
			<AdminCouponsContent />
		</ProtectedRoute>
	);
}

function AdminCouponsContent() {
	const navigate = useNavigate();
	const { logout } = useAuth();
	const [activeTab, setActiveTab] = useState('manage');
	const [coupons, setCoupons] = useState<Coupon[]>([]);
	const [products, setProducts] = useState<Product[]>([]);
	const [isRefreshing, setIsRefreshing] = useState(false);
	const [searchQuery, setSearchQuery] = useState('');
	const [filterType, setFilterType] = useState<'all' | 'active' | 'expired'>('all');
	const [couponToDelete, setCouponToDelete] = useState<{ id: string; name: string } | null>(null);
	const [isDeleting, setIsDeleting] = useState(false);
	const [selectedCoupon, setSelectedCoupon] = useState<Coupon | null>(null);
	const [showPromoDialog, setShowPromoDialog] = useState(false);

	// Form for creating coupons
	const couponForm = useForm<CouponFormData>({
		resolver: zodResolver(couponFormSchema),
		defaultValues: {
			name: '',
			type: 'percent',
			percentOff: '',
			amountOff: '',
			currency: 'usd',
			duration: 'once',
			durationInMonths: '',
			maxRedemptions: '',
			redeemBy: '',
			applyToProducts: [],
			restrictToNewCustomers: false,
		},
	});

	// Form for creating promotion codes
	const promoForm = useForm<PromoCodeFormData>({
		resolver: zodResolver(promoCodeFormSchema),
		defaultValues: {
			couponId: '',
			code: '',
			maxRedemptions: '',
			expiresAt: '',
			firstTimeOnly: false,
			minimumAmount: '',
		},
	});

	// Fetch coupons
	const fetchCoupons = async () => {
		setIsRefreshing(true);
		try {
			const response = await apiClient.get('/admin/coupons');
			setCoupons(response.data.coupons || []);
		} catch (error) {
			console.error('Error fetching coupons:', error);
			toast.error('Failed to fetch coupons');
		} finally {
			setIsRefreshing(false);
		}
	};

	// Fetch products for coupon restrictions
	const fetchProducts = async () => {
		try {
			const response = await apiClient.get('/products');
			setProducts(response.data.products || []);
		} catch (error) {
			console.error('Error fetching products:', error);
		}
	};

	// Initialize
	useEffect(() => {
		fetchCoupons();
		fetchProducts();
	}, []);

	// Create coupon
	const onSubmitCoupon = async (data: CouponFormData) => {
		try {
			const payload: any = {
				name: data.name,
				duration: data.duration,
				metadata: {
					created_via: 'admin_dashboard',
				},
			};

			// Set discount amount
			if (data.type === 'percent') {
				payload.percent_off = parseFloat(data.percentOff || '0');
			} else {
				payload.amount_off = Math.round(parseFloat(data.amountOff || '0') * 100);
				payload.currency = data.currency;
			}

			// Set duration
			if (data.duration === 'repeating' && data.durationInMonths) {
				payload.duration_in_months = parseInt(data.durationInMonths);
			}

			// Set limits
			if (data.maxRedemptions) {
				payload.max_redemptions = parseInt(data.maxRedemptions);
			}

			if (data.redeemBy) {
				payload.redeem_by = Math.floor(new Date(data.redeemBy).getTime() / 1000);
			}

			// Apply to specific products
			if (data.applyToProducts && data.applyToProducts.length > 0) {
				payload.applies_to_products = data.applyToProducts;
			}

			await apiClient.post('/admin/coupons', payload);
			toast.success('Coupon created successfully!');
			couponForm.reset();
			fetchCoupons();
			setActiveTab('manage');
		} catch (error) {
			console.error('Error creating coupon:', error);
			toast.error('Failed to create coupon');
		}
	};

	// Create promotion code
	const onSubmitPromoCode = async (data: PromoCodeFormData) => {
		try {
			const payload: any = {
				coupon_id: data.couponId,
				code: data.code,
				metadata: {
					created_via: 'admin_dashboard',
				},
			};

			if (data.maxRedemptions) {
				payload.max_redemptions = parseInt(data.maxRedemptions);
			}

			if (data.expiresAt) {
				payload.expires_at = Math.floor(new Date(data.expiresAt).getTime() / 1000);
			}

			payload.first_time_only = data.firstTimeOnly;

			if (data.minimumAmount) {
				payload.minimum_amount = Math.round(parseFloat(data.minimumAmount) * 100);
				payload.minimum_currency = 'usd';
			}

			await apiClient.post('/admin/promotion-codes', payload);
			toast.success('Promotion code created successfully!');
			promoForm.reset();
			setShowPromoDialog(false);
			setSelectedCoupon(null);
		} catch (error) {
			console.error('Error creating promotion code:', error);
			toast.error('Failed to create promotion code');
		}
	};

	// Delete coupon
	const handleDeleteCoupon = async () => {
		if (!couponToDelete) return;

		setIsDeleting(true);
		try {
			await apiClient.delete(`/admin/coupons/${couponToDelete.id}`);
			toast.success('Coupon deleted successfully');
			fetchCoupons();
			setCouponToDelete(null);
		} catch (error) {
			console.error('Error deleting coupon:', error);
			toast.error('Failed to delete coupon');
		} finally {
			setIsDeleting(false);
		}
	};

	// Filter coupons
	const filteredCoupons = coupons.filter(coupon => {
		// Search filter
		if (searchQuery && !coupon.name.toLowerCase().includes(searchQuery.toLowerCase())) {
			return false;
		}

		// Type filter
		if (filterType === 'active' && !coupon.valid) return false;
		if (filterType === 'expired' && coupon.valid) return false;

		return true;
	});

	// Format discount display
	const formatDiscount = (coupon: Coupon) => {
		if (coupon.percent_off) {
			return `${coupon.percent_off}% off`;
		} else if (coupon.amount_off && coupon.currency) {
			return `$${(coupon.amount_off / 100).toFixed(2)} off`;
		}
		return 'Unknown discount';
	};

	const handleLogout = async () => {
		await logout();
		navigate({ to: '/admin/login' });
	};

	return (
		<div className="min-h-screen bg-background">
			<div className="max-w-7xl mx-auto p-6">
				<div className="flex justify-between items-center mb-8">
					<div>
						<h1 className="text-3xl font-bold">Discount Management</h1>
						<p className="text-muted-foreground">Create and manage coupons and promotion codes</p>
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
							<Tag className="h-4 w-4 mr-2" />
							Manage Discounts
						</TabsTrigger>
						<TabsTrigger value="create">
							<Plus className="h-4 w-4 mr-2" />
							Create Coupon
						</TabsTrigger>
					</TabsList>

					{/* Manage Discounts Tab */}
					<TabsContent value="manage">
						<Card>
							<CardHeader>
								<div className="flex justify-between items-start">
									<div>
										<CardTitle>Active Coupons</CardTitle>
										<CardDescription>View and manage your discount coupons</CardDescription>
									</div>
									<Button
										variant="outline"
										size="sm"
										onClick={fetchCoupons}
										disabled={isRefreshing}
									>
										{isRefreshing ? (
											<Loader2 className="h-4 w-4 animate-spin" />
										) : (
											<CheckCircle className="h-4 w-4" />
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
												placeholder="Search coupons..."
												value={searchQuery}
												onChange={(e) => setSearchQuery(e.target.value)}
												className="pl-10"
											/>
										</div>
									</div>
									<Select value={filterType} onValueChange={(value: 'all' | 'active' | 'expired') => setFilterType(value)}>
										<SelectTrigger className="w-[180px]">
											<SelectValue placeholder="Filter by status" />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="all">All Coupons</SelectItem>
											<SelectItem value="active">Active Only</SelectItem>
											<SelectItem value="expired">Expired Only</SelectItem>
										</SelectContent>
									</Select>
								</div>

								{/* Coupon List */}
								{isRefreshing ? (
									<div className="space-y-4">
										{[...Array(3)].map((_, i) => (
											<Skeleton key={i} className="h-24 w-full" />
										))}
									</div>
								) : filteredCoupons.length === 0 ? (
									<div className="text-center py-12">
										<Tag className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
										<p className="text-muted-foreground mb-4">No coupons found</p>
										<Button onClick={() => setActiveTab('create')}>
											<Plus className="h-4 w-4 mr-2" />
											Create Your First Coupon
										</Button>
									</div>
								) : (
									<div className="space-y-4">
										{filteredCoupons.map((coupon) => (
											<div
												key={coupon.id}
												className="border rounded-lg p-4 hover:shadow-md transition-shadow"
											>
												<div className="flex items-start justify-between">
													<div className="flex-1">
														<div className="flex items-center gap-3 mb-2">
															<h3 className="font-semibold text-lg">{coupon.name}</h3>
															{coupon.valid ? (
																<Badge variant="default">
																	<CheckCircle className="h-3 w-3 mr-1" />
																	Active
																</Badge>
															) : (
																<Badge variant="secondary">
																	<XCircle className="h-3 w-3 mr-1" />
																	Expired
																</Badge>
															)}
															{coupon.percent_off ? (
																<Badge variant="outline">
																	<Percent className="h-3 w-3 mr-1" />
																	Percentage
																</Badge>
															) : (
																<Badge variant="outline">
																	<DollarSign className="h-3 w-3 mr-1" />
																	Fixed Amount
																</Badge>
															)}
														</div>

														<div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
															<div>
																<p className="text-muted-foreground">Discount</p>
																<p className="font-medium">{formatDiscount(coupon)}</p>
															</div>
															<div>
																<p className="text-muted-foreground">Duration</p>
																<p className="font-medium">
																	{coupon.duration === 'once' && 'One time'}
																	{coupon.duration === 'forever' && 'Forever'}
																	{coupon.duration === 'repeating' && `${coupon.duration_in_months} months`}
																</p>
															</div>
															<div>
																<p className="text-muted-foreground">Redemptions</p>
																<p className="font-medium">
																	{coupon.times_redeemed}
																	{coupon.max_redemptions && ` / ${coupon.max_redemptions}`}
																</p>
															</div>
														</div>

														{coupon.redeem_by && (
															<div className="mt-2 text-sm">
																<p className="text-muted-foreground">
																	Expires: {format(new Date(coupon.redeem_by * 1000), 'MMM dd, yyyy')}
																</p>
															</div>
														)}

														{coupon.applies_to?.products && coupon.applies_to.products.length > 0 && (
															<div className="mt-2">
																<Badge variant="secondary" className="text-xs">
																	<Package className="h-3 w-3 mr-1" />
																	Limited to {coupon.applies_to.products.length} products
																</Badge>
															</div>
														)}
													</div>

													<div className="flex gap-2 ml-4">
														<Button
															size="sm"
															variant="outline"
															onClick={() => {
																setSelectedCoupon(coupon);
																promoForm.setValue('couponId', coupon.id);
																setShowPromoDialog(true);
															}}
														>
															<QrCode className="h-4 w-4 mr-1" />
															Create Code
														</Button>
														<Button
															size="sm"
															variant="outline"
															onClick={() => {
																navigator.clipboard.writeText(coupon.id);
																toast.success('Coupon ID copied to clipboard');
															}}
														>
															<Copy className="h-4 w-4" />
														</Button>
														<Button
															size="sm"
															variant="outline"
															onClick={() => setCouponToDelete({ id: coupon.id, name: coupon.name })}
															disabled={!coupon.valid}
														>
															<Trash2 className="h-4 w-4" />
														</Button>
													</div>
												</div>
											</div>
										))}
									</div>
								)}
							</CardContent>
						</Card>
					</TabsContent>

					{/* Create Coupon Tab */}
					<TabsContent value="create">
						<Form {...couponForm}>
							<form onSubmit={couponForm.handleSubmit(onSubmitCoupon)}>
								<Card>
									<CardHeader>
										<CardTitle>Create New Coupon</CardTitle>
										<CardDescription>
											Create a discount coupon that can be applied to products or events
										</CardDescription>
									</CardHeader>
									<CardContent className="space-y-6">
										{/* Basic Info */}
										<FormField
											control={couponForm.control}
											name="name"
											render={({ field }) => (
												<FormItem>
													<FormLabel>Coupon Name</FormLabel>
													<FormControl>
														<Input {...field} placeholder="Summer Sale 20% Off" />
													</FormControl>
													<FormDescription>
														Internal name for this coupon (not shown to customers)
													</FormDescription>
													<FormMessage />
												</FormItem>
											)}
										/>

										{/* Discount Type */}
										<FormField
											control={couponForm.control}
											name="type"
											render={({ field }) => (
												<FormItem>
													<FormLabel>Discount Type</FormLabel>
													<Select value={field.value} onValueChange={field.onChange}>
														<FormControl>
															<SelectTrigger>
																<SelectValue />
															</SelectTrigger>
														</FormControl>
														<SelectContent>
															<SelectItem value="percent">Percentage Off</SelectItem>
															<SelectItem value="fixed">Fixed Amount Off</SelectItem>
														</SelectContent>
													</Select>
													<FormMessage />
												</FormItem>
											)}
										/>

										{/* Discount Amount */}
										{couponForm.watch('type') === 'percent' ? (
											<FormField
												control={couponForm.control}
												name="percentOff"
												render={({ field }) => (
													<FormItem>
														<FormLabel>Percentage Off</FormLabel>
														<FormControl>
															<Input {...field} type="number" min="1" max="100" placeholder="20" />
														</FormControl>
														<FormDescription>
															Enter a value between 1 and 100
														</FormDescription>
														<FormMessage />
													</FormItem>
												)}
											/>
										) : (
											<div className="grid grid-cols-2 gap-4">
												<FormField
													control={couponForm.control}
													name="amountOff"
													render={({ field }) => (
														<FormItem>
															<FormLabel>Amount Off</FormLabel>
															<FormControl>
																<Input {...field} type="number" step="0.01" placeholder="10.00" />
															</FormControl>
															<FormMessage />
														</FormItem>
													)}
												/>
												<FormField
													control={couponForm.control}
													name="currency"
													render={({ field }) => (
														<FormItem>
															<FormLabel>Currency</FormLabel>
															<Select value={field.value} onValueChange={field.onChange}>
																<FormControl>
																	<SelectTrigger>
																		<SelectValue />
																	</SelectTrigger>
																</FormControl>
																<SelectContent>
																	<SelectItem value="usd">USD</SelectItem>
																	<SelectItem value="eur">EUR</SelectItem>
																	<SelectItem value="gbp">GBP</SelectItem>
																</SelectContent>
															</Select>
															<FormMessage />
														</FormItem>
													)}
												/>
											</div>
										)}

										<Separator />

										{/* Duration */}
										<FormField
											control={couponForm.control}
											name="duration"
											render={({ field }) => (
												<FormItem>
													<FormLabel>Duration</FormLabel>
													<Select value={field.value} onValueChange={field.onChange}>
														<FormControl>
															<SelectTrigger>
																<SelectValue />
															</SelectTrigger>
														</FormControl>
														<SelectContent>
															<SelectItem value="once">Once</SelectItem>
															<SelectItem value="repeating">Multiple months</SelectItem>
															<SelectItem value="forever">Forever</SelectItem>
														</SelectContent>
													</Select>
													<FormDescription>
														How long the discount applies (mainly for subscriptions)
													</FormDescription>
													<FormMessage />
												</FormItem>
											)}
										/>

										{couponForm.watch('duration') === 'repeating' && (
											<FormField
												control={couponForm.control}
												name="durationInMonths"
												render={({ field }) => (
													<FormItem>
														<FormLabel>Number of Months</FormLabel>
														<FormControl>
															<Input {...field} type="number" min="1" placeholder="3" />
														</FormControl>
														<FormMessage />
													</FormItem>
												)}
											/>
										)}

										<Separator />

										{/* Restrictions */}
										<div className="space-y-4">
											<h3 className="text-lg font-medium">Restrictions</h3>

											<div className="grid grid-cols-2 gap-4">
												<FormField
													control={couponForm.control}
													name="maxRedemptions"
													render={({ field }) => (
														<FormItem>
															<FormLabel>Max Redemptions (Optional)</FormLabel>
															<FormControl>
																<Input {...field} type="number" min="1" placeholder="100" />
															</FormControl>
															<FormDescription>
																Leave empty for unlimited
															</FormDescription>
															<FormMessage />
														</FormItem>
													)}
												/>

												<FormField
													control={couponForm.control}
													name="redeemBy"
													render={({ field }) => (
														<FormItem>
															<FormLabel>Expiration Date (Optional)</FormLabel>
															<FormControl>
																<Input {...field} type="date" />
															</FormControl>
															<FormDescription>
																Leave empty for no expiration
															</FormDescription>
															<FormMessage />
														</FormItem>
													)}
												/>
											</div>

											<FormField
												control={couponForm.control}
												name="applyToProducts"
												render={({ field }) => (
													<FormItem>
														<FormLabel>Apply to Specific Products (Optional)</FormLabel>
														<FormDescription>
															Leave unchecked to apply to all products
														</FormDescription>
														<div className="space-y-2 mt-2 max-h-48 overflow-y-auto border rounded-md p-3">
															{products.map((product) => (
																<div key={product.id} className="flex items-center space-x-2">
																	<Checkbox
																		checked={field.value?.includes(product.id)}
																		onCheckedChange={(checked) => {
																			const newValue = checked
																				? [...(field.value || []), product.id]
																				: (field.value || []).filter(id => id !== product.id);
																			field.onChange(newValue);
																		}}
																	/>
																	<Label className="text-sm font-normal cursor-pointer">
																		{product.name}
																		{product.metadata.type === 'event' && (
																			<Badge variant="outline" className="ml-2 text-xs">
																				<Calendar className="h-3 w-3 mr-1" />
																				Event
																			</Badge>
																		)}
																	</Label>
																</div>
															))}
														</div>
														<FormMessage />
													</FormItem>
												)}
											/>
										</div>
									</CardContent>
									<CardFooter>
										<Button type="submit" className="w-full" disabled={couponForm.formState.isSubmitting}>
											{couponForm.formState.isSubmitting ? (
												<>
													<Loader2 className="mr-2 h-4 w-4 animate-spin" />
													Creating...
												</>
											) : (
												<>
													<Plus className="mr-2 h-4 w-4" />
													Create Coupon
												</>
											)}
										</Button>
									</CardFooter>
								</Card>
							</form>
						</Form>
					</TabsContent>
				</Tabs>

				{/* Create Promotion Code Dialog */}
				<Dialog open={showPromoDialog} onOpenChange={setShowPromoDialog}>
					<DialogContent>
						<DialogHeader>
							<DialogTitle>Create Promotion Code</DialogTitle>
							<DialogDescription>
								Create a customer-facing code for the coupon "{selectedCoupon?.name}"
							</DialogDescription>
						</DialogHeader>

						<Form {...promoForm}>
							<form onSubmit={promoForm.handleSubmit(onSubmitPromoCode)} className="space-y-4">
								<FormField
									control={promoForm.control}
									name="code"
									render={({ field }) => (
										<FormItem>
											<FormLabel>Promotion Code</FormLabel>
											<FormControl>
												<Input
													{...field}
													placeholder="SUMMER20"
													onChange={(e) => field.onChange(e.target.value.toUpperCase())}
												/>
											</FormControl>
											<FormDescription>
												Customer will enter this code at checkout
											</FormDescription>
											<FormMessage />
										</FormItem>
									)}
								/>

								<div className="grid grid-cols-2 gap-4">
									<FormField
										control={promoForm.control}
										name="maxRedemptions"
										render={({ field }) => (
											<FormItem>
												<FormLabel>Max Uses (Optional)</FormLabel>
												<FormControl>
													<Input {...field} type="number" min="1" placeholder="100" />
												</FormControl>
												<FormMessage />
											</FormItem>
										)}
									/>

									<FormField
										control={promoForm.control}
										name="expiresAt"
										render={({ field }) => (
											<FormItem>
												<FormLabel>Expires (Optional)</FormLabel>
												<FormControl>
													<Input {...field} type="date" />
												</FormControl>
												<FormMessage />
											</FormItem>
										)}
									/>
								</div>

								<FormField
									control={promoForm.control}
									name="firstTimeOnly"
									render={({ field }) => (
										<FormItem className="flex items-center space-x-2">
											<FormControl>
												<Checkbox
													checked={field.value}
													onCheckedChange={field.onChange}
												/>
											</FormControl>
											<FormLabel className="font-normal cursor-pointer">
												First-time customers only
											</FormLabel>
										</FormItem>
									)}
								/>

								<FormField
									control={promoForm.control}
									name="minimumAmount"
									render={({ field }) => (
										<FormItem>
											<FormLabel>Minimum Order Amount (Optional)</FormLabel>
											<FormControl>
												<Input {...field} type="number" step="0.01" placeholder="50.00" />
											</FormControl>
											<FormDescription>
												Minimum purchase amount required to use this code
											</FormDescription>
											<FormMessage />
										</FormItem>
									)}
								/>

								<DialogFooter>
									<Button type="button" variant="outline" onClick={() => setShowPromoDialog(false)}>
										Cancel
									</Button>
									<Button type="submit" disabled={promoForm.formState.isSubmitting}>
										{promoForm.formState.isSubmitting ? (
											<>
												<Loader2 className="mr-2 h-4 w-4 animate-spin" />
												Creating...
											</>
										) : (
											'Create Code'
										)}
									</Button>
								</DialogFooter>
							</form>
						</Form>
					</DialogContent>
				</Dialog>

				{/* Delete Confirmation Dialog */}
				<AlertDialog open={!!couponToDelete} onOpenChange={(open) => !open && setCouponToDelete(null)}>
					<AlertDialogContent>
						<AlertDialogHeader>
							<AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
							<AlertDialogDescription>
								This will permanently delete the coupon "{couponToDelete?.name}".
								This action cannot be undone and any promotion codes associated with this coupon will stop working.
							</AlertDialogDescription>
						</AlertDialogHeader>
						<AlertDialogFooter>
							<AlertDialogCancel>Cancel</AlertDialogCancel>
							<AlertDialogAction
								onClick={handleDeleteCoupon}
								disabled={isDeleting}
								className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
							>
								{isDeleting ? (
									<>
										<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										Deleting...
									</>
								) : (
									'Delete Coupon'
								)}
							</AlertDialogAction>
						</AlertDialogFooter>
					</AlertDialogContent>
				</AlertDialog>
			</div>
		</div>
	);
}
