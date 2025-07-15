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
import { cn } from '~/lib/utils';
import { Link } from '@tanstack/react-router';
import { ChevronRight } from 'lucide-react';
import { stripeService } from '~/lib/services/stripe-service';
import type { EventProduct as Event } from '~/lib/services/stripe-service';

type EventCardProps = {
	event: Event;
	index: number;
	className?: string;
}

// Loading skeleton component for event cards
function EventCardSkeleton({ index }: { index: number }) {
	return (
		<Card className={cn('relative rounded-none p-4 border-none flex flex-col', {
			'bg-secondary/10': index % 2 === 0,
		})}>
			<CardHeader className="px-2 gap-0">
				<Skeleton className="h-8 w-3/4 mb-2" />
				<div className="space-y-2">
					<Skeleton className="h-4 w-1/2" />
					<Skeleton className="h-4 w-full" />
					<Skeleton className="h-4 w-2/3" />
					<Skeleton className="h-4 w-1/3" />
				</div>
			</CardHeader>
			<CardContent className="p-0 flex-1 relative w-full aspect-[16/9] overflow-hidden">
				<Skeleton className="absolute w-full h-full" />
			</CardContent>
			<CardFooter className="mt-4">
				<Skeleton className="h-10 w-full" />
			</CardFooter>
		</Card>
	);
}

export default function EventCards() {
	const [events, setEvents] = useState<Event[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		async function fetchEvents() {
			try {
				setLoading(true);
				setError(null);

				// Fetch featured events from Stripe
				const featuredEvents = await stripeService.getFeaturedEvents();
				setEvents(featuredEvents);
			} catch (err) {
				console.error('Failed to fetch events:', err);
				setError('Failed to load events');
				setEvents([]);
			} finally {
				setLoading(false);
			}
		}

		fetchEvents();
	}, []);

	// Loading state
	if (loading) {
		return (
			<div className='grid md:grid-cols-2 lg:grid-cols-3 mx-auto gap-4 p-2 md:p-8'>
				{[0, 1, 2].map((index) => (
					<EventCardSkeleton key={index} index={index} />
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

	// No events state
	if (events.length === 0) {
		return (
			<div className='flex items-center justify-center p-8'>
				<div className='text-center'>
					<p className='text-lg text-muted-foreground mb-4'>
						No upcoming events at this time.
					</p>
					<Button asChild>
						<Link to="/events">View All Events</Link>
					</Button>
				</div>
			</div>
		);
	}

	return (
		<div className='grid md:grid-cols-2 lg:grid-cols-3 mx-auto gap-4 p-2 md:p-8'>
			{events.map((event, index) => (
				<EventCard key={event.slug} event={event} index={index} />
			))}
		</div>
	);
}

const EventCard: React.FC<EventCardProps> = ({
	event,
	index,
	className
}) => {
	// Calculate the price display
	const getPriceDisplay = () => {
		// Use the new hasTiers and lowestPrice fields
		if (event.hasTiers && event.lowestPrice) {
			return `From $${event.lowestPrice.toFixed(2)} USD`;
		}
		// Regular single price
		return `$${event.price.toFixed(2)} USD`;
	};

	return (
		<Card className={cn('relative group rounded-none p-4 border-none flex flex-col', {
			'bg-secondary/10 text-foreground': index % 2 === 0,
		}, className)}>
			<CardHeader
				className={cn('px-2 gap-0', {
					'order-2': index % 2 !== 0,
					'order-1': index % 2 === 0,
				})}
			>
				<CardTitle
					className={cn('font-normal text-xl lg:text-3xl tracking-wide my-2', {
						'order-1': index % 2 === 0,
						'order-2': index % 2 !== 0,
					})}
				>
					{event.title}
				</CardTitle>
				<CardDescription
					className={cn('font-normal text-sm lg:text-base tracking-wide my-2 flex flex-col', {
						'order-1': index % 2 !== 0,
						'order-2': index % 2 === 0,
					})}
				>
					<span>{new Date(event.date).toLocaleDateString('en-US', {
						year: 'numeric',
						month: 'long',
						day: 'numeric'
					})}</span>
					<span className="line-clamp-3">{event.description}</span>
					<span className="font-medium text-primary">{getPriceDisplay()}</span>
				</CardDescription>
			</CardHeader>
			<CardContent
				className={cn('p-0 flex-1 relative w-full aspect-[16/9] overflow-hidden', {
					'order-1': index % 2 !== 0,
					'order-2': index % 2 === 0,
				})}
			>
				<Image
					src={event.imageUrl}
					alt={event.title}
					className='absolute object-cover w-full h-full group-hover:scale-105 transition-transform duration-300'
				/>
			</CardContent>
			<CardFooter className='order-3'>
				<Button
					asChild
					className='mt-4 group w-full flex items-center gap-1'
					disabled={event.status === 'soldout' || event.status === 'cancelled'}
				>
					<Link to="/events/$slug" params={{ slug: event.slug }}>
						{event.status === 'soldout' ? 'Sold Out' : 'View Details'}
						<ChevronRight className='h-4 w-4 group-hover:translate-x-1 transition-transform' />
					</Link>
				</Button>
			</CardFooter>
		</Card>
	);
};
