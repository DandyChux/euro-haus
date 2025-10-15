import { Button } from '~/components/ui/button';
import { Card, CardContent } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Image } from '~/components/ui/image';
import { BundleProduct } from '~/lib/services/stripe-service';
import { ShoppingCart, Plus, TrendingDown } from 'lucide-react';
import { Link } from '@tanstack/react-router';

interface BundleOfferProps {
	bundles: BundleProduct[];
	onAddToCart: (bundle: BundleProduct) => void;
}

export function BundleOfferSection({ bundles, onAddToCart }: BundleOfferProps) {
	if (!bundles || bundles.length === 0) {
		return null;
	}

	return (
		<div className="mt-12">
			<h2 className="text-2xl font-bold mb-6">Frequently Bought Together</h2>
			<div className="space-y-6">
				{bundles.map((bundle) => (
					<Card key={bundle.id} className="overflow-hidden">
						<CardContent className="pt-6">
							<div className="flex flex-col md:flex-row gap-6">
								{/* Product Images */}
								<div className="flex items-center justify-center -space-x-4">
									{bundle.bundleItems.slice(0, 3).map((item, index) => (
										<a key={item.productId} href={`/catalog/${item.productId}`}>
											<div className="relative">
												<Image
													// You'll need a way to get the images for bundled products.
													// This might require an adjustment to how bundles are created/fetched.
													// For now, let's assume a placeholder or that this info is available.
													src={`/placeholder.svg?height=100&width=100`} // Placeholder
													alt={item.productName}
													className="h-20 w-20 rounded-full object-cover ring-2 ring-background"
												/>
												{index < bundle.bundleItems.length - 1 && index < 2 && (
													<div className="absolute top-1/2 -right-2 transform -translate-y-1/2 bg-muted rounded-full p-1">
														<Plus className="h-4 w-4 text-muted-foreground" />
													</div>
												)}
											</div>
										</a>
									))}
								</div>

								{/* Bundle Details */}
								<div className="flex-1">
									<p className="font-semibold">{bundle.title}</p>
									<p className="text-sm text-muted-foreground mt-1">
										Get these {bundle.bundleItems.length} items together and save!
									</p>
									<div className="flex items-baseline gap-2 mt-3">
										<span className="text-2xl font-bold text-primary">
											${bundle.price.toFixed(2)}
										</span>
										<span className="text-md text-muted-foreground line-through">
											${bundle.totalValue.toFixed(2)}
										</span>
									</div>
									<Badge variant="default" className="bg-green-600 mt-2">
										<TrendingDown className="h-3 w-3 mr-1" />
										Save ${bundle.savings.toFixed(2)}
									</Badge>
								</div>

								{/* Action */}
								<div className="flex items-center">
									<Button
										size="lg"
										onClick={() => onAddToCart(bundle)}
										disabled={!bundle.inStock}
									>
										<ShoppingCart className="mr-2 h-5 w-5" />
										{bundle.inStock ? 'Add Bundle to Cart' : 'Out of Stock'}
									</Button>
								</div>
							</div>
						</CardContent>
					</Card>
				))}
			</div>
		</div>
	);
}
