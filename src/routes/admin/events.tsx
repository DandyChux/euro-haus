import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '~/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { toast } from 'sonner';
import { stripeService } from '~/lib/services/stripe-service';
import { ticketService } from '~/lib/services/ticket-service';
import { ProtectedRoute } from '~/components/protected-route';
import {
	Calendar,
	Users,
	QrCode,
	Search,
	Check,
	X,
	RefreshCcw,
	Camera,
	Keyboard,
	Clock,
	MapPin,
	Ticket as TicketIcon
} from 'lucide-react';
import { format, isAfter } from 'date-fns';
import { Scanner } from '@yudiel/react-qr-scanner';

export const Route = createFileRoute('/admin/events')({
	loader: async () => {
		const events = await stripeService.getAllEvents(true);
		return { events };
	},
	component: AdminEventsPage,
});

function AdminEventsPage() {
	return (
		<ProtectedRoute>
			<AdminEventsContent />
		</ProtectedRoute>
	);
}

function AdminEventsContent() {
	const { events } = Route.useLoaderData();
	const [selectedTab, setSelectedTab] = useState('upcoming');

	// Separate events by status
	const now = new Date();
	const endOfToday = new Date(now);
	endOfToday.setHours(0, 0, 0, 999);

	// const upcomingEvents = events.filter(event =>
	// 	isAfter(new Date(event.date), endOfToday) && event.status !== 'cancelled'
	// );
	const upcomingEvents = events.filter(event => {
		console.log(new Date(event.date));
		return isAfter(new Date(event.date), endOfToday) && event.status !== 'cancelled';
	});
	// const pastEvents = events.filter(event =>
	// 	!isAfter(new Date(event.date), endOfToday) || event.status === 'cancelled'
	// );
	const pastEvents = events.filter(event => {
		return !isAfter(new Date(event.date), endOfToday) || event.status === 'cancelled';
	});

	return (
		<div className="p-6 space-y-6 min-h-screen">
			<div>
				<h1 className="text-3xl font-bold">Event Management</h1>
				<p className="text-muted-foreground">Manage events and check in attendees</p>
			</div>

			<Tabs value={selectedTab} onValueChange={setSelectedTab}>
				<TabsList>
					<TabsTrigger value="upcoming">
						Upcoming Events ({upcomingEvents.length})
					</TabsTrigger>
					<TabsTrigger value="past">
						Past Events ({pastEvents.length})
					</TabsTrigger>
				</TabsList>

				<TabsContent value="upcoming" className="space-y-4">
					{upcomingEvents.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<Calendar className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
								<p className="text-muted-foreground">No upcoming events</p>
							</CardContent>
						</Card>
					) : (
						upcomingEvents.map(event => (
							<EventCard key={event.id} event={event} />
						))
					)}
				</TabsContent>

				<TabsContent value="past" className="space-y-4">
					{pastEvents.length === 0 ? (
						<Card>
							<CardContent className="text-center py-8">
								<Calendar className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
								<p className="text-muted-foreground">No past events</p>
							</CardContent>
						</Card>
					) : (
						pastEvents.map(event => (
							<EventCard key={event.id} event={event} isPast />
						))
					)}
				</TabsContent>
			</Tabs>
		</div>
	);
}

interface EventCardProps {
	event: any;
	isPast?: boolean;
}

function EventCard({ event, isPast }: EventCardProps) {
	const [showCheckIn, setShowCheckIn] = useState(false);
	const navigate = useNavigate();

	return (
		<Card>
			<CardHeader>
				<div className="flex justify-between items-start">
					<div>
						<CardTitle>{event.title}</CardTitle>
						<CardDescription className="space-y-1 mt-2">
							<div className="flex items-center gap-2">
								<Calendar className="w-4 h-4" />
								{format(new Date(event.date), 'PPP')}
							</div>
							<div className="flex items-center gap-2">
								<MapPin className="w-4 h-4" />
								{event.location}
							</div>
							{event.capacity && (
								<div className="flex items-center gap-2">
									<Users className="w-4 h-4" />
									Capacity: {event.capacity}
								</div>
							)}
						</CardDescription>
					</div>
					<div className="flex gap-2">
						<Button
							variant="outline"
							size="sm"
							onClick={() => navigate({
								to: '/admin/event-details',
								search: { event_id: event.id }
							})}
						>
							Manage Event
						</Button>
						{!isPast && (
							<Button
								onClick={() => setShowCheckIn(!showCheckIn)}
								variant={showCheckIn ? "default" : "outline"}
							>
								<QrCode className="w-4 h-4 mr-2" />
								Check-In
							</Button>
						)}
						<Badge variant={isPast ? "secondary" : "default"}>
							{isPast ? "Past" : "Upcoming"}
						</Badge>
					</div>
				</div>
			</CardHeader>

			{showCheckIn && (
				<CardContent>
					<EventCheckIn eventId={event.id} eventName={event.title} />
				</CardContent>
			)}
		</Card>
	);
}

