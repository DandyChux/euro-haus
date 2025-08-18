import { createFileRoute } from '@tanstack/react-router';
import { Button } from '~/components/ui/button';
import { Image } from '~/components/ui/image';
import { Link } from '@tanstack/react-router';
import { Calendar, MapPin, Clock, ChevronLeft, Share2, Heart, Plus, Minus, Users, Car } from 'lucide-react';
import { Badge } from '~/components/ui/badge';
import { Separator } from '~/components/ui/separator';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Input } from '~/components/ui/input';
import { stripeService, TieredPrice, EventWithTiers } from '~/lib/services/stripe-service';
import { useCart } from '~/lib/contexts/cart-context';
import { useMemo, useState } from 'react';
import { toast } from 'sonner';
import { apiClient } from '~/lib/api';
import { TieredPricing } from '~/components/tiered-pricing';
import { Dialog, DialogContent } from '~/components/ui/dialog';
import { VehicleSubmissionForm } from '~/components/vehicle-submission-form';
import { loadStripe } from '@stripe/stripe-js';
import { EventSponsorTiers } from '~/components/event-sponsor-tiers';
import { MapLocation } from '~/components/ui/map-location';

export const Route = createFileRoute('/events/$slug')({
	loader: async ({ params }) => {
		// First try to get the event with price tiers
		const eventWithTiers = await stripeService.getEventWithPriceTiers(params.slug);

		if (eventWithTiers && eventWithTiers.priceTiers.length > 0) {
			// Event has tiers
			return {
				event: eventWithTiers,
				hasTiers: true,
				singlePriceInfo: null
			};
		}

		// If no tiers, get the basic event and fetch single price info
		const event = await stripeService.getEventBySlug(params.slug);
		if (!event) {
			throw new Error('Event not found');
		}

		// For single-price events, fetch the default price metadata
		let singlePriceInfo = null;
		if (event.priceId) {
			try {
				const response = await apiClient.get<{ prices: any[] }>(`/products/${event.id}/prices`);
				// Find the default price
				const defaultPrice = response.data.prices.find(p => p.id === event.priceId);
				if (defaultPrice) {
					singlePriceInfo = {
						requiresVehicleSubmission: defaultPrice.metadata?.requires_vehicle_submission === 'true',
						isMostPopular: defaultPrice.metadata?.is_most_popular === 'true'
					};
				}
			} catch (error) {
				console.error('Failed to fetch price metadata:', error);
			}
		}

		return {
			event,
			hasTiers: false,
			singlePriceInfo
		};
	},
	component: EventDetailPage,
});

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);

