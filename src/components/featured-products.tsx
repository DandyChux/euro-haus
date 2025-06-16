import { useState, useEffect } from 'react';
import { Button } from "~/components/ui/button";
import { Skeleton } from "~/components/ui/skeleton";
import { Link } from '@tanstack/react-router';
import { ProductCard } from './product-card';
import { stripeService, type Product } from '~/lib/services/stripe-service';

// Loading skeleton for product cards
function ProductCardSkeleton() {
	return (
		<div className="group relative flex flex-col overflow-hidden rounded-lg border bg-background">
			<div className="relative aspect-square overflow-hidden bg-muted">
				<Skeleton className="h-full w-full" />
			</div>
			<div className="flex flex-1 flex-col p-4">
				<Skeleton className="h-5 w-3/4 mb-2" />
				<Skeleton className="h-4 w-full mb-1" />
				<Skeleton className="h-4 w-2/3 mb-4" />
				<div className="mt-auto pt-4">
					<div className="flex items-center justify-between">
						<Skeleton className="h-6 w-16" />
						<Skeleton className="h-8 w-8 rounded-full" />
					</div>
				</div>
			</div>
		</div>
	);
}

export default function FeaturedProducts() {
	const [products, setProducts] = useState<Product[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		async function fetchProducts() {
			try {
				setLoading(true);
				setError(null);

				const featuredProducts = await stripeService.getFeaturedProducts(4);
				setProducts(featuredProducts);
			} catch (err) {
				console.error('Failed to fetch featured products:', err);
				setError('Failed to load products');
				setProducts([]);
			} finally {
				setLoading(false);
			}
		}

		fetchProducts();
	}, []);

	// Loading state
	if (loading) {
		return (
			<section className="py-12 md:py-16 lg:py-20">
				<div className="px-4 md:px-6">
					<div className="flex flex-col items-center justify-center space-y-4 text-center">
						<div className="space-y-2">
							<h2 className="text-3xl font-bold tracking-tighter sm:text-4xl md:text-5xl">Featured Products</h2>
							<p className="max-w-[700px] text-muted-foreground md:text-xl/relaxed lg:text-base/relaxed xl:text-xl/relaxed">
								Shop our latest merchandise and exclusive Euro Haus gear.
							</p>
						</div>
					</div>
					<div className="mx-auto grid max-w-5xl grid-cols-1 gap-8 pt-12 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
						{[1, 2, 3, 4].map((i) => (
							<ProductCardSkeleton key={i} />
						))}
					</div>
				</div>
			</section>
		);
	}

	// Error state
	if (error) {
		return (
			<section className="py-12 md:py-16 lg:py-20">
				<div className="px-4 md:px-6">
					<div className="flex flex-col items-center justify-center space-y-4 text-center">
						<div className="space-y-2">
							<h2 className="text-3xl font-bold tracking-tighter sm:text-4xl md:text-5xl">Featured Products</h2>
							<p className="text-muted-foreground">{error}</p>
						</div>
						<Button
							variant="outline"
							onClick={() => window.location.reload()}
						>
							Try Again
						</Button>
					</div>
				</div>
			</section>
		);
	}

	return (
		<section className="py-12 md:py-16 lg:py-20">
			<div className="px-4 md:px-6">
				<div className="flex flex-col items-center justify-center space-y-4 text-center">
					<div className="space-y-2">
						<h2 className="text-3xl font-bold tracking-tighter sm:text-4xl md:text-5xl">Featured Products</h2>
						<p className="max-w-[700px] text-muted-foreground md:text-xl/relaxed lg:text-base/relaxed xl:text-xl/relaxed">
							Shop our latest merchandise and exclusive Euro Haus gear.
						</p>
					</div>
				</div>

				{products.length === 0 ? (
					<div className="text-center pt-12">
						<p className="text-lg text-muted-foreground mb-4">No featured products available at this time.</p>
						<Button asChild>
							<Link to="/catalog">Browse All Products</Link>
						</Button>
					</div>
				) : (
					<>
						<div className="mx-auto grid max-w-5xl grid-cols-1 gap-8 pt-12 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
							{products.map((product) => (
								<ProductCard
									key={product.id}
									id={product.id}
									title={product.title}
									description={product.description}
									price={product.price}
									compareAtPrice={product.compareAtPrice}
									imageUrl={product.imageUrl}
									isNew={product.isNew}
									inStock={product.inStock}
								/>
							))}
						</div>
						<div className="mt-12 flex justify-center">
							<Button asChild>
								<Link to="/catalog">Shop All</Link>
							</Button>
						</div>
					</>
				)}
			</div>
		</section>
	);
}
