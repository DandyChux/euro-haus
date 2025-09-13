import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Check, Users, Star, Sparkles, Car, ChevronDown, ChevronUp } from 'lucide-react';
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
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from './ui/tooltip';

interface TieredPricingProps {
	tiers: TieredPrice[];
	onSelectTier: (tier: TieredPrice, quantity: number) => void;
	showFullDescriptions?: boolean;
	maxDescriptionLength?: number;
}

export function TieredPricing({
	tiers,
	onSelectTier,
	showFullDescriptions = false,
	maxDescriptionLength = 100
}: TieredPricingProps) {
	const [selectedQuantities, setSelectedQuantities] = useState<Record<string, number>>({});
	const [selectedTierId, setSelectedTierId] = useState<string>(tiers[0]?.id || '');
	const [viewMode, setViewMode] = useState<'cards' | 'list'>('cards');
	const [expandedDescriptions, setExpandedDescriptions] = useState<Record<string, boolean>>({});

	const handleQuantityChange = (tierId: string, quantity: number) => {
		setSelectedQuantities(prev => ({ ...prev, [tierId]: quantity }));
	};

	const toggleDescription = (tierId: string) => {
		setExpandedDescriptions(prev => ({ ...prev, [tierId]: !prev[tierId] }));
	};

	// Find the most popular tier based on metadata
	const popularTier = tiers.find(t => t.isMostPopular);
	const popularTierIndex = popularTier ? tiers.indexOf(popularTier) : -1;

	// Get icon based on tier position or popularity
	const getTierIcon = (index: number, tier: TieredPrice) => {
		if (tier.isMostPopular) return <Star className="h-5 w-5" />;
		if (index === 0) return <Users className="h-5 w-5" />;
		if (index === tiers.length - 1) return <Sparkles className="h-5 w-5" />;
		return <Check className="h-5 w-5" />;
	};

	const selectedTier = tiers.find(t => t.id === selectedTierId);

	// Helper function to render description with truncation
	const renderDescription = (tier: TieredPrice, layout: 'grid' | 'mobile-stack' | 'list') => {
		if (!tier.description) return null;

		const isExpanded = expandedDescriptions[tier.id] || showFullDescriptions;
		const needsTruncation = tier.description.length > maxDescriptionLength && !showFullDescriptions;

		// For truncation, we need to be careful with line breaks
		let displayText = tier.description;
		if (!isExpanded && needsTruncation) {
			// Find a good truncation point that doesn't break in the middle of a word
			displayText = tier.description.substring(0, maxDescriptionLength);
			const lastSpace = displayText.lastIndexOf(' ');
			if (lastSpace > maxDescriptionLength * 0.8) {
				displayText = displayText.substring(0, lastSpace);
			}
			displayText += '...';
		}

		// Split text into paragraphs and render with proper spacing
		const paragraphs = displayText.split(/\n\n+/).filter(p => p.trim());

		return (
			<div className="mb-3">
				<div className={cn(
					"text-sm text-muted-foreground space-y-2",
					layout === 'list' && "text-xs"
				)}>
					{paragraphs.map((paragraph, index) => (
						<p key={index} className="whitespace-pre-wrap">
							{paragraph.trim()}
						</p>
					))}
				</div>
				{needsTruncation && (
					<Button
						variant="ghost"
						size="sm"
						className="h-auto p-0 text-xs text-primary hover:bg-transparent mt-1"
						onClick={(e) => {
							e.stopPropagation();
							toggleDescription(tier.id);
						}}
					>
						{isExpanded ? (
							<>Show less <ChevronUp className="h-3 w-3 ml-1" /></>
						) : (
							<>Show more <ChevronDown className="h-3 w-3 ml-1" /></>
						)}
					</Button>
				)}
			</div>
		);
	};

	const renderTierCard = (tier: TieredPrice, index: number, layout: 'grid' | 'mobile-stack') => {
		const isPopular = tier.isMostPopular;
		const cardClasses = cn(
			"relative transition-all hover:shadow-lg",
			tier.soldOut && "opacity-60",
			isPopular && "ring-2 ring-primary",
			layout === 'grid' && "flex flex-col" // Remove "h-full" from here
		);

		const badgeClasses = cn(
			"absolute z-10",
			layout === 'grid' ? "-top-3 left-1/2 -translate-x-1/2 whitespace-nowrap" : "-top-3 left-4"
		);

		return (
			<Card key={tier.id} className={cardClasses}>
				{isPopular && (
					<Badge className={badgeClasses}>
						Most Popular
					</Badge>
				)}
				<CardHeader className="pb-3">
					{layout === 'mobile-stack' ? (
						<div className="flex items-center justify-between">
							<div className="flex items-center gap-2">
								{getTierIcon(index, tier)}
								<CardTitle className="text-lg">{tier.name}</CardTitle>
							</div>
							<div className="text-right">
								<span className="text-2xl font-bold">${tier.amount}</span>
								<span className="text-xs ml-1 block text-muted-foreground">per ticket</span>
							</div>
						</div>
					) : (
						<>
							<div className="flex items-start gap-2 min-h-[3rem]">
								<span className="flex-shrink-0 pt-1">{getTierIcon(index, tier)}</span>
								<CardTitle className="text-base">{tier.name}</CardTitle>
								{tier.requiresVehicleSubmission && (
									<TooltipProvider>
										<Tooltip>
											<TooltipTrigger>
												<Car className="h-5 w-5 text-muted-foreground" />
											</TooltipTrigger>
											<TooltipContent>
												<p>Vehicle submission required</p>
											</TooltipContent>
										</Tooltip>
									</TooltipProvider>
								)}
							</div>
							<CardDescription className="mt-2">
								<span className="text-2xl font-bold">${tier.amount}</span>
								<span className="text-sm ml-1">{tier.currency.toUpperCase()}</span>
							</CardDescription>
						</>
					)}
				</CardHeader>

				<CardContent className={cn("pb-3", layout === 'grid' && "flex-1")}>
					{/* Render description with better formatting */}
					{renderDescription(tier, layout)}

					{/* Features section */}
					<div className={cn(layout === 'grid' && tier.features && tier.features.length > 0 && "min-h-[60px]")}>
						{tier.features && tier.features.length > 0 && (
							<ul className="space-y-1.5">
								{tier.features.slice(0, layout === 'grid' ? 3 : undefined).map((feature, idx) => (
									<li key={idx} className="flex items-start">
										<Check className="h-3 w-3 text-primary mr-1.5 mt-0.5 flex-shrink-0" />
										<span className="text-xs line-clamp-1">{feature}</span>
									</li>
								))}
								{layout === 'grid' && tier.features.length > 3 && (
									<li className="text-xs text-muted-foreground">
										+{tier.features.length - 3} more
									</li>
								)}
							</ul>
						)}
					</div>
				</CardContent>

				<CardFooter className={cn(
					"flex gap-2 pt-3",
					layout === 'grid' ? "flex-col" : "flex-row"
				)}>
					<div className={cn(layout === 'grid' ? "h-9 w-full" : "w-24")}>
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
											{num} {layout !== 'mobile-stack' && (num === 1 ? 'ticket' : 'tickets')}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						)}
					</div>

					<Button
						className={cn(layout === 'grid' ? "w-full h-9 text-sm" : "flex-1")}
						disabled={tier.soldOut}
						variant={isPopular ? "default" : "outline"}
						onClick={() => onSelectTier(tier, selectedQuantities[tier.id] || 1)}
					>
						{tier.soldOut ? 'Sold Out' : (
							layout === 'mobile-stack' ? `Select - $${(tier.amount * (selectedQuantities[tier.id] || 1)).toFixed(2)}` : 'Select'
						)}
					</Button>

					<div className={cn(layout === 'grid' && "h-4")}>
						{tier.maxQuantity && !tier.soldOut && tier.maxQuantity < 20 && (
							<p className="text-xs text-center text-orange-600">
								Only {tier.maxQuantity} left!
							</p>
						)}
					</div>
				</CardFooter>
			</Card>
		);
	};

	// For 1-2 tiers, use a simple grid layout for all screen sizes
	if (tiers.length <= 2) {
		return (
			<div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full">
				{tiers.map((tier, index) => renderTierCard(tier, index, 'grid'))}
			</div>
		);
	}

	// For 3+ tiers, provide a toggle between card and list view on mobile
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

			{/* Desktop: Auto-responsive grid that adjusts based on content */}
			<div className="hidden md:grid grid-cols-[repeat(auto-fit,minmax(min(100%,280px),1fr))] gap-4 items-start">
				{tiers.map((tier, index) => renderTierCard(tier, index, 'grid'))}
			</div>



			{/* Mobile: Show cards or list based on selection */}
			<div className="md:hidden">
				{viewMode === 'cards' ? (
					// Mobile card view - stack vertically
					<div className="space-y-4">
						{tiers.map((tier, index) => renderTierCard(tier, index, 'mobile-stack'))}
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
														{tier.isMostPopular && (
															<Badge variant="secondary" className="text-xs">Popular</Badge>
														)}
														{tier.requiresVehicleSubmission && (
															<TooltipProvider>
																<Tooltip>
																	<TooltipTrigger>
																		<Car className="h-4 w-4 text-muted-foreground" />
																	</TooltipTrigger>
																	<TooltipContent>
																		<p>Vehicle submission required</p>
																	</TooltipContent>
																</Tooltip>
															</TooltipProvider>
														)}
													</div>
													{/* Render description in list view */}
													{renderDescription(tier, 'list')}
												</div>
											</div>
											<div className="text-right ml-3 flex-shrink-0">
												<div className="font-bold">${tier.amount}</div>
												{tier.soldOut && <Badge variant="secondary" className="text-xs">Sold Out</Badge>}
											</div>
										</label>
									))}
								</div>
							</RadioGroup>

							{selectedTier && !selectedTier.soldOut && (
								<div className="mt-4 pt-4 border-t space-y-3">
									{/* Show selected tier's features if any */}
									{selectedTier.features && selectedTier.features.length > 0 && (
										<div className="space-y-1">
											<p className="text-sm font-medium">Includes:</p>
											<ul className="space-y-1">
												{selectedTier.features.map((feature, idx) => (
													<li key={idx} className="flex items-start">
														<Check className="h-3 w-3 text-primary mr-1.5 mt-0.5 flex-shrink-0" />
														<span className="text-xs">{feature}</span>
													</li>
												))}
											</ul>
										</div>
									)}

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