interface EventCheckInProps {
	eventId: string;
	eventName: string;
}

function EventCheckIn({ eventId, eventName }: EventCheckInProps) {
	const [scanMode, setScanMode] = useState<'camera' | 'manual'>('manual');
	const [manualCode, setManualCode] = useState('');
	const [isChecking, setIsChecking] = useState(false);
	const [lastCheckedIn, setLastCheckedIn] = useState<any>(null);
	const [attendees, setAttendees] = useState<any[]>([]);
	const [stats, setStats] = useState({
		total: 0,
		checkedIn: 0,
		remaining: 0,
	});
	const [showAttendees, setShowAttendees] = useState(false);
	const [loading, setLoading] = useState(false);

	// Fetch event statistics and attendees
	const fetchEventData = async () => {
		setLoading(true);
		try {
			const tickets = await ticketService.getEventAttendees(eventId);
			const checkedInCount = tickets.filter(t => t.checkedIn).length;

			setAttendees(tickets);
			setStats({
				total: tickets.length,
				checkedIn: checkedInCount,
				remaining: tickets.length - checkedInCount,
			});
		} catch (error) {
			console.error('Failed to fetch event data:', error);
			toast.error('Failed to load event data');
		} finally {
			setLoading(false);
		}
	};

	// Load data on mount
	useState(() => {
		fetchEventData();
	});

	const handleScan = (result: any) => {
		if (result && !isChecking) {
			const text = result.text || result;
			checkInTicket(text);
		}
	};

	const handleManualSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		if (manualCode.trim()) {
			await checkInTicket(manualCode.trim());
			setManualCode('');
		}
	};

	const checkInTicket = async (ticketCode: string) => {
		setIsChecking(true);

		try {
			// Validate the ticket
			const validationResponse = await ticketService.validateTicket(ticketCode);

			if (!validationResponse) {
				toast.error('Invalid ticket code');
				setLastCheckedIn({ error: 'Invalid ticket code' });
				return;
			}

			// Check if it's for the correct event
			if (validationResponse.eventId !== eventId) {
				toast.error('This ticket is for a different event');
				setLastCheckedIn({
					error: 'Wrong event',
					ticketEvent: validationResponse.eventName
				});
				return;
			}

			if (validationResponse.checkedIn) {
				toast.warning(`Already checked in at ${format(new Date(validationResponse.checkedInAt!), 'PPp')}`);
				setLastCheckedIn({
					...validationResponse,
					alreadyCheckedIn: true
				});
				return;
			}

			// Check in the ticket
			const checkedInTicket = await ticketService.checkInTicket(ticketCode);

			toast.success(`Checked in: ${checkedInTicket.attendeeName}`);
			setLastCheckedIn(checkedInTicket);

			// Refresh stats
			await fetchEventData();

		} catch (error) {
			console.error('Check-in error:', error);
			toast.error('Failed to check in ticket');
			setLastCheckedIn({ error: 'Check-in failed' });
		} finally {
			setIsChecking(false);
		}
	};

	const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
		if (e.key === 'Enter' && !isChecking) {
			e.preventDefault();
			handleManualSubmit(e as any);
		}
	};

	return (
		<div className="space-y-6">
			{/* Statistics */}
			<div className="grid grid-cols-3 gap-4">
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium">Total Tickets</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="text-2xl font-bold">{stats.total}</div>
					</CardContent>
				</Card>
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium">Checked In</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="text-2xl font-bold text-green-600">{stats.checkedIn}</div>
						{stats.total > 0 && (
							<p className="text-xs text-muted-foreground">
								{Math.round((stats.checkedIn / stats.total) * 100)}%
							</p>
						)}
					</CardContent>
				</Card>
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium">Remaining</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="text-2xl font-bold text-blue-600">{stats.remaining}</div>
					</CardContent>
				</Card>
			</div>

			{/* Mode Toggle */}
			<div className="flex gap-2">
				<Button
					variant={scanMode === 'camera' ? 'default' : 'outline'}
					onClick={() => setScanMode('camera')}
					className="flex-1"
				>
					<Camera className="w-4 h-4 mr-2" />
					Scan QR Code
				</Button>
				<Button
					variant={scanMode === 'manual' ? 'default' : 'outline'}
					onClick={() => setScanMode('manual')}
					className="flex-1"
				>
					<Keyboard className="w-4 h-4 mr-2" />
					Enter Code
				</Button>
			</div>

			{/* Scanner/Input */}
			<Card>
				<CardContent className="p-6">
					{scanMode === 'camera' ? (
						<div className="max-w-md mx-auto">
							<Scanner
								scanDelay={300}
								onError={(err: any) => {
									console.error('Scanner error:', err);
									toast.error('Scanner error. Please try manual entry.');
								}}
								onScan={handleScan}
								styles={{
									container: {
										width: '100%'
									}
								}}
								constraints={{
									facingMode: 'environment'
								}}
							/>
							<p className="text-sm text-muted-foreground text-center mt-4">
								Position the QR code within the camera view
							</p>
						</div>
					) : (
						<form onSubmit={handleManualSubmit} className="flex gap-2">
							<Input
								value={manualCode}
								onChange={(e) => setManualCode(e.target.value)}
								onKeyDown={handleKeyDown}
								placeholder="Enter ticket code..."
								className="flex-1"
								disabled={isChecking}
								autoFocus
							/>
							<Button type="submit" disabled={isChecking || !manualCode.trim()}>
								{isChecking ? (
									<RefreshCcw className="w-4 h-4 mr-2 animate-spin" />
								) : (
									<Search className="w-4 h-4 mr-2" />
								)}
								Check In
							</Button>
						</form>
					)}
				</CardContent>
			</Card>

			{/* Last Check-in Result */}
			{lastCheckedIn && (
				<Alert className={
					lastCheckedIn.error ? 'border-red-500' :
						lastCheckedIn.alreadyCheckedIn ? 'border-yellow-500' :
							'border-green-500'
				}>
					<div className="flex items-start gap-2">
						{lastCheckedIn.error ? (
							<X className="w-5 h-5 text-red-500" />
						) : lastCheckedIn.alreadyCheckedIn ? (
							<Clock className="w-5 h-5 text-yellow-500" />
						) : (
							<Check className="w-5 h-5 text-green-500" />
						)}
						<div className="flex-1">
							<AlertDescription>
								{lastCheckedIn.error ? (
									<div className="font-semibold text-red-600">
										{lastCheckedIn.error}
										{lastCheckedIn.ticketEvent && (
											<p className="text-sm mt-1">
												This ticket is for: {lastCheckedIn.ticketEvent}
											</p>
										)}
									</div>
								) : (
									<>
										<div className="font-semibold">{lastCheckedIn.attendeeName}</div>
										<div className="text-sm text-muted-foreground">
											{lastCheckedIn.attendeeEmail}
										</div>
										<div className="text-sm text-muted-foreground mt-1">
											Ticket: {lastCheckedIn.ticketType} • Code: {lastCheckedIn.ticketCode}
										</div>
										{lastCheckedIn.alreadyCheckedIn && (
											<div className="text-sm text-yellow-600 mt-1">
												Already checked in at {format(new Date(lastCheckedIn.checkedInAt), 'PPp')}
											</div>
										)}
									</>
								)}
							</AlertDescription>
						</div>
					</div>
				</Alert>
			)}

			{/* Attendee List Toggle */}
			<Button
				variant="outline"
				onClick={() => setShowAttendees(!showAttendees)}
				className="w-full"
			>
				<Users className="w-4 h-4 mr-2" />
				{showAttendees ? 'Hide' : 'Show'} Attendee List ({attendees.length})
			</Button>

			{/* Attendee List */}
			{showAttendees && (
				<Card>
					<CardHeader>
						<div className="flex justify-between items-center">
							<CardTitle>Attendees</CardTitle>
							<Button
								size="sm"
								variant="ghost"
								onClick={fetchEventData}
								disabled={loading}
							>
								<RefreshCcw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
							</Button>
						</div>
					</CardHeader>
					<CardContent>
						<div className="max-h-96 overflow-y-auto space-y-2">
							{attendees.length === 0 ? (
								<p className="text-center text-muted-foreground py-4">
									No tickets sold yet
								</p>
							) : (
								attendees.map((ticket) => (
									<div
										key={ticket.id}
										className="flex justify-between items-center p-3 border rounded-lg"
									>
										<div>
											<div className="font-medium">{ticket.attendeeName}</div>
											<div className="text-sm text-muted-foreground">
												{ticket.attendeeEmail}
											</div>
											<div className="text-xs text-muted-foreground mt-1">
												{ticket.ticketType} • {ticket.ticketCode}
											</div>
										</div>
										<div className="text-right">
											<Badge
												variant={ticket.checkedIn ? "default" : "outline"}
												className={ticket.checkedIn ? "bg-green-100 text-green-800" : ""}
											>
												{ticket.checkedIn ? "Checked In" : "Not Checked In"}
											</Badge>
											{ticket.checkedInAt && (
												<div className="text-xs text-muted-foreground mt-1">
													{format(new Date(ticket.checkedInAt), 'pp')}
												</div>
											)}
										</div>
									</div>
								))
							)}
						</div>
					</CardContent>
				</Card>
			)}
		</div>
	);
}
