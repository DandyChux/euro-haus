import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { Button } from '~/components/ui/button'
import { Badge } from '~/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card'
import { Separator } from '~/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbSeparator } from '~/components/ui/breadcrumb'
import { Image } from '~/components/ui/image'
import { InfiniteMovingCards } from '~/components/ui/infinite-moving-cards'
import {
	ShoppingCart,
	Heart,
	Share2,
	Truck,
	Shield,
	ArrowLeft,
	Minus,
	Plus,
	Calendar,
	MapPin,
	Users,
	Clock,
	CheckCircle,
	Info,
	Package
} from 'lucide-react'
import { apiClient } from '~/lib/api'
import { useCart } from '~/lib/contexts/cart-context'
import { toast } from 'sonner'
import { cn } from '~/lib/utils'
import { stripeService, ProductVariant, ProductWithVariants, EventWithTiers, TieredPrice } from '~/lib/services/stripe-service'
import { TieredPricing } from '~/components/tiered-pricing'

// Stripe product interface
interface StripeProduct {
	id: string
	name: string
	description: string | null
	images: string[]
	metadata: Record<string, string>
	active: boolean
	default_price: {
		id: string
		unit_amount: number
		currency: string
	} | null
	created: number
	updated: number
}

export const Route = createFileRoute('/catalog/$id')({
	loader: async ({ params }) => {
		try {
			// First fetch the basic product
			const response = await apiClient.get(`/products/${params.id}`)
			const product: StripeProduct = response.data

			if (!product) {
				throw new Error('Product not found')
			}

			// Check if it's an event
			const isEvent = product.metadata.type === 'event';

			if (isEvent) {
				// Try to fetch event with tiers
				const eventWithTiers = await stripeService.getEventWithPriceTiers(params.id);
				if (eventWithTiers) {
					return { product: eventWithTiers, isEvent: true, hasTiers: true };
				}
				// Fall back to regular event
				const transformedEvent = stripeService.transformStripeEventProduct(product);
				return { product: transformedEvent, isEvent: true, hasTiers: false };
			} else {
				// Try to fetch product with variants
				const productWithVariants = await stripeService.getProductWithVariants(params.id);
				if (productWithVariants) {
					return { product: productWithVariants, isEvent: false, hasVariants: true };
				}
				// Fall back to regular product
				const transformedProduct = stripeService.transformStripeProduct(product);
				return { product: transformedProduct, isEvent: false, hasVariants: false };
			}
		} catch (error: any) {
			if (error.response?.status === 404) {
				throw new Error('Product not found')
			}
			throw new Error('Failed to load product details')
		}
	},
	errorComponent: ({ error }) => {
		return (
			<div className="min-h-screen flex items-center justify-center">
				<div className="text-center">
					<p className="text-destructive mb-4">{error.message}</p>
					<div className="space-y-2">
						<Button onClick={() => window.location.href = '/catalog'}>
							<ArrowLeft className="mr-2 h-4 w-4" />
							Back to Catalog
						</Button>
						<Button variant="outline" onClick={() => window.location.reload()}>
							Retry
						</Button>
					</div>
				</div>
			</div>
		)
	},
	component: ProductDetailPage,
})

