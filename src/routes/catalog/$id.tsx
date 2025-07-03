import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { Button } from '~/components/ui/button'
import { Badge } from '~/components/ui/badge'
import { Card, CardContent } from '~/components/ui/card'
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
	Info
} from 'lucide-react'
import { apiClient } from '~/lib/api'
import { useCart } from '~/lib/contexts/cart-context'
import { toast } from 'sonner'
import { cn } from '~/lib/utils'

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
			const response = await apiClient.get(`/products/${params.id}`)
			const product: StripeProduct = response.data

			if (!product) {
				throw new Error('Product not found')
			}

			return { product }
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
	const { product } = Route.useLoaderData()
	const navigate = useNavigate()
	const { addItem } = useCart()

	const [selectedImage, setSelectedImage] = useState(0)
	const [quantity, setQuantity] = useState(1)
	const [isFavorite, setIsFavorite] = useState(false)

	// Parse product metadata
	const type = product.metadata.type || 'product'
	const isEvent = type === 'event'
	const inStock = product.metadata.in_stock !== 'false'
	const isNew = product.metadata.is_new === 'true'
	const isFeatured = product.metadata.featured === 'true'
	const maxQuantity = parseInt(product.metadata.max_quantity || '10')
	const category = product.metadata.category || 'merchandise'
	const compareAtPrice = product.metadata.compare_at_price ?
		parseFloat(product.metadata.compare_at_price) : undefined

	// Event-specific metadata
	const eventDate = product.metadata.event_date
	const location = product.metadata.location
	const capacity = product.metadata.capacity
	const availableSpots = product.metadata.available_spots
	const organizer = product.metadata.organizer
	const status = product.metadata.status
	const tags = product.metadata.tags ?
		JSON.parse(product.metadata.tags).filter(Boolean) : []
	const agenda = product.metadata.agenda ?
		JSON.parse(product.metadata.agenda) : []
	const includes = product.metadata.includes ?
		JSON.parse(product.metadata.includes).filter(Boolean) : []

	// Calculate price
	const price = product.default_price ? product.default_price.unit_amount / 100 : 0
	const discount = compareAtPrice ?
		Math.round(((compareAtPrice - price) / compareAtPrice) * 100) : 0

	// Add to cart handler
	const handleAddToCart = () => {
		if (!product.default_price) {
			toast.error('This product is not available for purchase')
			return
		}

		if (!inStock) {
			toast.error('This item is out of stock')
			return
		}

		if (isEvent && availableSpots && parseInt(availableSpots) < quantity) {
			toast.error(`Only ${availableSpots} spots available`)
			return
		}

		// Add each item individually to cart
		for (let i = 0; i < quantity; i++) {
			addItem({
				id: product.id,
				title: product.name,
				description: product.description || '',
				price: price,
				imageUrl: product.images[0] || '/placeholder.svg?height=400&width=400',
			})
		}

		toast.success(`Added ${quantity} ${quantity === 1 ? 'item' : 'items'} to cart`)
	}

	// Share handler
	const handleShare = async () => {
		const url = window.location.href
		const text = `Check out ${product.name} at Euro Haus`

		if (navigator.share) {
			try {
				await navigator.share({ title: product.name, text, url })
			} catch (err) {
				console.error('Error sharing:', err)
			}
		} else {
			// Fallback to clipboard
			navigator.clipboard.writeText(url)
			toast.success('Link copied to clipboard')
		}
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
							<BreadcrumbLink>{product.name}</BreadcrumbLink>
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
									src={product.images[selectedImage] || '/placeholder.svg?height=600&width=600'}
									alt={product.name}
									className="h-full w-full object-cover"
								/>
							</div>

							{product.images.length > 1 && (
								<div className="grid grid-cols-4 gap-2">
									{product.images.map((image, index) => (
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
												alt={`${product.name} ${index + 1}`}
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
									<h1 className="text-3xl font-bold">{product.name}</h1>
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
									{isNew && <Badge>New</Badge>}
									{isFeatured && <Badge variant="secondary">Featured</Badge>}
									{isEvent && <Badge variant="secondary">Event</Badge>}
									{discount > 0 && <Badge variant="destructive">-{discount}%</Badge>}
									{!inStock && <Badge variant="outline">Out of Stock</Badge>}
								</div>

								{/* Price */}
								<div className="flex items-baseline gap-2">
									<span className="text-3xl font-bold">${price.toFixed(2)}</span>
									{compareAtPrice && (
										<span className="text-xl text-muted-foreground line-through">
											${compareAtPrice.toFixed(2)}
										</span>
									)}
								</div>
							</div>

							<Separator />

							{/* Event Details */}
							{isEvent && (
								<div className="space-y-3">
									{eventDate && (
										<div className="flex items-center gap-2 text-sm">
											<Calendar className="h-4 w-4 text-muted-foreground" />
											<span>{formatEventDate(eventDate)}</span>
										</div>
									)}
									{location && (
										<div className="flex items-center gap-2 text-sm">
											<MapPin className="h-4 w-4 text-muted-foreground" />
											<span>{location}</span>
										</div>
									)}
									{capacity && availableSpots && (
										<div className="flex items-center gap-2 text-sm">
											<Users className="h-4 w-4 text-muted-foreground" />
											<span>{availableSpots} of {capacity} spots available</span>
										</div>
									)}
									{organizer && (
										<div className="flex items-center gap-2 text-sm">
											<Info className="h-4 w-4 text-muted-foreground" />
											<span>Organized by {organizer}</span>
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
							{tags.length > 0 && (
								<div className="flex flex-wrap gap-2">
									{tags.map((tag: string) => (
										<Badge key={tag} variant="outline">
											{tag}
										</Badge>
									))}
								</div>
							)}

							<Separator />

							{/* Purchase Options */}
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
											onClick={() => setQuantity(Math.min(maxQuantity, quantity + 1))}
											disabled={quantity >= maxQuantity || (isEvent && !!availableSpots && quantity >= parseInt(availableSpots))}
										>
											<Plus className="h-4 w-4" />
										</Button>
									</div>
								</div>

								{/* Action Buttons */}
								<div className="flex gap-2">
									<Button
										className="flex-1"
										size="lg"
										onClick={handleAddToCart}
										disabled={!inStock || (isEvent && availableSpots === '0')}
									>
										<ShoppingCart className="mr-2 h-5 w-5" />
										{!inStock ? 'Out of Stock' :
											isEvent && availableSpots === '0' ? 'Sold Out' :
												'Add to Cart'}
									</Button>
									<Button
										variant="outline"
										size="lg"
										onClick={handleShare}
									>
										<Share2 className="h-5 w-5" />
									</Button>
								</div>
							</div>

							{/* Trust Badges */}
							<div className="grid grid-cols-2 gap-4 pt-4">
								<div className="flex items-center gap-2 text-sm text-muted-foreground">
									<Truck className="h-4 w-4" />
									<span>Free shipping on orders over $100</span>
								</div>
								<div className="flex items-center gap-2 text-sm text-muted-foreground">
									<Shield className="h-4 w-4" />
									<span>Secure checkout</span>
								</div>
							</div>
						</div>
					</div>

					{/* Additional Info Tabs */}
					<div className="mt-12">
						<Tabs defaultValue={isEvent ? "details" : "info"} className="w-full">
							<TabsList className="grid w-full grid-cols-3">
								<TabsTrigger value={isEvent ? "details" : "info"}>
									{isEvent ? "Event Details" : "Product Info"}
								</TabsTrigger>
								{isEvent && agenda.length > 0 && (
									<TabsTrigger value="agenda">Agenda</TabsTrigger>
								)}
								<TabsTrigger value="shipping">
									{isEvent ? "What's Included" : "Shipping & Returns"}
								</TabsTrigger>
							</TabsList>

							<TabsContent value={isEvent ? "details" : "info"} className="mt-6">
								<Card>
									<CardContent className="pt-6">
										{isEvent ? (
											<div className="space-y-4">
												<h3 className="font-semibold text-lg">Event Information</h3>
												<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
													<div>
														<p className="text-sm text-muted-foreground">Date & Time</p>
														<p className="font-medium">{eventDate ? formatEventDate(eventDate) : 'TBA'}</p>
													</div>
													<div>
														<p className="text-sm text-muted-foreground">Location</p>
														<p className="font-medium">{location || 'TBA'}</p>
													</div>
													<div>
														<p className="text-sm text-muted-foreground">Status</p>
														<p className="font-medium capitalize">{status || 'Upcoming'}</p>
													</div>
													<div>
														<p className="text-sm text-muted-foreground">Capacity</p>
														<p className="font-medium">{capacity || 'Limited'} attendees</p>
													</div>
												</div>
											</div>
										) : (
											<div className="space-y-4">
												<h3 className="font-semibold text-lg">Product Information</h3>
												<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
													<div>
														<p className="text-sm text-muted-foreground">Category</p>
														<p className="font-medium capitalize">{category}</p>
													</div>
													<div>
														<p className="text-sm text-muted-foreground">SKU</p>
														<p className="font-medium">{product.id}</p>
													</div>
													<div>
														<p className="text-sm text-muted-foreground">Availability</p>
														<p className="font-medium">{inStock ? 'In Stock' : 'Out of Stock'}</p>
													</div>
													<div>
														<p className="text-sm text-muted-foreground">Max per Order</p>
														<p className="font-medium">{maxQuantity} items</p>
													</div>
												</div>
											</div>
										)}
									</CardContent>
								</Card>
							</TabsContent>

							{isEvent && agenda.length > 0 && (
								<TabsContent value="agenda" className="mt-6">
									<Card>
										<CardContent className="pt-6">
											<h3 className="font-semibold text-lg mb-4">Event Agenda</h3>
											<div className="space-y-3">
												{agenda.map((item: { time: string; activity: string }, index: number) => (
													<div key={index} className="flex items-start gap-3">
														<div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/10">
															<Clock className="h-3 w-3 text-primary" />
														</div>
														<div>
															<p className="font-medium">{item.time}</p>
															<p className="text-sm text-muted-foreground">{item.activity}</p>
														</div>
													</div>
												))}
											</div>
										</CardContent>
									</Card>
								</TabsContent>
							)}

							<TabsContent value="shipping" className="mt-6">
								<Card>
									<CardContent className="pt-6">
										{isEvent && includes.length > 0 ? (
											<div className="space-y-4">
												<h3 className="font-semibold text-lg">What's Included</h3>
												<div className="space-y-2">
													{includes.map((item: string, index: number) => (
														<div key={index} className="flex items-start gap-2">
															<CheckCircle className="h-4 w-4 text-green-500 mt-0.5" />
															<span className="text-sm">{item}</span>
														</div>
													))}
												</div>
											</div>
										) : (
											<div className="space-y-4">
												<div>
													<h3 className="font-semibold text-lg mb-2">Shipping Information</h3>
													<ul className="space-y-1 text-sm text-muted-foreground">
														<li>• Free shipping on orders over $100</li>
														<li>• Standard shipping: 5-7 business days</li>
														<li>• Express shipping: 2-3 business days</li>
														<li>• International shipping available</li>
													</ul>
												</div>
												<Separator />
												<div>
													<h3 className="font-semibold text-lg mb-2">Return Policy</h3>
													<ul className="space-y-1 text-sm text-muted-foreground">
														<li>• 30-day return window</li>
														<li>• Items must be unused and in original packaging</li>
														<li>• Free returns on defective items</li>
														<li>• Refunds processed within 5-7 business days</li>
													</ul>
												</div>
											</div>
										)}
									</CardContent>
								</Card>
							</TabsContent>
						</Tabs>

						{/* Sponsors Section - Only for events with sponsors */}
						{isEvent && product.metadata.sponsors && (() => {
							try {
								const sponsors = JSON.parse(product.metadata.sponsors);
								if (sponsors && sponsors.length > 0) {
									return (
										<div className="mt-16">
											<div className="text-center mb-8">
												<h2 className="text-3xl font-bold mb-2">Event Sponsors</h2>
												<p className="text-muted-foreground">Thank you to our partners who make this event possible</p>
											</div>
											<InfiniteMovingCards
												items={sponsors.map((sponsor: any) => ({
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
									);
								}
								return null;
							} catch (error) {
								console.error('Failed to parse sponsors:', error);
								return null;
							}
						})()}
					</div>
				</div>
			</div>
		</div>
	)
}
