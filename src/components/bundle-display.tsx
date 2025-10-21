import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Separator } from '~/components/ui/separator';
import { Package, Tag, TrendingDown } from 'lucide-react';
import { BundleProduct, BundleItem } from '~/lib/services/stripe-service';
import { cn } from '~/lib/utils';

interface BundleDisplayProps {
	bundle: BundleProduct;
	className?: string;
}

export function BundleDisplay({ bundle, className }: BundleDisplayProps) {
	const formatPrice = (price: number) => `$${price.toFixed(2)}`;

	const savingsPercentage = bundle.totalValue > 0
		? ((bundle.savings / bundle.totalValue) * 100).toFixed(0)
		: '0';

	return (
		<div className={cn('space-y-6', className)}>
			{/* Bundle Highlight */}
			<Card className="border-primary/50 bg-primary/5">
				<CardContent className="pt-6">
					<div className="flex items-center gap-2 mb-3">
						<Tag className="h-5 w-5 text-primary" />
						<h3 className="font-semibold text-lg">Bundle Deal</h3>
					</div>
					<div className="space-y-2">
						<div className="flex items-baseline gap-2">
							<span className="text-3xl font-bold text-primary">
								{formatPrice(bundle.price)}
							</span>
							<span className="text-lg text-muted-foreground line-through">
								{formatPrice(bundle.totalValue)}
							</span>
						</div>
						<div className="flex items-center gap-2">
							<Badge variant="default" className="bg-green-600">
								<TrendingDown className="h-3 w-3 mr-1" />
								Save {formatPrice(bundle.savings)} ({savingsPercentage}% OFF)
							</Badge>
						</div>
						<p className="text-sm text-muted-foreground mt-2">
							Get all {bundle.bundleItems.length} items together at a discounted price
						</p>
					</div>
				</CardContent>
			</Card>

			{/* Bundle Contents */}
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2">
						<Package className="h-5 w-5" />
						What's Included
					</CardTitle>
				</CardHeader>
				<CardContent>
					<div className="space-y-4">
						{bundle.bundleItems.map((item, index) => (
							<div key={item.productId}>
								<div className="flex items-start justify-between gap-4">
									<div className="flex-1">
										<div className="flex items-center gap-2">
											<h4 className="font-medium">{item.productName}</h4>
											{item.quantity > 1 && (
												<Badge variant="secondary" className="text-xs">
													x{item.quantity}
												</Badge>
											)}
										</div>
										<p className="text-sm text-muted-foreground mt-1">
											{formatPrice(item.price)} each
										</p>
									</div>
									<div className="text-right">
										<p className="font-semibold">
											{formatPrice(item.price * item.quantity)}
										</p>
										{item.quantity > 1 && (
											<p className="text-xs text-muted-foreground">
												{item.quantity} × {formatPrice(item.price)}
											</p>
										)}
									</div>
								</div>
								{index < bundle.bundleItems.length - 1 && (
									<Separator className="mt-4" />
								)}
							</div>
						))}

						<Separator />

						{/* Total Summary */}
						<div className="space-y-2 pt-2">
							<div className="flex justify-between text-sm text-muted-foreground">
								<span>Individual items total:</span>
								<span>{formatPrice(bundle.totalValue)}</span>
							</div>
							<div className="flex justify-between text-sm font-medium text-green-600">
								<span>Bundle discount:</span>
								<span>-{formatPrice(bundle.savings)}</span>
							</div>
							<Separator />
							<div className="flex justify-between text-lg font-bold">
								<span>Bundle Price:</span>
								<span className="text-primary">{formatPrice(bundle.price)}</span>
							</div>
						</div>
					</div>
				</CardContent>
			</Card>

			{/* Value Proposition */}
			<Card className="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/20 border-green-200 dark:border-green-800">
				<CardContent className="pt-6">
					<div className="flex items-start gap-3">
						<div className="p-2 bg-green-100 dark:bg-green-900 rounded-full">
							<TrendingDown className="h-5 w-5 text-green-600 dark:text-green-400" />
						</div>
						<div className="flex-1">
							<h4 className="font-semibold text-green-900 dark:text-green-100 mb-1">
								Why This Bundle?
							</h4>
							<ul className="text-sm text-green-800 dark:text-green-200 space-y-1">
								<li>• Save {savingsPercentage}% compared to buying items separately</li>
								<li>• Carefully curated product combination</li>
								<li>• Everything you need in one purchase</li>
								<li>• Limited time bundle offer</li>
							</ul>
						</div>
					</div>
				</CardContent>
			</Card>

			{/* Bundle Info */}
			<div className="bg-muted/50 rounded-lg p-4 text-sm space-y-2">
				<div className="flex items-center gap-2 text-muted-foreground">
					<Package className="h-4 w-4" />
					<span className="font-medium">Bundle Information</span>
				</div>
				<ul className="space-y-1 text-muted-foreground ml-6">
					<li>• All items shipped together</li>
					<li>• Bundle contents cannot be modified</li>
					<li>• Quantities are per bundle purchased</li>
					<li>• Standard return policy applies to the entire bundle</li>
				</ul>
			</div>
		</div>
	);
}

// Compact version for catalog listings
interface BundleBadgeProps {
	bundle: BundleProduct;
	compact?: boolean;
}

export function BundleBadge({ bundle, compact = false }: BundleBadgeProps) {
	const savingsPercentage = bundle.totalValue > 0
		? ((bundle.savings / bundle.totalValue) * 100).toFixed(0)
		: '0';

	if (compact) {
		return (
			<Badge variant="default" className="bg-gradient-to-r from-green-600 to-emerald-600">
				<Tag className="h-3 w-3 mr-1" />
				Bundle - Save {savingsPercentage}%
			</Badge>
		);
	}

	return (
		<div className="flex flex-wrap gap-2">
			<Badge variant="default" className="bg-gradient-to-r from-green-600 to-emerald-600">
				<Tag className="h-3 w-3 mr-1" />
				Bundle Deal
			</Badge>
			<Badge variant="secondary">
				{bundle.bundleItems.length} Items
			</Badge>
			<Badge variant="outline" className="text-green-600 border-green-600">
				Save {savingsPercentage}%
			</Badge>
		</div>
	);
}