function ProductDetailPage() {
	const { product, isEvent, hasVariants, hasTiers } = Route.useLoaderData()
	const navigate = useNavigate()
	const { addItem } = useCart()

	const [selectedImage, setSelectedImage] = useState(0)
	const [quantity, setQuantity] = useState(1)
	const [isFavorite, setIsFavorite] = useState(false)
	const [selectedVariant, setSelectedVariant] = useState<ProductVariant | null>(null)
	const [selectedTier, setSelectedTier] = useState<TieredPrice | null>(null)

	// Type guards
	const isEventProduct = (p: any): p is EventWithTiers => isEvent;
	const hasProductVariants = (p: any): p is ProductWithVariants => hasVariants && !isEvent;

	// Event specific data
	const eventData = isEventProduct(product) ? product : null;

	// Determine current price based on variant/tier selection
	const getCurrentPrice = () => {
		if (selectedTier) return selectedTier.amount;
		if (selectedVariant) return selectedVariant.price;
		return product.price;
	}

	const getCurrentPriceId = () => {
		if (selectedTier) return selectedTier.priceId;
		if (selectedVariant) return selectedVariant.priceId;
		return product.priceId;
	}

	const isInStock = () => {
		if (isEvent) {
			return eventData?.status !== 'soldout' && (eventData?.availableSpots || 0) > 0;
		}
		if (selectedVariant) return selectedVariant.inStock;
		return product.inStock !== false;
	}

	const getMaxQuantity = () => {
		if (isEvent && eventData) {
			return Math.min(eventData.availableSpots || 10, eventData.maxQuantity || 10);
		}
		return product.maxQuantity || 10;
	}

	// Format event date
	const formatEventDate = (dateString: string) => {
		const date = new Date(dateString)
		return date.toLocaleDateString('en-US', {
			weekday: 'long',
			year: 'numeric',
			month: 'long',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit',
			timeZoneName: 'short'
		})
	}

	// Handle tier selection for events
	const handleSelectTier = async (tier: TieredPrice, tierQuantity: number) => {
		try {
			const response = await apiClient.post('/checkout/session', {
				priceId: tier.priceId,
				quantity: tierQuantity,
				metadata: {
					eventId: product.id,
					eventName: product.title,
					tierName: tier.name,
				},
			});

			const stripe = window.Stripe?.(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);
			if (stripe && response.data.sessionId) {
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			}
		} catch (error) {
			toast.error('Failed to start checkout');
		}
	};

	// Add to cart handler
	const handleAddToCart = () => {
		if (!getCurrentPriceId()) {
			toast.error('This product is not available for purchase')
			return
		}

		if (!isInStock()) {
			toast.error(isEvent ? 'This event is sold out' : 'This item is out of stock')
			return
		}

		// For products with variants, ensure one is selected
		if (hasVariants && !isEvent && !selectedVariant) {
			toast.error('Please select an option')
			return
		}

		// For events with tiers, ensure one is selected
		if (hasTiers && isEvent && !selectedTier) {
			toast.error('Please select a ticket type')
			return
		}

		const variantName = selectedVariant ? ` - ${selectedVariant.variant}` : '';
		const tierName = selectedTier ? ` - ${selectedTier.name}` : '';

		// Add to cart
		addItem({
			id: product.id,
			priceId: getCurrentPriceId(),
			title: `${product.title}${variantName}${tierName}`,
			description: product.description || '',
			price: getCurrentPrice(),
			quantity,
			imageUrl: selectedVariant?.images?.[0] || product.imageUrl,
			maxQuantity: getMaxQuantity(),
			variant: selectedVariant?.variant,
			type: isEvent ? 'event' : 'product',
			eventDate: eventData?.date,
		})

		toast.success(`Added ${quantity} ${quantity === 1 ? 'item' : 'items'} to cart`)
	}

	// Share handler
	const handleShare = async () => {
		const url = window.location.href
		const text = `Check out ${product.title} at Euro Haus`

		if (navigator.share) {
			try {
				await navigator.share({ title: product.title, text, url })
			} catch (err) {
				console.error('Error sharing:', err)
			}
		} else {
			navigator.clipboard.writeText(url)
			toast.success('Link copied to clipboard')
		}
	}

	// Get all images
	const getAllImages = () => {
		if (selectedVariant?.images && selectedVariant.images.length > 0) {
			return selectedVariant.images;
		}
		if (product.imageUrl) {
			return [product.imageUrl];
		}
		return ['/placeholder.svg?height=600&width=600'];
	}

	const allImages = getAllImages();

	return (
		<div className="min-h-screen bg-background">
			{/* Breadcrumb */}
			<div className="px-4 sm:px-6 lg:px-8 py-4">
				<Breadcrumb>
					<BreadcrumbList>
						<BreadcrumbItem>
							<BreadcrumbLink href="/">Home</BreadcrumbLink>
						</BreadcrumbItem>
						<BreadcrumbSeparator />
						<BreadcrumbItem>
							<BreadcrumbLink href="/catalog">Catalog</BreadcrumbLink>
						</BreadcrumbItem>
						<BreadcrumbSeparator />
						<BreadcrumbItem>
							<BreadcrumbLink>{product.title}</BreadcrumbLink>
						</BreadcrumbItem>
					</BreadcrumbList>
				</Breadcrumb>
			</div>

			<div className="px-4 sm:px-6 lg:px-8 pb-16">
				<div className="mx-auto max-w-7xl">
					{/* Back button */}
					<Button
						variant="ghost"
						className="mb-6"
						onClick={() => navigate({ to: '/catalog' })}
					>
						<ArrowLeft className="mr-2 h-4 w-4" />
						Back to Catalog
					</Button>

					<div className="grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-12">
						{/* Images */}
						<div className="space-y-4">
							<div className="aspect-square overflow-hidden rounded-lg bg-muted">
								<Image
									src={allImages[selectedImage] || allImages[0]}
									alt={product.title}
									className="h-full w-full object-cover"
								/>
							</div>

							{/* Image Thumbnails */}
							{allImages.length > 1 && (
								<div className="grid grid-cols-4 gap-2">
									{allImages.map((image, index) => (
										<button
											key={index}
											onClick={() => setSelectedImage(index)}
											className={cn(
												"aspect-square overflow-hidden rounded-md border-2 bg-muted transition-colors",
												selectedImage === index
													? "border-primary"
													: "border-transparent hover:border-muted-foreground/50"
											)}
										>
											<Image
												src={image}
												alt={`${product.title} ${index + 1}`}
												className="h-full w-full object-cover"
											/>
										</button>
									))}
								</div>
							)}
						</div>

						{/* Product Info */}
						<div className="space-y-6">
							{/* Header */}
							<div>
								<div className="flex items-start justify-between mb-2">
									<h1 className="text-3xl font-bold">{product.title}</h1>
									<button
										onClick={() => setIsFavorite(!isFavorite)}
										className={cn(
											"rounded-full p-2 transition-colors hover:bg-muted",
											isFavorite && "text-red-500"
										)}
									>
										<Heart className="h-6 w-6" fill={isFavorite ? "currentColor" : "none"} />
									</button>
								</div>

								{/* Badges */}
								<div className="flex flex-wrap gap-2 mb-4">
									{isEvent && <Badge variant="secondary">Event</Badge>}
									{product.isNew && <Badge>New</Badge>}
									{product.featured && <Badge variant="secondary">Featured</Badge>}
									{product.compareAtPrice && <Badge variant="destructive">Sale</Badge>}
									{!isInStock() && <Badge variant="outline">{isEvent ? 'Sold Out' : 'Out of Stock'}</Badge>}
								</div>

								{/* Price - Show only if no tiers */}
								{!hasTiers && (
									<div className="flex items-baseline gap-2">
										<span className="text-3xl font-bold">${getCurrentPrice().toFixed(2)}</span>
										{product.compareAtPrice && !selectedVariant && (
											<span className="text-xl text-muted-foreground line-through">
												${product.compareAtPrice.toFixed(2)}
											</span>
										)}
									</div>
								)}

								{/* Category or Event Status */}
								<p className="text-sm text-muted-foreground mt-2">
									{isEvent ? `Status: ${eventData?.status || 'upcoming'}` : `Category: ${product.category?.charAt(0).toUpperCase() + product.category?.slice(1)}`}
								</p>
							</div>

							<Separator />

							{/* Event Details */}
							{isEvent && eventData && (
								<div className="space-y-3">
									{eventData.date && (
										<div className="flex items-center gap-2 text-sm">
											<Calendar className="h-4 w-4 text-muted-foreground" />
											<span>{formatEventDate(eventData.date)}</span>
										</div>
									)}
									{eventData.location && (
										<div className="flex items-center gap-2 text-sm">
											<MapPin className="h-4 w-4 text-muted-foreground" />
											<span>{eventData.location}</span>
										</div>
									)}
									{eventData.capacity && (
										<div className="flex items-center gap-2 text-sm">
											<Users className="h-4 w-4 text-muted-foreground" />
											<span>{eventData.availableSpots} of {eventData.capacity} spots available</span>
										</div>
									)}
									{eventData.organizer && (
										<div className="flex items-center gap-2 text-sm">
											<Info className="h-4 w-4 text-muted-foreground" />
											<span>Organized by {eventData.organizer}</span>
										</div>
									)}
								</div>
							)}

							{/* Description */}
							{product.description && (
								<div className="prose prose-sm max-w-none">
									<p>{product.description}</p>
								</div>
							)}

							{/* Tags */}
							{eventData?.tags && eventData.tags.length > 0 && (
								<div className="flex flex-wrap gap-2">
									{eventData.tags.map((tag: string) => (
										<Badge key={tag} variant="outline">
											{tag}
										</Badge>
									))}
								</div>
							)}

							<Separator />

							{/* Tiered Pricing for Events */}
							{hasTiers && eventData && (eventData as EventWithTiers).priceTiers && (
								<div className="space-y-4">
									<h3 className="font-semibold">Select Ticket Type</h3>
									<TieredPricing
										tiers={(eventData as EventWithTiers).priceTiers}
										onSelectTier={handleSelectTier}
									/>
								</div>
							)}

							{/* Variant Selection for Products */}
							{hasProductVariants(product) && product.variants && product.variants.length > 0 && (
								<div className="space-y-4">
									<h3 className="font-semibold">Select Option:</h3>

									{/* For apparel with sizes */}
									{product.category === 'apparel' && (
										<div className="space-y-3">
											<div>
												<label className="text-sm font-medium mb-2 block">Size</label>
												<div className="grid grid-cols-4 gap-2">
													{product.variants.map((variant) => (
														<Button
															key={variant.id}
															variant={selectedVariant?.id === variant.id ? "default" : "outline"}
															size="sm"
															className={cn(
																!variant.inStock && "opacity-50 cursor-not-allowed"
															)}
															onClick={() => {
																if (variant.inStock) {
																	setSelectedVariant(variant);
																	setSelectedImage(0);
																}
															}}
															disabled={!variant.inStock}
														>
															{variant.size || variant.variant}
														</Button>
													))}
												</div>
											</div>
										</div>
									)}

									{/* For other products */}
									{product.category !== 'apparel' && (
										<div className="grid grid-cols-2 gap-2">
											{product.variants.map((variant) => (
												<Card
													key={variant.id}
													className={cn(
														"p-4 cursor-pointer transition-all",
														selectedVariant?.id === variant.id && "border-primary ring-2 ring-primary",
														!variant.inStock && "opacity-50 cursor-not-allowed"
													)}
													onClick={() => {
														if (variant.inStock) {
															setSelectedVariant(variant);
															setSelectedImage(0);
														}
													}}
												>
													<div className="text-sm font-medium">{variant.variant}</div>
													<div className="text-sm mt-1">${variant.price.toFixed(2)}</div>
													{!variant.inStock && (
														<div className="text-xs text-red-500 mt-1">Out of stock</div>
													)}
												</Card>
											))}
										</div>
									)}
								</div>
							)}

							{/* Purchase Options - Don't show for events with tiers (they have their own checkout) */}
							{(!hasTiers || !isEvent) && (
								<div className="space-y-4">
									{/* Quantity Selector */}
									<div className="flex items-center justify-between">
										<span className="text-sm font-medium">Quantity</span>
										<div className="flex items-center gap-2">
											<Button
												variant="outline"
												size="icon"
												onClick={() => setQuantity(Math.max(1, quantity - 1))}
												disabled={quantity <= 1}
											>
												<Minus className="h-4 w-4" />
											</Button>
											<span className="w-12 text-center">{quantity}</span>
											<Button
												variant="outline"
												size="icon"
												onClick={() => setQuantity(Math.min(getMaxQuantity(), quantity + 1))}
												disabled={quantity >= getMaxQuantity()}
											>
												<Plus className="h-4 w-4" />
											</Button>
										</div>
									</div>

									{/* Total Price */}
									<div className="bg-muted p-4 rounded-lg">
										<div className="flex justify-between items-center">
											<span className="text-sm">Total</span>
											<span className="text-xl font-bold">
												${(getCurrentPrice() * quantity).toFixed(2)}
											</span>
										</div>
									</div>

									{/* Action Buttons */}
									<div className="flex gap-2">
										<Button
											className="flex-1"
											size="lg"
											onClick={handleAddToCart}
											disabled={!isInStock() || (hasVariants && !selectedVariant)}
										>
											<ShoppingCart className="mr-2 h-5 w-5" />
											{!isInStock() ? (isEvent ? 'Sold Out' : 'Out of Stock') : 'Add to Cart'}
										</Button>
										<Button
											variant="outline"
											size="lg"
											onClick={handleShare}
										>
											<Share2 className="h-5 w-5" />
										</Button>
									</div>

									{/* Stock Notice */}
									{isInStock() && getMaxQuantity() < 10 && (
										<p className="text-xs text-center text-orange-600 font-medium">
											Only {getMaxQuantity()} left!
										</p>
									)}
								</div>
							)}

							{/* Trust Badges */}
							{!isEvent && (
								<div className="grid grid-cols-2 gap-4 pt-4">
									<div className="flex items-center gap-2 text-sm text-muted-foreground">
										<Truck className="h-4 w-4" />
										<span>Free shipping over $100</span>
									</div>
									<div className="flex items-center gap-2 text-sm text-muted-foreground">
										<Shield className="h-4 w-4" />
										<span>Secure checkout</span>
									</div>
									<div className="flex items-center gap-2 text-sm text-muted-foreground">
										<Package className="h-4 w-4" />
										<span>Easy returns</span>
									</div>
									<div className="flex items-center gap-2 text-sm text-muted-foreground">
										<CheckCircle className="h-4 w-4" />
										<span>Authentic products</span>
									</div>
								</div>
							)}
						</div>
					</div>

					{/* Event-specific content */}
					{isEvent && eventData && (
						<>
							{/* Agenda */}
							{eventData.agenda && eventData.agenda.length > 0 && (
								<Card className="mt-8">
									<CardHeader>
										<CardTitle>Event Schedule</CardTitle>
									</CardHeader>
									<CardContent>
										<div className="space-y-4">
											{eventData.agenda.map((item, index) => (
												<div key={index} className="flex gap-4">
													<div className="flex items-center gap-2 min-w-[100px]">
														<Clock className="h-4 w-4 text-muted-foreground" />
														<span className="text-sm font-medium">{item.time}</span>
													</div>
													<div className="flex-1">
														<p className="text-sm">{item.activity}</p>
													</div>
												</div>
											))}
										</div>
									</CardContent>
								</Card>
							)}

							{/* What's Included */}
							{eventData.includes && eventData.includes.length > 0 && (
								<Card className="mt-8">
									<CardHeader>
										<CardTitle>What's Included</CardTitle>
									</CardHeader>
									<CardContent>
										<ul className="space-y-2">
											{eventData.includes.map((item, index) => (
												<li key={index} className="flex items-center gap-2">
													<span className="text-primary mt-1">•</span>
													<span className="text-sm">{item}</span>
												</li>
											))}
										</ul>
									</CardContent>
								</Card>
							)}

							{/* Sponsors */}
							{eventData.sponsors && eventData.sponsors.length > 0 && (
								<div className="mt-16">
									<div className="text-center mb-8">
										<h2 className="text-3xl font-bold mb-2">Event Sponsors</h2>
										<p className="text-muted-foreground">Thank you to our partners who make this event possible</p>
									</div>
									<InfiniteMovingCards
										items={eventData.sponsors.map(sponsor => ({
											name: sponsor.name,
											logoUrl: sponsor.logoUrl,
											link: sponsor.link
										}))}
										variant="logo"
										direction="left"
										speed="normal"
										pauseOnHover={true}
										className="mb-8"
									/>
								</div>
							)}
						</>
					)}

					{/* Product Details Tabs */}
					{!isEvent && (
						<Tabs defaultValue="details" className="mt-12">
							<TabsList>
								<TabsTrigger value="details">Details</TabsTrigger>
								<TabsTrigger value="shipping">Shipping & Returns</TabsTrigger>
								<TabsTrigger value="reviews">Reviews</TabsTrigger>
							</TabsList>
							<TabsContent value="details" className="mt-4">
								<Card>
									<CardContent className="pt-6">
										<h3 className="font-semibold mb-2">Product Details</h3>
										<div className="space-y-2 text-sm">
											<p><strong>Product ID:</strong> {product.id}</p>
											<p><strong>Category:</strong> {product.category}</p>
											{hasVariants && (product as ProductWithVariants).variants.length > 0 && (
												<p><strong>Available Options:</strong> {(product as ProductWithVariants).variants.length}</p>
											)}
										</div>
									</CardContent>
								</Card>
							</TabsContent>
							<TabsContent value="shipping" className="mt-4">
								<Card>
									<CardContent className="pt-6">
										<h3 className="font-semibold mb-2">Shipping Information</h3>
										<ul className="space-y-2 text-sm">
											<li>• Free standard shipping on orders over $100</li>
											<li>• Express shipping available at checkout</li>
											<li>• International shipping to select countries</li>
										</ul>
										<h3 className="font-semibold mt-4 mb-2">Return Policy</h3>
										<ul className="space-y-2 text-sm">
											<li>• 30-day return window</li>
											<li>• Items must be unused and in original packaging</li>
											<li>• Free returns on all orders</li>
										</ul>
									</CardContent>
								</Card>
							</TabsContent>
							<TabsContent value="reviews" className="mt-4">
								<Card>
									<CardContent className="pt-6">
										<p className="text-sm text-muted-foreground">No reviews yet. Be the first to review this product!</p>
									</CardContent>
								</Card>
							</TabsContent>
						</Tabs>
					)}

					{/* Related Products */}
					<div className="mt-16">
						<h2 className="text-2xl font-bold mb-6">You Might Also Like</h2>
						<p className="text-muted-foreground">More {isEvent ? 'events' : 'products'} coming soon...</p>
					</div>
				</div>
			</div>
		</div>
	)
}
