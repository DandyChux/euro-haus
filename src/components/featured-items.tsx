import { useState, useEffect } from 'react';
import {
	Card,
	CardContent,
	CardHeader,
	CardTitle,
	CardFooter,
	CardDescription
} from './ui/card';
import { Image } from './ui/image';
import { Button } from './ui/button';
import { Skeleton } from './ui/skeleton';
import { Badge } from './ui/badge';
import { cn } from '~/lib/utils';
import { Link } from '@tanstack/react-router';
import { ChevronRight, Calendar, ShoppingCart, Heart } from 'lucide-react';
import { stripeService } from '~/lib/services/stripe-service';
import { useCart } from '~/lib/contexts/cart-context';
import { toast } from 'sonner';

type FeaturedItem = {
	id: string;
	type: 'event' | 'product';
	title: string;
	description: string;
	price: number;
	imageUrl: string;
	slug?: string;
	date?: string;
	location?: string;
	compareAtPrice?: number;
	inStock?: boolean;
	status?: string;
	hasTiers?: boolean;
	lowestPrice?: number;
};

type ItemCardProps = {
	item: FeaturedItem;
	index: number;
	className?: string;
}

// Loading skeleton component
function ItemCardSkeleton({ index }: { index: number }) {
	return (
		<Card className={cn('relative rounded-none p-4 border-none flex flex-col', {
			'bg-secondary/10': index % 2 === 0,
		})}>
			<CardHeader className="px-2 gap-0">
				<Skeleton className="h-6 w-16 mb-2" /> {/* Badge */}
				<Skeleton className="h-8 w-3/4 mb-2" />
				<div className="space-y-2">
					<Skeleton className="h-4 w-full" />
					<Skeleton className="h-4 w-2/3" />
					<Skeleton className="h-4 w-1/3" />
				</div>
			</CardHeader>
			<CardContent className="p-0 flex-1 relative w-full min-h-[200px] max-h-[400px] overflow-hidden">
				<Skeleton className="absolute w-full h-full" />
			</CardContent>
			<CardFooter className="mt-4">
				<Skeleton className="h-10 w-full" />
			</CardFooter>
		</Card>
	);
}

