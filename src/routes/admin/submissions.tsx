import { createFileRoute } from '@tanstack/react-router';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs';
import { SubmissionReview } from '~/components/admin/submission-review';
import { stripeService } from '~/lib/services/stripe-service';
import { useState } from 'react';
import { Card, CardContent } from '~/components/ui/card';
import { Calendar } from 'lucide-react';
import { isAfter } from 'date-fns';

export const Route = createFileRoute('/admin/submissions')({
	loader: async () => {
		const events = await stripeService.getAllEvents();
		// Filter only upcoming events that might have submissions
		const now = new Date();
		const endOfToday = new Date(now);
		endOfToday.setHours(0, 0, 0, 999);
		const upcomingEvents = events.filter(event =>
			isAfter(new Date(event.date), endOfToday) && event.status !== 'cancelled'
		);
		return { events: upcomingEvents };
	},
	component: AdminSubmissionsPage,
});

function AdminSubmissionsPage() {
	const { events } = Route.useLoaderData();
	const [selectedEvent, setSelectedEvent] = useState(events[0]?.id || '');

	if (events.length === 0) {
		return (
			<div className="p-6">
				<Card>
					<CardContent className="text-center py-8">
						<Calendar className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
						<p className="text-muted-foreground">No upcoming events found</p>
					</CardContent>
				</Card>
			</div>
		);
	}

	return (
		<div className="p-6 space-y-6 min-h-screen">
			<div>
				<h1 className="text-3xl font-bold">Vehicle Submissions</h1>
				<p className="text-muted-foreground">Review and manage participant vehicle submissions</p>
			</div>

			<Tabs value={selectedEvent} onValueChange={setSelectedEvent}>
				<TabsList className="grid w-full" style={{ gridTemplateColumns: `repeat(${Math.min(events.length, 4)}, 1fr)` }}>
					{events.map((event) => (
						<TabsTrigger key={event.id} value={event.id}>
							{event.title}
						</TabsTrigger>
					))}
				</TabsList>

				{events.map((event) => (
					<TabsContent key={event.id} value={event.id}>
						<SubmissionReview eventId={event.id} eventName={event.title} />
					</TabsContent>
				))}
			</Tabs>
		</div>
	);
}
