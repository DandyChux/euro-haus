import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { Checkbox } from '~/components/ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { Input } from '~/components/ui/input';
import { Label } from '~/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs';
import { toast } from 'sonner';
import { apiClient } from '~/lib/api';
import { Plus, X, Package, Shirt, Tag, Link as LinkIcon, Loader2 } from 'lucide-react';
import { Image } from '../ui/image';

interface EventProductsManagerProps {
	eventId: string;
	eventName: string;
	tiers?: any[];
}

interface Product {
	id: string;
	name: string;
	description?: string;
	images?: string[];
	price?: {
		id: string;
		unit_amount: number;
		currency: string;
	};
	metadata?: Record<string, string>;
}

interface TierProduct {
	id: string;
	name: string;
	quantity: number;
}

export function EventProductsManager({ eventId, eventName, tiers = [] }: EventProductsManagerProps) {
	const [linkedProducts, setLinkedProducts] = useState<Product[]>([]);
	const [availableProducts, setAvailableProducts] = useState<Product[]>([]);
	const [tierProducts, setTierProducts] = useState<Record<string, TierProduct[]>>({});
	const [loading, setLoading] = useState(false);
	const [selectedProducts, setSelectedProducts] = useState<string[]>([]);
	const [productQuantities, setProductQuantities] = useState<Record<string, number>>({});

	useEffect(() => {
		fetchLinkedProducts();
		fetchAvailableProducts();
	}, [eventId]);

	const fetchLinkedProducts = async () => {
		try {
			const response = await apiClient.get(`/admin/events/${eventId}/linked-products`);
			setLinkedProducts(response.data.linkedProducts || []);

			// Parse tier products
			const tierProds: Record<string, TierProduct[]> = {};
			(response.data.tierProducts || []).forEach((tier: any) => {
				tierProds[tier.tierId] = tier.includedProducts || [];
			});
			setTierProducts(tierProds);
		} catch (error) {
			console.error('Failed to fetch linked products:', error);
		}
	};

	const fetchAvailableProducts = async () => {
		try {
			const response = await apiClient.get('/products');
			// Filter out events and already linked products
			const products = response.data.products.filter((p: any) =>
				p.metadata?.type !== 'event' &&
				!linkedProducts.find(lp => lp.id === p.id)
			);
			setAvailableProducts(products);
		} catch (error) {
			console.error('Failed to fetch products:', error);
		}
	};

	const linkProducts = async () => {
		if (selectedProducts.length === 0) {
			toast.error('Please select at least one product');
			return;
		}

		setLoading(true);
		try {
			await apiClient.post(`/admin/events/${eventId}/link-products`, {
				eventId,
				productIds: selectedProducts
			});

			toast.success('Products linked successfully');
			setSelectedProducts([]);
			await fetchLinkedProducts();
			await fetchAvailableProducts();
		} catch (error) {
			toast.error('Failed to link products');
		} finally {
			setLoading(false);
		}
	};

	const unlinkProduct = async (productId: string) => {
		setLoading(true);
		try {
			await apiClient.delete(`/admin/events/${eventId}/products/${productId}`);
			toast.success('Product unlinked');
			await fetchLinkedProducts();
			await fetchAvailableProducts();
		} catch (error) {
			toast.error('Failed to unlink product');
		} finally {
			setLoading(false);
		}
	};

	const addProductToTier = async (tierId: string, productId: string, quantity: number) => {
		setLoading(true);
		try {
			const currentProducts = tierProducts[tierId] || [];

			// Check if product already exists in tier
			if (currentProducts.find(p => p.id === productId)) {
				toast.error('Product already included in this tier');
				setLoading(false);
				return;
			}

			// Find product details
			const product = [...linkedProducts, ...availableProducts].find(p => p.id === productId);
			if (!product) {
				toast.error('Product not found');
				setLoading(false);
				return;
			}

			const newProducts = [
				...currentProducts,
				{ productId, quantity }
			];

			await apiClient.post('/admin/tiers/add-products', {
				priceId: tierId,
				products: newProducts
			});

			toast.success('Product added to tier');
			await fetchLinkedProducts();
		} catch (error) {
			toast.error('Failed to add product to tier');
		} finally {
			setLoading(false);
		}
	};

	const removeProductFromTier = async (tierId: string, productId: string) => {
		setLoading(true);
		try {
			const currentProducts = tierProducts[tierId] || [];
			const newProducts = currentProducts.filter(p => p.id !== productId);

			await apiClient.post('/admin/tiers/add-products', {
				priceId: tierId,
				products: newProducts.map(p => ({ productId: p.id, quantity: p.quantity || 1 }))
			});

			toast.success('Product removed from tier');
			await fetchLinkedProducts();
		} catch (error) {
			toast.error('Failed to remove product from tier');
		} finally {
			setLoading(false);
		}
	};

	const updateProductQuantity = async (tierId: string, productId: string, quantity: number) => {
		if (quantity < 1) return;

		setLoading(true);
		try {
			const currentProducts = tierProducts[tierId] || [];
			const updatedProducts = currentProducts.map(p =>
				p.id === productId ? { ...p, quantity } : p
			);

			await apiClient.post('/admin/tiers/add-products', {
				priceId: tierId,
				products: updatedProducts.map(p => ({ productId: p.id, quantity: p.quantity || 1 }))
			});

			toast.success('Quantity updated');
			await fetchLinkedProducts();
		} catch (error) {
			toast.error('Failed to update quantity');
		} finally {
			setLoading(false);
		}
	};

	const toggleProductSelection = (productId: string) => {
		setSelectedProducts(prev => {
			if (prev.includes(productId)) {
				return prev.filter(id => id !== productId);
			}
			return [...prev, productId];
		});
	};

	return (
		<Card>
			<CardHeader>
				<CardTitle>Event Products & Add-ons</CardTitle>
				<CardDescription>
					Manage products linked to {eventName}
				</CardDescription>
			</CardHeader>
			<CardContent>
				<Tabs defaultValue="linked">
					<TabsList className="grid w-full grid-cols-3">
						<TabsTrigger value="linked">Linked Products</TabsTrigger>
						<TabsTrigger value="tiers">Tier Bundles</TabsTrigger>
						<TabsTrigger value="add">Add Products</TabsTrigger>
					</TabsList>

					<TabsContent value="linked" className="space-y-4">
						<div className="text-sm text-muted-foreground mb-4">
							These products are available as add-ons when purchasing event tickets.
						</div>

						{linkedProducts.length === 0 ? (
							<div className="text-center py-8 border-2 border-dashed rounded-lg">
								<Package className="w-12 h-12 mx-auto text-muted-foreground mb-2" />
								<p className="text-muted-foreground">No products linked to this event</p>
								<p className="text-sm text-muted-foreground">Add products in the "Add Products" tab</p>
							</div>
						) : (
							<div className="space-y-2">
								{linkedProducts.map(product => (
									<div key={product.id} className="flex items-center justify-between p-3 border rounded-lg hover:bg-accent/50 transition-colors">
										<div className="flex items-center gap-3">
											{product.images?.[0] ? (
												<Image
													src={product.images[0]}
													alt={product.name}
													className="w-12 h-12 object-cover rounded"
												/>
											) : (
												<div className="w-12 h-12 bg-muted rounded flex items-center justify-center">
													<Package className="w-6 h-6 text-muted-foreground" />
												</div>
											)}
											<div>
												<div className="font-medium">{product.name}</div>
												{product.price && (
													<div className="text-sm text-muted-foreground">
														${(product.price.unit_amount / 100).toFixed(2)}
													</div>
												)}
											</div>
										</div>
										<Button
											variant="ghost"
											size="sm"
											onClick={() => unlinkProduct(product.id)}
											disabled={loading}
										>
											<X className="w-4 h-4" />
										</Button>
									</div>
								))}
							</div>
						)}
					</TabsContent>

					<TabsContent value="tiers" className="space-y-4">
						<div className="text-sm text-muted-foreground mb-4">
							Include products with specific ticket tiers. Customers buying these tiers will automatically receive the included products.
						</div>

						{tiers.length === 0 ? (
							<div className="text-center py-8 border-2 border-dashed rounded-lg">
								<Tag className="w-12 h-12 mx-auto text-muted-foreground mb-2" />
								<p className="text-muted-foreground">This event doesn't have tiers</p>
								<p className="text-sm text-muted-foreground">Create tiers in the event settings</p>
							</div>
						) : (
							<div className="space-y-4">
								{tiers.map(tier => (
									<Card key={tier.id || tier.priceId}>
										<CardHeader>
											<div className="flex justify-between items-start">
												<div>
													<CardTitle className="text-base">{tier.name}</CardTitle>
													<CardDescription>
														${(tier.amount).toFixed(2)} {tier.currency?.toUpperCase()}
													</CardDescription>
												</div>
												<Badge variant="outline">Tier</Badge>
											</div>
										</CardHeader>
										<CardContent>
											<div className="space-y-3">
												<Label>Included Products</Label>
												{tierProducts[tier.priceId]?.length > 0 ? (
													<div className="space-y-2">
														{tierProducts[tier.priceId].map((product: TierProduct) => (
															<div key={product.id} className="flex items-center justify-between p-2 bg-muted rounded">
																<div className="flex items-center gap-2">
																	<Shirt className="w-4 h-4" />
																	<span className="text-sm font-medium">{product.name}</span>
																</div>
																<div className="flex items-center gap-2">
																	<div className="flex items-center gap-1">
																		<Button
																			variant="ghost"
																			size="sm"
																			onClick={() => updateProductQuantity(tier.priceId, product.id, product.quantity - 1)}
																			disabled={loading || product.quantity <= 1}
																		>
																			-
																		</Button>
																		<Badge variant="secondary" className="min-w-[40px] justify-center">
																			{product.quantity}x
																		</Badge>
																		<Button
																			variant="ghost"
																			size="sm"
																			onClick={() => updateProductQuantity(tier.priceId, product.id, product.quantity + 1)}
																			disabled={loading}
																		>
																			+
																		</Button>
																	</div>
																	<Button
																		variant="ghost"
																		size="sm"
																		onClick={() => removeProductFromTier(tier.priceId, product.id)}
																		disabled={loading}
																	>
																		<X className="w-3 h-3" />
																	</Button>
																</div>
															</div>
														))}
													</div>
												) : (
													<p className="text-sm text-muted-foreground py-4 text-center border-2 border-dashed rounded">
														No products included in this tier
													</p>
												)}

												<div className="flex gap-2 mt-4">
													<Select
														value=""
														onValueChange={(productId) => {
															if (productId) {
																const quantity = productQuantities[productId] || 1;
																addProductToTier(tier.priceId, productId, quantity);
															}
														}}
														disabled={loading}
													>
														<SelectTrigger>
															<SelectValue placeholder="Add product to tier" />
														</SelectTrigger>
														<SelectContent>
															{[...linkedProducts, ...availableProducts].map(product => (
																<SelectItem key={product.id} value={product.id}>
																	{product.name} - ${((product.price?.unit_amount || 0) / 100).toFixed(2)}
																</SelectItem>
															))}
														</SelectContent>
													</Select>
												</div>
											</div>
										</CardContent>
									</Card>
								))}
							</div>
						)}
					</TabsContent>

					<TabsContent value="add" className="space-y-4">
						<div className="text-sm text-muted-foreground mb-4">
							Select products to link as add-ons for this event. Linked products can be offered during checkout.
						</div>

						{availableProducts.length === 0 ? (
							<div className="text-center py-8 border-2 border-dashed rounded-lg">
								<Package className="w-12 h-12 mx-auto text-muted-foreground mb-2" />
								<p className="text-muted-foreground">No available products to link</p>
								<p className="text-sm text-muted-foreground">Create more products in the Products section</p>
							</div>
						) : (
							<>
								<div className="space-y-2 max-h-96 overflow-y-auto">
									{availableProducts.map(product => (
										<div key={product.id} className="flex items-center space-x-3 p-3 border rounded-lg hover:bg-accent/50 transition-colors">
											<Checkbox
												id={product.id}
												checked={selectedProducts.includes(product.id)}
												onCheckedChange={() => toggleProductSelection(product.id)}
												disabled={loading}
											/>
											<label
												htmlFor={product.id}
												className="flex-1 flex items-center gap-3 cursor-pointer"
											>
												{product.images?.[0] ? (
													<Image
														src={product.images[0]}
														alt={product.name}
														className="w-10 h-10 object-cover rounded"
													/>
												) : (
													<div className="w-10 h-10 bg-muted rounded flex items-center justify-center">
														<Package className="w-5 h-5 text-muted-foreground" />
													</div>
												)}
												<div className="flex-1">
													<div className="font-medium">{product.name}</div>
													{product.price && (
														<div className="text-sm text-muted-foreground">
															${(product.price.unit_amount / 100).toFixed(2)}
														</div>
													)}
												</div>
											</label>
										</div>
									))}
								</div>

								<Button
									onClick={linkProducts}
									disabled={loading || selectedProducts.length === 0}
									className="w-full"
								>
									{loading ? (
										<>
											<Loader2 className="w-4 h-4 mr-2 animate-spin" />
											Linking Products...
										</>
									) : (
										<>
											<LinkIcon className="w-4 h-4 mr-2" />
											Link Selected Products ({selectedProducts.length})
										</>
									)}
								</Button>
							</>
						)}
					</TabsContent>
				</Tabs>
			</CardContent>
		</Card>
	);
}
