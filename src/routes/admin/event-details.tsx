import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs';
import { EventProductsManager } from '~/components/admin/event-products-manager';
import { ProtectedRoute } from '~/components/protected-route';
import { stripeService } from '~/lib/services/stripe-service';
import { ArrowLeft, Calendar, MapPin, Users, DollarSign } from 'lucide-react';
import { format } from 'date-fns';
import z from 'zod';
import { apiClient } from '~/lib/api';
import { Image } from '~/components/ui/image';

export const Route = createFileRoute('/admin/event-details')({
	validateSearch: z.object({
		event_id: z.string()
	}),
	loaderDeps: ({ search: { event_id } }) => ({
		eventId: event_id
	}),
	loader: async ({ deps: { eventId } }) => {
		// Fetch event details
		const eventResponse = await apiClient.get(`/products/${eventId}`);
		const event = eventResponse.data;

		// Fetch event tiers if applicable
		let tiers = [];
		if (event.metadata?.has_tiers === 'true') {
			const pricesResponse = await apiClient.get(`/products/${eventId}/prices`);
			tiers = pricesResponse.data.prices?.filter((p: any) => p.nickname) || [];
		}

		return { event, tiers };
	},
	component: EventDetailsPage,
});

function EventDetailsPage() {
	return (
		<ProtectedRoute>
			<EventDetailsContent />
		</ProtectedRoute>
	);
}

function EventDetailsContent() {
	const { event, tiers } = Route.useLoaderData();
	const navigate = useNavigate();
	const [activeTab, setActiveTab] = useState('overview');

	// Parse event date safely
	const eventDate = event.metadata?.event_date ? new Date(event.metadata.event_date) : null;

	return (
		<div className="p-6 space-y-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-4">
					<Button
						variant="ghost"
						size="icon"
						onClick={() => navigate({ to: '/admin/events' })}
					>
						<ArrowLeft className="w-4 h-4" />
					</Button>
					<div>
						<h1 className="text-3xl font-bold">{event.name}</h1>
						<p className="text-muted-foreground">Event Management</p>
					</div>
				</div>
			</div>

			{/* Event Stats */}
			<div className="grid grid-cols-1 md:grid-cols-4 gap-4">
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium text-muted-foreground">Date</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="flex items-center gap-2">
							<Calendar className="w-4 h-4" />
							<span className="font-semibold">
								{eventDate ? format(eventDate, 'PPP') : 'Not set'}
							</span>
						</div>
					</CardContent>
				</Card>

				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium text-muted-foreground">Location</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="flex items-center gap-2">
							<MapPin className="w-4 h-4" />
							<span className="font-semibold">
								{event.metadata?.location || 'Not set'}
							</span>
						</div>
					</CardContent>
				</Card>

				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium text-muted-foreground">Capacity</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="flex items-center gap-2">
							<Users className="w-4 h-4" />
							<span className="font-semibold">
								{event.metadata?.capacity || 'Unlimited'}
							</span>
						</div>
					</CardContent>
				</Card>

				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium text-muted-foreground">Price Range</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="flex items-center gap-2">
							<DollarSign className="w-4 h-4" />
							<span className="font-semibold">
								{tiers.length > 0 ? (
									<>
										${Math.min(...tiers.map((t: any) => t.unit_amount / 100)).toFixed(2)} -
										${Math.max(...tiers.map((t: any) => t.unit_amount / 100)).toFixed(2)}
									</>
								) : (
									event.default_price ?
										`$${(event.default_price.unit_amount / 100).toFixed(2)}` :
										'Not set'
								)}
							</span>
						</div>
					</CardContent>
				</Card>
			</div>

			{/* Main Content Tabs */}
			<Tabs value={activeTab} onValueChange={setActiveTab}>
				<TabsList>
					<TabsTrigger value="overview">Overview</TabsTrigger>
					<TabsTrigger value="products">Products & Add-ons</TabsTrigger>
					<TabsTrigger value="attendees">Attendees</TabsTrigger>
					<TabsTrigger value="settings">Settings</TabsTrigger>
				</TabsList>

				<TabsContent value="overview" className="space-y-4">
					<Card>
						<CardHeader>
							<CardTitle>Event Details</CardTitle>
						</CardHeader>
						<CardContent>
							<div className="space-y-4">
								<div>
									<label className="text-sm font-medium">Description</label>
									<p className="text-muted-foreground mt-1">
										{event.description || 'No description provided'}
									</p>
								</div>

								{event.images?.length > 0 && (
									<div>
										<label className="text-sm font-medium">Images</label>
										<div className="grid grid-cols-3 gap-4 mt-2">
											{event.images.map((img: string, idx: number) => (
												<Image
													key={idx}
													src={img}
													alt={`Event ${idx + 1}`}
													className="rounded-lg object-cover w-full h-32"
												/>
											))}
										</div>
									</div>
								)}

								{tiers.length > 0 && (
									<div>
										<label className="text-sm font-medium">Ticket Tiers</label>
										<div className="space-y-2 mt-2">
											{tiers.map((tier: any) => (
												<div key={tier.id} className="flex justify-between items-center p-3 bg-muted rounded">
													<div>
														<div className="font-medium">{tier.nickname}</div>
														{tier.metadata?.description && (
															<div className="text-sm text-muted-foreground">
																{tier.metadata.description}
															</div>
														)}
													</div>
													<div className="font-semibold">
														${(tier.unit_amount / 100).toFixed(2)}
													</div>
												</div>
											))}
										</div>
									</div>
								)}
							</div>
						</CardContent>
					</Card>
				</TabsContent>

				<TabsContent value="products">
					<EventProductsManager
						eventId={event.id}
						eventName={event.name}
						tiers={tiers.map((t: any) => ({
							id: t.id,
							priceId: t.id,
							name: t.nickname || 'Unnamed Tier',
							amount: t.unit_amount / 100,
							currency: t.currency
						}))}
					/>
				</TabsContent>

				<TabsContent value="attendees">
					{/* Add attendee management here */}
					<Card>
						<CardHeader>
							<CardTitle>Attendee Management</CardTitle>
						</CardHeader>
						<CardContent>
							<p className="text-muted-foreground">Attendee management coming soon...</p>
						</CardContent>
					</Card>
				</TabsContent>

				<TabsContent value="settings">
					{/* Add event settings here */}
					<Card>
						<CardHeader>
							<CardTitle>Event Settings</CardTitle>
						</CardHeader>
						<CardContent>
							<p className="text-muted-foreground">Event settings coming soon...</p>
						</CardContent>
					</Card>
				</TabsContent>
			</Tabs>
		</div>
	);
}