function EventDetailPage() {
	const { event, hasTiers, singlePriceInfo } = Route.useLoaderData();
	const { addItem } = useCart();
	const [quantity, setQuantity] = useState(1);
	const [selectedTier, setSelectedTier] = useState<TieredPrice | null>(null);
	const [showSubmissionForm, setShowSubmissionForm] = useState(false);

	// Determine if single price requires vehicle submission
	const singlePriceRequiresVehicle = singlePriceInfo?.requiresVehicleSubmission;

	// Parse sponsor data
	const sponsorTiers = useMemo(() => {
		try {
			if (event.sponsorTiers) {
				return event.sponsorTiers;
			}
			if (event.sponsors && event.sponsors.length > 0) {
				return [{
					tierName: 'Event Sponsors',
					displayOrder: 0,
					sponsors: event.sponsors
				}];
			}
			return [];
		} catch (error) {
			console.error('Error parsing sponsors:', error);
			return [];
		}
	}, [event]);

	const handleSelectTier = async (tier: TieredPrice, tierQuantity: number) => {
		// If tier requires vehicle submission, show submission form
		if (tier.requiresVehicleSubmission) {
			setSelectedTier(tier);
			setQuantity(tierQuantity);
			setShowSubmissionForm(true);
			return;
		}

		try {
			const response = await apiClient.post('/create-checkout-session', {
				line_items: [
					{
						price: tier.priceId,
						quantity: tierQuantity,
					}
				],
				success_url: `${window.location.origin}/checkout/success`,
				cancel_url: `${window.location.origin}/checkout/cancel`,
				metadata: {
					event_id: event?.id,
					event_name: event?.title,
					tier_name: tier.name,
				},
			});

			const stripe = await stripePromise;
			if (stripe && response.data.sessionId) {
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			}
		} catch (error) {
			console.error('Checkout error:', error);
			toast.error('Failed to start checkout');
		}
	};

	const handleSingleCheckout = async () => {
		// For single price events, check if vehicle submission is required
		if (!hasTiers && singlePriceRequiresVehicle) {
			setShowSubmissionForm(true);
			return;
		}

		// Regular checkout flow
		try {
			const response = await apiClient.post('/create-checkout-session', {
				priceId: event?.priceId,
				quantity,
			});

			const stripe = await stripePromise;
			if (stripe && response.data.sessionId) {
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			}
		} catch (error) {
			console.error('Checkout error:', error);
			toast.error('Failed to start checkout');
		}
	};

	const handleSubmissionSuccess = async (newSubmissionId: string) => {
		setShowSubmissionForm(false);

		try {
			const priceId = selectedTier?.priceId || event?.priceId;
			const response = await apiClient.post('/create-participant-checkout', {
				submissionId: newSubmissionId,
				priceId: priceId,
				eventName: event?.title,
				quantity: quantity,
			});

			const stripe = await stripePromise;
			if (stripe && response.data.sessionId) {
				await stripe.redirectToCheckout({ sessionId: response.data.sessionId });
			}
		} catch (error) {
			console.error('Checkout error:', error);
			toast.error('Failed to create checkout session');
		}
	};

	const handleTierSelection = (tier: TieredPrice) => {
		setSelectedTier(tier);
		setQuantity(1);
	};

	const handleAddToCart = () => {
		if (event.status === 'soldout') {
			toast.error('This event is sold out');
			return;
		}

		// If has tiers and no tier selected
		if (hasTiers && (event as EventWithTiers).priceTiers && !selectedTier) {
			toast.error('Please select a ticket type');
			return;
		}

		// Check if we need vehicle submission based on current selection
		const requiresVehicle = hasTiers
			? selectedTier?.requiresVehicleSubmission
			: singlePriceRequiresVehicle;

		if (requiresVehicle) {
			setShowSubmissionForm(true);
			return;
		}

		const ticketPrice = selectedTier ? selectedTier.amount : event.price;
		const ticketType = selectedTier ? selectedTier.name : 'General Admission';
		const maxAvailable = selectedTier?.maxQuantity ?
			Math.min(selectedTier.maxQuantity, event.availableSpots || 10) :
			(event.availableSpots || event.maxQuantity || 10);

		if (quantity > maxAvailable) {
			toast.error(`Only ${maxAvailable} tickets remaining for this type`);
			return;
		}

		addItem({
			id: event.id,
			priceId: selectedTier?.priceId || event.priceId,
			title: `${event.title} - ${ticketType}`,
			description: `Event on ${new Date(event.date).toLocaleDateString()}`,
			price: ticketPrice,
			quantity,
			imageUrl: event.imageUrl,
			maxQuantity: maxAvailable,
			type: 'event',
			eventDate: event.date,
		});

		toast.success(`Added ${quantity} ${ticketType} ticket${quantity > 1 ? 's' : ''} to cart`);
	};

	const handleShare = async () => {
		if (navigator.share) {
			try {
				await navigator.share({
					title: event.title,
					text: event.description,
					url: window.location.href,
				});
			} catch (error) {
				console.log('Error sharing:', error);
			}
		} else {
			navigator.clipboard.writeText(window.location.href);
			toast.success('Link copied to clipboard');
		}
	};

	const getMapUrl = () => {
		const encodedAddress = encodeURIComponent(event.location);
		return `https://www.google.com/maps/embed/v1/place?key=${import.meta.env.VITE_YOUTUBE_API_KEY || ''}&q=${encodedAddress}`;
	};

	if (!event) {
		return (
			<div className="min-h-screen flex items-center justify-center">
				<div className="text-center">
					<h2 className="text-2xl font-bold mb-4">Event not found</h2>
					<Button asChild>
						<Link to="/events">
							<ChevronLeft className="mr-2 h-4 w-4" />
							Back to Events
						</Link>
					</Button>
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-gradient-to-br from-background via-muted/20 to-background">
			{/* Hero Section with Enhanced Visuals */}
			<div className="relative overflow-hidden">
				<div className="absolute inset-0 bg-gradient-to-br from-primary/20 via-secondary/10 to-accent/15 animate-pulse" />
				<div className="w-full h-[650px] relative">
					<div className="absolute inset-0 bg-gradient-to-b from-transparent via-background/20 to-background/60 z-10" />
					<Image
						src={event.imageUrl}
						alt={event.title}
						className="w-full h-full object-cover"
					/>
					{/* Decorative elements */}
					<div className="absolute top-0 left-0 w-64 h-64 bg-gradient-to-br from-primary/30 to-transparent rounded-full blur-3xl" />
					<div className="absolute bottom-0 right-0 w-96 h-96 bg-gradient-to-tl from-accent/20 to-transparent rounded-full blur-3xl" />
				</div>

				{/* Hero Content Overlay */}
				<div className="absolute bottom-0 left-0 right-0 z-20 p-8 bg-gradient-to-t from-background via-background/95 to-transparent">
					<div className="max-w-7xl mx-auto">
						<div className="flex flex-wrap items-center gap-3 mb-6">
							{event.tags?.map((tag) => (
								<Badge
									key={tag}
									className="bg-gradient-to-r from-primary/80 to-secondary/80 text-primary-foreground border-0 px-4 py-1.5 text-sm font-medium shadow-lg backdrop-blur-sm"
								>
									{tag}
								</Badge>
							))}
							{event.status === 'soldout' && (
								<Badge className="bg-gradient-to-r from-destructive to-destructive/80 text-destructive-foreground border-0 px-4 py-1.5 shadow-lg animate-pulse">
									SOLD OUT
								</Badge>
							)}
						</div>
						<h1 className="text-5xl lg:text-6xl font-bold mb-4 bg-gradient-to-r from-foreground via-primary to-accent bg-clip-text text-transparent animate-gradient bg-300%">
							{event.title}
						</h1>
						<p className="text-xl text-muted-foreground max-w-3xl leading-relaxed">
							{event.description}
						</p>
					</div>
				</div>
			</div>

			{/* Main Content */}
			<div className="max-w-[1750px] mx-auto px-4 py-12">
				{/* Sponsors Section */}
				{sponsorTiers.length > 0 && (
					<section className="relative py-8 mb-12 rounded-3xl overflow-hidden">
						{/* Background gradients */}
						<div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-secondary/5 to-accent/5" />
						<div className="absolute top-0 left-0 w-96 h-96 bg-gradient-to-br from-chart-1/20 to-transparent rounded-full blur-3xl" />
						<div className="absolute bottom-0 right-0 w-96 h-96 bg-gradient-to-tl from-chart-2/20 to-transparent rounded-full blur-3xl" />

						<div className="relative mx-auto px-6">
							<h2 className="text-4xl font-bold text-center mb-12">
								<span className="bg-gradient-to-r from-primary via-secondary to-accent bg-clip-text text-transparent animate-gradient bg-300%">
									Event Partners
								</span>
							</h2>
							<EventSponsorTiers sponsorTiers={sponsorTiers} />
						</div>
					</section>
				)}
				<div className="grid lg:grid-cols-3 gap-10 mb-12">
					{/* Left Column - Main Info */}
					<div className="lg:col-span-2 space-y-8">
						{/* Event Details */}
						<div className="relative group">
							<div className="absolute -inset-0.5 bg-gradient-to-r from-primary via-secondary to-accent rounded-2xl blur opacity-30 group-hover:opacity-50 transition duration-1000 group-hover:duration-200"></div>
							<Card className="relative shadow-neumorph hover:shadow-neumorph-hover transition-all duration-300">
								<CardHeader className="bg-gradient-to-r from-primary/5 to-secondary/5 rounded-t-xl">
									<CardTitle className="text-2xl bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">
										Event Details
									</CardTitle>
								</CardHeader>
								<CardContent className="space-y-6 pt-8">
									<div className="flex items-start gap-4 p-4 rounded-xl bg-gradient-to-r from-primary/5 to-transparent hover:from-primary/10 transition-all">
										<div className="p-3 bg-gradient-to-br from-primary to-secondary rounded-xl shadow-lg">
											<Calendar className="h-6 w-6 text-primary-foreground" />
										</div>
										<div className="flex-1">
											<p className="font-semibold text-lg mb-1">Date & Time</p>
											<p className="text-muted-foreground">
												{new Date(event.date).toLocaleDateString('en-US', {
													weekday: 'long',
													year: 'numeric',
													month: 'long',
													day: 'numeric'
												})}
											</p>
										</div>
									</div>

									<div className="flex items-start gap-4 p-4 rounded-xl bg-gradient-to-r from-secondary/5 to-transparent hover:from-secondary/10 transition-all">
										<div className="p-3 bg-gradient-to-br from-secondary to-accent rounded-xl shadow-lg">
											<MapPin className="h-6 w-6 text-secondary-foreground" />
										</div>
										<div className="flex-1">
											<p className="font-semibold text-lg mb-1">Venue</p>
											<p className="text-muted-foreground">{event.location}</p>
										</div>
									</div>

									{event.capacity && (
										<div className="flex items-start gap-4 p-4 rounded-xl bg-gradient-to-r from-accent/5 to-transparent hover:from-accent/10 transition-all">
											<div className="p-3 bg-gradient-to-br from-accent to-chart-1 rounded-xl shadow-lg">
												<Users className="h-6 w-6 text-accent-foreground" />
											</div>
											<div className="flex-1">
												<p className="font-semibold text-lg mb-1">Capacity</p>
												<div className="space-y-2">
													<p className="text-muted-foreground">
														{event.availableSpots} of {event.capacity} spots available
													</p>
													<div className="w-full bg-muted rounded-full h-2.5 overflow-hidden">
														<div
															className="h-full bg-gradient-to-r from-primary to-accent rounded-full transition-all duration-500"
															style={{ width: `${((event.capacity - (event.availableSpots || 0)) / event.capacity) * 100}%` }}
														/>
													</div>
												</div>
											</div>
										</div>
									)}

									{event.organizer && (
										<div className="p-4 rounded-xl bg-gradient-to-r from-chart-2/5 to-transparent">
											<p className="font-semibold text-lg mb-1">Organized by</p>
											<p className="text-muted-foreground">{event.organizer}</p>
										</div>
									)}
								</CardContent>
							</Card>
						</div>

						{/* Map Location */}
						{event.location && (
							<div className="space-y-4">
								<h3 className="text-2xl font-bold bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">
									Event Location
								</h3>
								<div className="rounded-2xl overflow-hidden">
									<MapLocation
										address={event.location}
										mapUrl={getMapUrl()}
										title={event.venue || `${event.title} Venue`}
										description={`Join us at ${event.location} for ${event.title}`}
										eventDate={new Date(event.date).toLocaleDateString('en-US', {
											weekday: 'long',
											year: 'numeric',
											month: 'long',
											day: 'numeric',
											hour: event.startTime ? 'numeric' : undefined,
											minute: event.startTime ? '2-digit' : undefined
										})}
										phone={event.contactPhone}
										email={event.contactEmail}
										website={event.venueWebsite}
										hours={event.venueHours}
										additionalInfo={[
											event.parking && `Parking: ${event.parking}`,
											event.accessibility && `Accessibility: ${event.accessibility}`,
											event.publicTransport && `Public Transport: ${event.publicTransport}`,
											event.specialInstructions
										].filter(Boolean) as string[]}
									/>
								</div>
							</div>
						)}

						{/* Agenda */}
						{event.agenda && event.agenda.length > 0 && (
							<Card className="shadow-neumorph hover:shadow-neumorph-hover transition-all">
								<CardHeader className="bg-gradient-to-r from-secondary/5 to-accent/5 rounded-t-xl">
									<CardTitle className="text-2xl bg-gradient-to-r from-secondary to-accent bg-clip-text text-transparent">
										Event Schedule
									</CardTitle>
								</CardHeader>
								<CardContent className="pt-8">
									<div className="relative">
										{/* Timeline line */}
										<div className="absolute left-8 top-0 bottom-0 w-0.5 bg-gradient-to-b from-primary via-secondary to-accent"></div>

										<div className="space-y-6">
											{event.agenda.map((item, index) => (
												<div key={index} className="relative flex gap-6 group">
													{/* Timeline dot */}
													<div className="absolute left-6 w-4 h-4 bg-gradient-to-br from-primary to-secondary rounded-full border-4 border-background shadow-lg group-hover:scale-125 transition-transform" />

													<div className="flex gap-4 ml-16 p-4 rounded-xl bg-gradient-to-r from-muted/50 to-transparent hover:from-muted transition-all flex-1">
														<div className="flex items-center gap-2 min-w-[120px]">
															<Clock className="h-4 w-4 text-primary" />
															<span className="font-semibold text-primary">{item.time}</span>
														</div>
														<div className="flex-1">
															<p className="text-foreground/90">{item.activity}</p>
														</div>
													</div>
												</div>
											))}
										</div>
									</div>
								</CardContent>
							</Card>
						)}

						{/* What's Included */}
						{event.includes && event.includes.length > 0 && (
							<Card className="shadow-neumorph hover:shadow-neumorph-hover transition-all">
								<CardHeader className="bg-gradient-to-r from-accent/5 to-chart-1/5 rounded-t-xl">
									<CardTitle className="text-2xl bg-gradient-to-r from-accent to-chart-1 bg-clip-text text-transparent">
										What's Included
									</CardTitle>
								</CardHeader>
								<CardContent className="pt-8">
									<div className="grid sm:grid-cols-2 gap-4">
										{event.includes.map((item, index) => (
											<div
												key={index}
												className="flex items-center gap-3 p-3 rounded-xl bg-gradient-to-r from-accent/5 to-transparent hover:from-accent/10 transition-all"
											>
												<div className="w-2 h-2 bg-gradient-to-br from-accent to-chart-1 rounded-full shadow-lg" />
												<span className="text-foreground/90">{item}</span>
											</div>
										))}
									</div>
								</CardContent>
							</Card>
						)}
					</div>

					{/* Right Column - Booking */}
					<div className="space-y-4">
						{/* Tiered Pricing */}
						{hasTiers && (event as EventWithTiers).priceTiers && (event as EventWithTiers).priceTiers.length > 0 ? (
							<div className="space-y-4">
								<Card className="shadow-lg border-primary/20">
									<CardHeader>
										<CardTitle>Select Tickets</CardTitle>
									</CardHeader>
									<CardContent className="pt-6">
										<TieredPricing
											tiers={(event as EventWithTiers).priceTiers}
											onSelectTier={(tier, qty) => {
												handleTierSelection(tier);
												handleSelectTier(tier, qty);
											}}
										/>
									</CardContent>
								</Card>

								{/* Add to Cart for Tiered Events */}
								{selectedTier && (
									<Card className="shadow-lg border-primary/20">
										<CardHeader>
											<CardTitle>Add to Cart</CardTitle>
										</CardHeader>
										<CardContent className="space-y-4 pt-6">
											<div className="text-lg">
												<span className="font-semibold">{selectedTier.name}</span>
												<span className="text-muted-foreground"> - ${selectedTier.amount}</span>
											</div>

											{/* Quantity Selector */}
											<div className="space-y-2">
												<label className="text-sm font-medium">Number of Tickets</label>
												<div className="flex items-center gap-2">
													<Button
														variant="outline"
														size="icon"
														className="h-8 w-8"
														onClick={() => setQuantity(Math.max(1, quantity - 1))}
														disabled={quantity <= 1}
													>
														<Minus className="h-3 w-3" />
													</Button>
													<Input
														type="number"
														value={quantity}
														onChange={(e) => setQuantity(Math.max(1, Math.min(selectedTier.maxQuantity || 10, parseInt(e.target.value) || 1)))}
														className="w-16 h-8 text-center"
														min="1"
														max={selectedTier.maxQuantity || 10}
													/>
													<Button
														variant="outline"
														size="icon"
														className="h-8 w-8"
														onClick={() => setQuantity(Math.min(selectedTier.maxQuantity || 10, quantity + 1))}
														disabled={quantity >= (selectedTier.maxQuantity || 10)}
													>
														<Plus className="h-3 w-3" />
													</Button>
												</div>
											</div>

											<Button
												className="w-full bg-gradient-to-r from-primary to-primary/80 hover:from-primary/90 hover:to-primary/70"
												size="lg"
												onClick={handleAddToCart}
											>
												{selectedTier.requiresVehicleSubmission ? 'Submit Vehicle for Review' : `Add to Cart - $${(selectedTier.amount * quantity).toFixed(2)}`}
											</Button>
										</CardContent>
									</Card>
								)}
							</div>
						) : (
							/* Single Price Booking */
							<div className="sticky top-24">
								<div className="relative group">
									<div className="absolute -inset-0.5 bg-gradient-to-r from-primary via-secondary to-accent rounded-2xl blur opacity-40 group-hover:opacity-60 transition duration-1000"></div>
									<Card className="relative shadow-neumorph">
										<CardHeader className="bg-gradient-to-r from-primary/10 via-secondary/10 to-accent/10 rounded-t-xl">
											<CardTitle className="text-xl">Book Your Spot</CardTitle>
										</CardHeader>
										<CardContent className="space-y-6 pt-8">
											{/* Price Display */}
											<div className="text-center p-6 rounded-2xl bg-gradient-to-br from-primary/5 via-secondary/5 to-accent/5">
												<div className="text-4xl font-bold bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">
													${event.price.toFixed(2)}
												</div>
												<span className="text-sm text-muted-foreground">per ticket</span>
											</div>

											{/* Enhanced Quantity Selector */}
											<div className="space-y-3">
												<label className="text-sm font-semibold text-muted-foreground">Number of Tickets</label>
												<div className="flex items-center gap-3 p-2 rounded-xl bg-muted/50">
													<Button
														variant="outline"
														size="icon"
														className="h-10 w-10 rounded-xl border-2 hover:bg-primary hover:text-primary-foreground transition-all"
														onClick={() => setQuantity(Math.max(1, quantity - 1))}
														disabled={quantity <= 1}
													>
														<Minus className="h-4 w-4" />
													</Button>
													<div className="flex-1 text-center">
														<Input
															type="number"
															value={quantity}
															onChange={(e) => setQuantity(Math.max(1, Math.min(event.availableSpots || 10, parseInt(e.target.value) || 1)))}
															className="w-full h-10 text-center text-lg font-bold bg-background border-2 rounded-xl"
															min="1"
															max={event.availableSpots || 10}
														/>
													</div>
													<Button
														variant="outline"
														size="icon"
														className="h-10 w-10 rounded-xl border-2 hover:bg-primary hover:text-primary-foreground transition-all"
														onClick={() => setQuantity(Math.min(event.availableSpots || 10, quantity + 1))}
														disabled={quantity >= (event.availableSpots || 10)}
													>
														<Plus className="h-4 w-4" />
													</Button>
												</div>
											</div>

											<Separator className="bg-gradient-to-r from-transparent via-border to-transparent" />

											{/* Price Breakdown */}
											<div className="space-y-3 p-4 rounded-xl bg-muted/30">
												<div className="flex justify-between text-sm">
													<span className="text-muted-foreground">Ticket Price</span>
													<span className="font-medium">${event.price.toFixed(2)} × {quantity}</span>
												</div>
												<div className="flex justify-between text-sm">
													<span className="text-muted-foreground">Processing Fee</span>
													<span className="font-medium text-green-600">FREE</span>
												</div>
												<Separator />
												<div className="flex justify-between text-lg font-bold">
													<span>Total</span>
													<span className="text-2xl bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">
														${(event.price * quantity).toFixed(2)}
													</span>
												</div>
											</div>

											{/* Action Buttons */}
											<div className="space-y-3">
												<Button
													className="w-full h-14 text-lg font-bold bg-gradient-to-r from-primary via-secondary to-accent hover:opacity-90 transition-all shadow-lg animate-gradient bg-300%"
													onClick={handleSingleCheckout}
													disabled={event.status === 'soldout' || event.availableSpots === 0}
												>
													{singlePriceRequiresVehicle ? (
														<>
															<Car className="mr-2 h-5 w-5" />
															Submit Vehicle for Review
														</>
													) : (
														event.status === 'soldout' ? 'Sold Out' : 'Book Now'
													)}
												</Button>

												{!singlePriceRequiresVehicle && (
													<Button
														className="w-full h-12 font-semibold shadow-neumorph-button hover:shadow-neumorph-button-hover transition-all"
														variant="outline"
														onClick={handleAddToCart}
														disabled={event.status === 'soldout' || event.availableSpots === 0}
													>
														Add to Cart
													</Button>
												)}

												<div className="flex gap-3">
													<Button
														variant="outline"
														className="flex-1 h-12 shadow-neumorph-button hover:shadow-neumorph-button-hover hover:bg-destructive/10 hover:text-destructive transition-all group"
													>
														<Heart className="h-5 w-5 group-hover:fill-current transition-all" />
													</Button>
													<Button
														variant="outline"
														className="flex-1 h-12 shadow-neumorph-button hover:shadow-neumorph-button-hover hover:bg-primary/10 hover:text-primary transition-all"
														onClick={handleShare}
													>
														<Share2 className="h-5 w-5" />
													</Button>
												</div>
											</div>

											{/* Urgency Indicator */}
											{event.availableSpots && event.availableSpots < 10 && (
												<div className="p-3 rounded-xl bg-gradient-to-r from-destructive/10 to-chart-5/10 border border-destructive/20">
													<p className="text-sm text-center font-semibold text-destructive animate-pulse">
														🔥 Only {event.availableSpots} tickets remaining!
													</p>
												</div>
											)}

											{/* Trust Badge */}
											<div className="text-center space-y-2 pt-2">
												<p className="text-xs text-muted-foreground">
													✓ Free cancellation up to 48 hours before
												</p>
												<p className="text-xs text-muted-foreground">
													✓ Instant confirmation
												</p>
												<p className="text-xs text-muted-foreground">
													✓ Secure payment
												</p>
											</div>
										</CardContent>
									</Card>
								</div>
							</div>
						)}
					</div>
				</div>
			</div>

			{/* Vehicle Submission Dialog */}
			<Dialog open={showSubmissionForm} onOpenChange={setShowSubmissionForm}>
				<DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
					<VehicleSubmissionForm
						eventId={event.id}
						eventSlug={event.slug}
						eventName={event.title}
						ticketTier={selectedTier?.name}
						ticketPrice={selectedTier?.amount || event.price}
						ticketQuantity={quantity}
						onSuccess={handleSubmissionSuccess}
						onCancel={() => setShowSubmissionForm(false)}
					/>
				</DialogContent>
			</Dialog>
		</div>
	);
}