export default function FeaturedItems() {
	const [items, setItems] = useState<FeaturedItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const { addItem } = useCart();

	useEffect(() => {
		async function fetchFeaturedItems() {
			try {
				setLoading(true);
				setError(null);

				// Fetch both events and products in parallel
				const [events, productsResponse] = await Promise.all([
					stripeService.getFeaturedEvents(),
					stripeService.getFeaturedProducts()
				]);

				// Transform events to common format
				const eventItems: FeaturedItem[] = events.map(event => ({
					id: event.id,
					type: 'event' as const,
					title: event.title,
					description: event.description,
					price: event.price,
					imageUrl: event.images[0],
					slug: event.slug,
					date: event.date,
					location: event.location,
					status: event.status,
					hasTiers: event.hasTiers,
					lowestPrice: event.lowestPrice
				}));

				// Transform products to common format
				const productItems: FeaturedItem[] = (productsResponse || []).map((product) => ({
					id: product.id,
					type: 'product' as const,
					title: product.title,
					description: product.description,
					price: product.price,
					imageUrl: product.images[0],
					compareAtPrice: product.compareAtPrice,
					inStock: product.inStock
				}));

				// Combine and shuffle items for a mixed display
				const allItems = [...eventItems, ...productItems];
				const shuffled = allItems.sort(() => Math.random() - 0.5);

				setItems(shuffled);
			} catch (err) {
				console.error('Failed to fetch featured items:', err);
				setError('Failed to load featured items');
				setItems([]);
			} finally {
				setLoading(false);
			}
		}

		fetchFeaturedItems();
	}, []);

	// Loading state
	if (loading) {
		return (
			<div className='grid md:grid-cols-2 lg:grid-cols-3 mx-auto gap-4 p-2 md:p-8'>
				{[0, 1, 2, 3, 4, 5].map((index) => (
					<ItemCardSkeleton key={index} index={index} />
				))}
			</div>
		);
	}

	// Error state
	if (error) {
		return (
			<div className='flex items-center justify-center p-8'>
				<div className='text-center'>
					<p className='text-muted-foreground mb-4'>{error}</p>
					<Button
						variant="outline"
						onClick={() => window.location.reload()}
					>
						Try Again
					</Button>
				</div>
			</div>
		);
	}

	// No items state
	if (items.length === 0) {
		return (
			<div className='flex items-center justify-center p-8'>
				<div className='text-center'>
					<p className='text-lg text-muted-foreground mb-4'>
						No featured items at this time.
					</p>
					<div className="flex gap-4 justify-center">
						<Button asChild>
							<Link to="/events">Browse Events</Link>
						</Button>
						<Button asChild variant="outline">
							<Link to="/catalog">Browse Products</Link>
						</Button>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className='grid md:grid-cols-2 lg:grid-cols-3 mx-auto gap-4 p-2 md:p-8'>
			{items.map((item, index) => (
				<ItemCard key={`${item.type}-${item.id}`} item={item} index={index} />
			))}
		</div>
	);
}

const ItemCard: React.FC<ItemCardProps> = ({
	item,
	index,
	className
}) => {
	const { addItem: addToCart } = useCart();
	const [isFavorite, setIsFavorite] = useState(false);

	// Calculate the price display
	const getPriceDisplay = () => {
		if (item.type === 'event') {
			if (item.hasTiers && item.lowestPrice) {
				return `From $${item.lowestPrice.toFixed(2)}`;
			}
			return `$${item.price.toFixed(2)}`;
		}
		// Product pricing
		return `$${item.price.toFixed(2)}`;
	};

	const handleAddToCart = (e: React.MouseEvent) => {
		e.preventDefault();

		if (item.type === 'product') {
			if (!item.inStock) {
				toast.error('This item is out of stock');
				return;
			}

			addToCart({
				id: item.id,
				title: item.title,
				description: item.description,
				price: item.price,
				imageUrl: item.imageUrl,
			});

			toast.success('Added to cart');
		}
	};

	const handleToggleFavorite = (e: React.MouseEvent) => {
		e.preventDefault();
		e.stopPropagation();
		setIsFavorite(!isFavorite);
	};

	const discount = item.compareAtPrice ? Math.round(((item.compareAtPrice - item.price) / item.compareAtPrice) * 100) : 0;

	return (
		<Card className={cn('relative group rounded-none p-4 border-none flex flex-col', {
			'bg-secondary/10 text-foreground': index % 2 === 0,
		}, className)}>
			{/* Type Badge */}
			<div className="absolute -top-2 -left-2 z-10">
				<Badge
					variant={item.type === 'event' ? 'default' : 'secondary'}
					className={cn(
						"shadow-sm",
						item.type === 'event'
							? "bg-primary/90 text-primary-foreground"
							: "bg-secondary/90 text-secondary-foreground"
					)}
				>
					{item.type === 'event' ? (
						<><Calendar className="h-3 w-3 mr-1" /> Event</>
					) : (
						<><ShoppingCart className="h-3 w-3 mr-1" /> Product</>
					)}
				</Badge>
			</div>

			<CardHeader className={cn('px-2 gap-0', {
				'order-2': index % 2 !== 0,
				'order-1': index % 2 === 0,
			})}>
				<CardTitle className={cn('font-normal text-xl lg:text-3xl tracking-wide my-2', {
					'order-1': index % 2 === 0,
					'order-2': index % 2 !== 0,
				})}>
					{item.title}
				</CardTitle>
				<CardDescription className={cn('font-normal text-sm lg:text-base tracking-wide my-2 flex flex-col', {
					'order-1': index % 2 !== 0,
					'order-2': index % 2 === 0,
				})}>
					{item.type === 'event' && item.date && (
						<span className="font-medium">{new Date(item.date).toLocaleDateString('en-US', {
							year: 'numeric',
							month: 'long',
							day: 'numeric'
						})}</span>
					)}
					<span className="line-clamp-3">{item.description}</span>
					<div className="flex items-center gap-2 mt-1">
						<span className="font-medium text-primary">{getPriceDisplay()}</span>
						{item.compareAtPrice && (
							<>
								<span className="text-sm text-muted-foreground line-through">${item.compareAtPrice.toFixed(2)}</span>
								<Badge variant="destructive" className="text-xs">-{discount}%</Badge>
							</>
						)}
					</div>
				</CardDescription>
			</CardHeader>

			<CardContent className={cn('p-0 flex-1 relative w-full overflow-hidden', {
				'order-1': index % 2 !== 0,
				'order-2': index % 2 === 0,
			})}>
				<div className="relative w-full h-full min-h-[200px] max-h-[400px]">
					<Image
						src={item.imageUrl}
						alt={item.title}
						className='w-full h-full object-contain bg-muted group-hover:scale-105 transition-transform duration-300'
					/>
					{/* Favorite button for products */}
					{item.type === 'product' && (
						<button
							className={cn(
								"absolute right-2 top-2 rounded-full bg-background/80 p-1.5 text-foreground transition-all hover:bg-background",
								isFavorite && "text-red-500",
							)}
							onClick={handleToggleFavorite}
						>
							<Heart className="h-5 w-5" fill={isFavorite ? "currentColor" : "none"} />
						</button>
					)}
				</div>
			</CardContent>

			<CardFooter className='order-3'>
				{item.type === 'event' ? (
					<Button
						asChild
						className='mt-4 group w-full flex items-center gap-1'
						disabled={item.status === 'soldout' || item.status === 'cancelled'}
					>
						<Link to="/events/$slug" params={{ slug: item.slug! }}>
							{item.status === 'soldout' ? 'Sold Out' : 'View Details'}
							<ChevronRight className='h-4 w-4 group-hover:translate-x-1 transition-transform' />
						</Link>
					</Button>
				) : (
					<div className="mt-4 flex gap-2 w-full">
						<Button
							asChild
							variant="outline"
							className='flex-1 group flex items-center gap-1'
						>
							<Link to={`/catalog/$id`} params={{ id: item.id }}>
								View Details
								<ChevronRight className='h-4 w-4 group-hover:translate-x-1 transition-transform' />
							</Link>
						</Button>
						<Button
							size="icon"
							onClick={handleAddToCart}
							disabled={!item.inStock}
						>
							<ShoppingCart className="h-4 w-4" />
						</Button>
					</div>
				)}
			</CardFooter>
		</Card>
	);
};
