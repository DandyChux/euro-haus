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
import { Image } from '~/components/ui/image';
import { ticketService } from '~/lib/services/ticket-service';
import { AttendeeTabs } from '~/components/attendee-tabs';

export const Route = createFileRoute('/admin/event-details')({
	validateSearch: z.object({
		slug: z.string()
	}),
	loaderDeps: ({ search: { slug } }) => ({
		slug: slug
	}),
	loader: async ({ deps: { slug } }) => {
		// Fetch event details
		const event = await stripeService.getEventWithPriceTiers(slug);

		if (!event) {
			throw new Error("Event not found");
		}

		const attendees = await ticketService.getEventAttendees(event.id);

		return { event, attendees };
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
	const { event, attendees } = Route.useLoaderData();
	const navigate = useNavigate();
	const [activeTab, setActiveTab] = useState('overview');

	// Parse event date safely
	const eventDate = event.date ? new Date(event.date) : null;

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
						<h1 className="text-3xl font-bold">{event.title}</h1>
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
								{event.location || 'Not set'}
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
								{event.capacity || 'Unlimited'}
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
								{event.priceTiers?.length > 0 ? (
									<>
										${Math.min(...event.priceTiers.map((t) => t.amount)).toFixed(2)} -
										${Math.max(...event.priceTiers.map((t) => t.amount)).toFixed(2)}
									</>
								) : (
									event.price ?
										`$${(event.price).toFixed(2)}` :
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

								{event.priceTiers?.length > 0 && (
									<div>
										<label className="text-sm font-medium">Ticket Tiers</label>
										<div className="space-y-2 mt-2">
											{event.priceTiers.map((tier) => (
												<div key={tier.id} className="flex justify-between items-center p-3 bg-muted rounded">
													<div>
														<div className="font-medium">{tier.name}</div>
														{tier.description && (
															<div className="text-sm text-muted-foreground">
																{tier.description}
															</div>
														)}
													</div>
													<div className="font-semibold">
														${(tier.amount).toFixed(2)}
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
						eventName={event.title}
						tiers={event.priceTiers.map((t) => ({
							id: t.id,
							priceId: t.id,
							name: t.name || 'Unnamed Tier',
							amount: t.amount,
							currency: t.currency
						}))}
					/>
				</TabsContent>

				<TabsContent value="attendees">
					<Card>
						<CardHeader>
							<CardTitle>Attendee Management</CardTitle>
						</CardHeader>
						<CardContent>
							{/*{attendees?.map((attendee) => (
								<div key={attendee.id} className="flex justify-between items-center p-3 bg-muted rounded">
									<div>
										<span className="font-medium">{attendee.attendeeEmail}</span>
										<span className="text-sm text-muted-foreground">
											{attendee.attendeeName}
										</span>
									</div>
								</div>
							))}*/}
							<AttendeeTabs
								attendees={attendees}
							/>
						</CardContent>
					</Card>
				</TabsContent>

				<TabsContent value="settings">
					{/* TODO: Add event settings here */}
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
