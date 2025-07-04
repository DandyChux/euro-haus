// euro-haus/src/components/tiered-pricing.tsx
import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Check, Users, Star, Sparkles } from 'lucide-react';
import { TieredPrice } from '../lib/services/stripe-service';
import { cn } from '../lib/utils';
import { Badge } from './ui/badge';
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "./ui/select";
import { RadioGroup, RadioGroupItem } from "./ui/radio-group";
import { Label } from "./ui/label";

interface TieredPricingProps {
	tiers: TieredPrice[];
	onSelectTier: (tier: TieredPrice, quantity: number) => void;
}

export function TieredPricing({ tiers, onSelectTier }: TieredPricingProps) {
	const [selectedQuantities, setSelectedQuantities] = useState<Record<string, number>>({});
	const [selectedTierId, setSelectedTierId] = useState<string>(tiers[0]?.id || '');
	const [viewMode, setViewMode] = useState<'cards' | 'list'>('cards');

	const handleQuantityChange = (tierId: string, quantity: number) => {
		setSelectedQuantities(prev => ({ ...prev, [tierId]: quantity }));
	};

	// Find the most popular tier (usually middle-priced)
	const popularTierIndex = Math.floor(tiers.length / 2);

	// Get icon based on tier position
	const getTierIcon = (index: number) => {
		if (index === 0) return <Users className="h-5 w-5" />;
		if (index === popularTierIndex) return <Star className="h-5 w-5" />;
		if (index === tiers.length - 1) return <Sparkles className="h-5 w-5" />;
		return <Check className="h-5 w-5" />;
	};

	const selectedTier = tiers.find(t => t.id === selectedTierId);

	// For 1-2 tiers, use side-by-side layout
	// For 3+ tiers, use responsive grid or list view
	if (tiers.length <= 2) {
		return (
			<div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full">
				{tiers.map((tier, index) => (
					<Card
						key={tier.id}
						className={cn(
							"relative transition-all hover:shadow-lg",
							tier.soldOut && "opacity-60",
							index === popularTierIndex && "ring-2 ring-primary"
						)}
					>
						{index === popularTierIndex && (
							<Badge className="absolute -top-3 left-1/2 -translate-x-1/2 z-10">
								Most Popular
							</Badge>
						)}
						<CardHeader>
							<div className="flex items-center gap-2">
								{getTierIcon(index)}
								<CardTitle className="text-lg">{tier.name}</CardTitle>
							</div>
							<CardDescription>
								<span className="text-2xl font-bold">
									${tier.amount}
								</span>
								<span className="text-sm ml-1">{tier.currency.toUpperCase()}</span>
							</CardDescription>
						</CardHeader>

						<CardContent className="space-y-4">
							{tier.description && (
								<p className="text-sm text-muted-foreground">{tier.description}</p>
							)}

							{tier.features && tier.features.length > 0 && (
								<ul className="space-y-2">
									{tier.features.map((feature, idx) => (
										<li key={idx} className="flex items-start">
											<Check className="h-4 w-4 text-primary mr-2 mt-0.5 flex-shrink-0" />
											<span className="text-sm">{feature}</span>
										</li>
									))}
								</ul>
							)}
						</CardContent>

						<CardFooter className="flex flex-col gap-3">
							{!tier.soldOut && tier.maxQuantity && (
								<Select
									value={String(selectedQuantities[tier.id] || 1)}
									onValueChange={(value) => handleQuantityChange(tier.id, parseInt(value))}
								>
									<SelectTrigger className="w-full">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										{Array.from({ length: Math.min(tier.maxQuantity, 10) }, (_, i) => i + 1).map(num => (
											<SelectItem key={num} value={String(num)}>
												{num} {num === 1 ? 'ticket' : 'tickets'}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							)}

							<Button
								className="w-full"
								disabled={tier.soldOut}
								variant={index === popularTierIndex ? "default" : "outline"}
								onClick={() => onSelectTier(tier, selectedQuantities[tier.id] || 1)}
							>
								{tier.soldOut ? 'Sold Out' : 'Select'}
							</Button>

							{tier.maxQuantity && !tier.soldOut && tier.maxQuantity < 20 && (
								<p className="text-xs text-center text-orange-600">
									Only {tier.maxQuantity} left!
								</p>
							)}
						</CardFooter>
					</Card>
				))}
			</div>
		);
	}

	// For 3+ tiers, provide a toggle between card and list view
	return (
		<div className="w-full space-y-4">
			{/* View Mode Toggle - Only show on smaller screens */}
			<div className="flex justify-end md:hidden">
				<div className="flex gap-1 p-1 bg-muted rounded-lg">
					<Button
						size="sm"
						variant={viewMode === 'cards' ? 'default' : 'ghost'}
						onClick={() => setViewMode('cards')}
						className="px-3 py-1 h-8"
					>
						Cards
					</Button>
					<Button
						size="sm"
						variant={viewMode === 'list' ? 'default' : 'ghost'}
						onClick={() => setViewMode('list')}
						className="px-3 py-1 h-8"
					>
						List
					</Button>
				</div>
			</div>

			{/* Desktop: Always show cards in grid */}
			<div className="hidden md:grid md:grid-cols-2 lg:grid-cols-3 gap-4">
				{tiers.map((tier, index) => (
					<Card
						key={tier.id}
						className={cn(
							"relative transition-all hover:shadow-lg h-full flex flex-col",
							tier.soldOut && "opacity-60",
							index === popularTierIndex && "ring-2 ring-primary"
						)}
					>
						{index === popularTierIndex && (
							<Badge className="absolute -top-3 left-1/2 -translate-x-1/2 z-10 whitespace-nowrap">
								Most Popular
							</Badge>
						)}
						<CardHeader className="pb-3">
							<div className="flex items-start justify-between gap-2">
								<div className="flex items-center gap-2 min-w-0">
									{getTierIcon(index)}
									<CardTitle className="text-base truncate">{tier.name}</CardTitle>
								</div>
							</div>
							<CardDescription className="mt-2">
								<span className="text-2xl font-bold">
									${tier.amount}
								</span>
								<span className="text-sm ml-1">{tier.currency.toUpperCase()}</span>
							</CardDescription>
						</CardHeader>

						<CardContent className="flex-1 pb-3">
							{tier.description && (
								<p className="text-sm text-muted-foreground mb-3 line-clamp-2">
									{tier.description}
								</p>
							)}

							{tier.features && tier.features.length > 0 && (
								<ul className="space-y-1.5">
									{tier.features.slice(0, 4).map((feature, idx) => (
										<li key={idx} className="flex items-start">
											<Check className="h-3 w-3 text-primary mr-1.5 mt-0.5 flex-shrink-0" />
											<span className="text-xs line-clamp-2">{feature}</span>
										</li>
									))}
									{tier.features.length > 4 && (
										<li className="text-xs text-muted-foreground">
											+{tier.features.length - 4} more
										</li>
									)}
								</ul>
							)}
						</CardContent>

						<CardFooter className="flex flex-col gap-2 pt-3">
							{!tier.soldOut && tier.maxQuantity && (
								<Select
									value={String(selectedQuantities[tier.id] || 1)}
									onValueChange={(value) => handleQuantityChange(tier.id, parseInt(value))}
								>
									<SelectTrigger className="w-full h-9 text-sm">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										{Array.from({ length: Math.min(tier.maxQuantity, 10) }, (_, i) => i + 1).map(num => (
											<SelectItem key={num} value={String(num)}>
												{num} {num === 1 ? 'ticket' : 'tickets'}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							)}

							<Button
								className="w-full h-9 text-sm"
								disabled={tier.soldOut}
								variant={index === popularTierIndex ? "default" : "outline"}
								onClick={() => onSelectTier(tier, selectedQuantities[tier.id] || 1)}
							>
								{tier.soldOut ? 'Sold Out' : 'Select'}
							</Button>

							{tier.maxQuantity && !tier.soldOut && tier.maxQuantity < 20 && (
								<p className="text-xs text-center text-orange-600">
									Only {tier.maxQuantity} left!
								</p>
							)}
						</CardFooter>
					</Card>
				))}
			</div>

			{/* Mobile: Show cards or list based on selection */}
			<div className="md:hidden">
				{viewMode === 'cards' ? (
					// Mobile card view - stack vertically
					<div className="space-y-4">
						{tiers.map((tier, index) => (
							<Card
								key={tier.id}
								className={cn(
									"relative transition-all",
									tier.soldOut && "opacity-60",
									index === popularTierIndex && "ring-2 ring-primary"
								)}
							>
								{index === popularTierIndex && (
									<Badge className="absolute -top-3 left-4 z-10">
										Most Popular
									</Badge>
								)}
								<CardHeader className="pb-3">
									<div className="flex items-center justify-between">
										<div className="flex items-center gap-2">
											{getTierIcon(index)}
											<CardTitle className="text-lg">{tier.name}</CardTitle>
										</div>
										<div className="text-right">
											<span className="text-2xl font-bold">${tier.amount}</span>
											<span className="text-xs ml-1 block text-muted-foreground">per ticket</span>
										</div>
									</div>
								</CardHeader>

								<CardContent className="pb-3">
									{tier.description && (
										<p className="text-sm text-muted-foreground mb-3">{tier.description}</p>
									)}

									{tier.features && tier.features.length > 0 && (
										<ul className="space-y-1">
											{tier.features.map((feature, idx) => (
												<li key={idx} className="flex items-start">
													<Check className="h-4 w-4 text-primary mr-2 mt-0.5 flex-shrink-0" />
													<span className="text-sm">{feature}</span>
												</li>
											))}
										</ul>
									)}
								</CardContent>

								<CardFooter className="flex gap-2">
									{!tier.soldOut && tier.maxQuantity && (
										<Select
											value={String(selectedQuantities[tier.id] || 1)}
											onValueChange={(value) => handleQuantityChange(tier.id, parseInt(value))}
										>
											<SelectTrigger className="w-24">
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												{Array.from({ length: Math.min(tier.maxQuantity, 10) }, (_, i) => i + 1).map(num => (
													<SelectItem key={num} value={String(num)}>
														{num}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
									)}

									<Button
										className="flex-1"
										disabled={tier.soldOut}
										variant={index === popularTierIndex ? "default" : "outline"}
										onClick={() => onSelectTier(tier, selectedQuantities[tier.id] || 1)}
									>
										{tier.soldOut ? 'Sold Out' : `Select - $${(tier.amount * (selectedQuantities[tier.id] || 1)).toFixed(2)}`}
									</Button>
								</CardFooter>
							</Card>
						))}
					</div>
				) : (
					// Mobile list view - compact radio selection
					<Card>
						<CardContent className="p-4">
							<RadioGroup value={selectedTierId} onValueChange={setSelectedTierId}>
								<div className="space-y-3">
									{tiers.map((tier, index) => (
										<label
											key={tier.id}
											className={cn(
												"flex items-center justify-between p-3 rounded-lg border cursor-pointer transition-colors",
												selectedTierId === tier.id && "border-primary bg-primary/5",
												tier.soldOut && "opacity-60 cursor-not-allowed"
											)}
										>
											<div className="flex items-center gap-3">
												<RadioGroupItem value={tier.id} disabled={tier.soldOut} />
												<div>
													<div className="font-medium flex items-center gap-2">
														{tier.name}
														{index === popularTierIndex && (
															<Badge variant="secondary" className="text-xs">Popular</Badge>
														)}
													</div>
													{tier.description && (
														<p className="text-sm text-muted-foreground mt-0.5">{tier.description}</p>
													)}
												</div>
											</div>
											<div className="text-right">
												<div className="font-bold">${tier.amount}</div>
												{tier.soldOut && <Badge variant="secondary" className="text-xs">Sold Out</Badge>}
											</div>
										</label>
									))}
								</div>
							</RadioGroup>

							{selectedTier && !selectedTier.soldOut && (
								<div className="mt-4 pt-4 border-t space-y-3">
									<div className="flex items-center justify-between">
										<Label>Quantity</Label>
										<Select
											value={String(selectedQuantities[selectedTierId] || 1)}
											onValueChange={(value) => handleQuantityChange(selectedTierId, parseInt(value))}
										>
											<SelectTrigger className="w-24">
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												{Array.from({ length: Math.min(selectedTier.maxQuantity || 10, 10) }, (_, i) => i + 1).map(num => (
													<SelectItem key={num} value={String(num)}>
														{num}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
									</div>

									<Button
										className="w-full"
										onClick={() => onSelectTier(selectedTier, selectedQuantities[selectedTierId] || 1)}
									>
										Continue - ${(selectedTier.amount * (selectedQuantities[selectedTierId] || 1)).toFixed(2)}
									</Button>
								</div>
							)}
						</CardContent>
					</Card>
				)}
			</div>
		</div>
	);
}
