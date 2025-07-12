import { createFileRoute } from '@tanstack/react-router'
import { useState, useEffect, useRef } from 'react'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '~/components/ui/card'
import { Badge } from '~/components/ui/badge'
import { toast } from 'sonner'
import { apiClient } from '~/lib/api'
import { EventProduct as Event } from '~/lib/services/stripe-service'
import { ticketService, Ticket } from '~/lib/services/ticket-service'

interface WebSocketMessage {
	action: string;
	ticket: string;
	customer: string;
	timestamp: string;
}

export const Route = createFileRoute('/events/$slug_/checkin')({
	component: CheckInComponent,
	loader: async ({ params }) => {
		const { slug } = params

		const response = await apiClient.get(`/events/${slug}`)
			.catch(error => {
				console.error('Error loading event details:', error)
				toast.error('Error loading event details', {
					description: error instanceof Error ? error.message : 'Could not load event details'
				})
				throw error
			})

		const event: Event = response.data

		if (!event) {
			throw new Error(`Failed to fetch event: ${response.status}`)
		}

		return {
			event
		}
	}
})

function CheckInComponent() {
	const { slug } = Route.useParams()
	const { event: eventDetails } = Route.useLoaderData()
	const wsRef = useRef<WebSocket | null>(null)

	const [ticketToken, setTicketToken] = useState('')
	const [attendees, setAttendees] = useState<Ticket[]>([])
	const [currentTicket, setCurrentTicket] = useState<Ticket | null>(null)
	const [loading, setLoading] = useState(false)

	// Set up WebSocket when we have event details
	useEffect(() => {
		if (eventDetails?.id) {
			// Load initial attendees
			fetchAttendees()

			// Set up WebSocket for real-time updates
			const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			const wsUrl = `${wsProtocol}//${window.location.host}/api/events/updates?event_id=${eventDetails.id}`

			try {
				const ws = new WebSocket(wsUrl)

				ws.onopen = () => {
					console.log('WebSocket connected')
				}

				ws.onmessage = (event) => {
					try {
						const data = JSON.parse(event.data) as WebSocketMessage
						console.log('WebSocket message:', data)

						if (data.action === 'check_in') {
							// Refresh attendee list when someone checks in
							fetchAttendees()

							toast.info('New Check-in', {
								description: `${data.customer} just checked in`,
							})
						}
					} catch (error) {
						console.error('Error parsing WebSocket message:', error)
					}
				}

				ws.onerror = (error) => {
					console.error('WebSocket error:', error)
				}

				ws.onclose = () => {
					console.log('WebSocket disconnected')
				}

				wsRef.current = ws
			} catch (error) {
				console.error('Error setting up WebSocket:', error)
				toast.error('Connection Error', {
					description: error instanceof Error ? error.message : 'Could not establish real-time connection. Check-ins will still work but may require manual refresh.',
				})
			}
		}

		return () => {
			if (wsRef.current) {
				wsRef.current.close()
			}
		}
	}, [eventDetails?.id])

	const fetchAttendees = async () => {
		if (!eventDetails?.id) return

		try {
			setLoading(true)
			const tickets = await ticketService.getEventAttendees(eventDetails.id)
			setAttendees(tickets || [])
		} catch (error) {
			console.error('Error loading attendees:', error)
			toast.error('Error loading attendees', {
				description: error instanceof Error ? error.message : 'Could not load attendee list'
			})
		} finally {
			setLoading(false)
		}
	}

	const validateTicket = async () => {
		if (!ticketToken.trim()) return

		setLoading(true)
		try {
			// Note: The validateTicket method in ticket-service seems incomplete
			// You might need to update it to accept the ticketCode parameter
			const ticket = await ticketService.validateTicket(ticketToken)

			if (ticket) {
				setCurrentTicket(ticket)
				if (ticket.checkedIn) {
					toast.info('Ticket already checked in', {
						description: `This ticket was checked in at ${formatTime(ticket.checkedInAt!)}`
					})
				}
			} else {
				setCurrentTicket(null)
				toast.error('Invalid ticket', {
					description: "This ticket is not valid"
				})
			}
		} catch (error) {
			console.error('Error validating ticket:', error)
			toast.error('Error validating ticket', {
				description: 'Could not validate ticket'
			})
			setCurrentTicket(null)
		} finally {
			setLoading(false)
		}
	}

	const checkInTicket = async () => {
		if (!currentTicket || currentTicket.checkedIn) return

		setLoading(true)
		try {
			const updatedTicket = await ticketService.checkInTicket(ticketToken)

			setCurrentTicket(updatedTicket)

			toast.success('Ticket checked in successfully!', {
				description: 'Thank you for your participation!'
			})

			// Refresh attendee list
			fetchAttendees()

			// Clear the form after a delay
			setTimeout(() => {
				setTicketToken('')
				setCurrentTicket(null)
			}, 2000)

		} catch (error) {
			console.error('Error checking in ticket:', error)
			toast.error('Error checking in ticket', {
				description: error instanceof Error ? error.message : 'Could not check in ticket'
			})
		} finally {
			setLoading(false)
		}
	}

	const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
		if (e.key === 'Enter') {
			validateTicket()
		}
	}

	// Format a date for display
	const formatDate = (dateString: string) => {
		try {
			return new Date(dateString).toLocaleDateString(undefined, {
				year: 'numeric',
				month: 'long',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
			})
		} catch (_) {
			return dateString
		}
	}

	// Format a time for display
	const formatTime = (dateString: string) => {
		try {
			return new Date(dateString).toLocaleTimeString(undefined, {
				hour: '2-digit',
				minute: '2-digit',
				second: '2-digit',
			})
		} catch (_) {
			return dateString
		}
	}

	// Calculate check-in statistics
	const checkedInCount = attendees.filter(a => a.checkedIn).length
	const totalCount = attendees.length
	const checkedInPercentage = totalCount > 0 ? Math.round((checkedInCount / totalCount) * 100) : 0

	return (
		<div className="py-8 px-4 md:px-8 min-h-screen">
			<div className="flex flex-col mb-16 md:mb-24">
				<h1 className="text-3xl font-bold">Check-in: {eventDetails?.title || slug}</h1>
				{eventDetails && (
					<p className="text-muted-foreground">
						{formatDate(eventDetails.date)} at {eventDetails.location}
					</p>
				)}
			</div>

			<div className="grid grid-cols-1 md:grid-cols-2 gap-8">
				<Card>
					<CardHeader>
						<CardTitle>Check In Tickets</CardTitle>
						<CardDescription>Enter ticket code or scan QR code</CardDescription>
					</CardHeader>
					<CardContent>
						<div className="flex gap-2">
							<Input
								placeholder="Enter ticket code"
								value={ticketToken}
								onChange={(e) => setTicketToken(e.target.value)}
								onKeyDown={handleKeyDown}
								disabled={loading}
							/>
							<Button
								onClick={validateTicket}
								disabled={!ticketToken.trim() || loading}
							>
								{loading ? 'Loading...' : 'Validate'}
							</Button>
						</div>

						{currentTicket && (
							<div className="mt-6 p-4 border rounded-lg">
								<h3 className="text-lg font-medium">{currentTicket.attendeeName}</h3>
								<p className="text-sm text-muted-foreground">{currentTicket.attendeeEmail}</p>
								<div className="flex justify-between items-center mt-2">
									<span>Type: {currentTicket.ticketType}</span>
									{currentTicket.checkedIn ? (
										<Badge className="bg-green-100 text-green-800">
											Checked In
										</Badge>
									) : (
										<Button onClick={checkInTicket} disabled={loading}>
											Check In
										</Button>
									)}
								</div>
								{currentTicket.checkedIn && currentTicket.checkedInAt && (
									<p className="text-xs text-muted-foreground mt-2">
										Checked in at {formatTime(currentTicket.checkedInAt)}
									</p>
								)}
							</div>
						)}
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<div className="flex justify-between items-center">
							<div>
								<CardTitle>Attendees</CardTitle>
								<CardDescription>
									{checkedInCount} of {totalCount} checked in ({checkedInPercentage}%)
								</CardDescription>
							</div>
							<Button size="sm" variant="outline" onClick={fetchAttendees} disabled={loading}>
								{loading ? 'Refreshing...' : 'Refresh'}
							</Button>
						</div>
					</CardHeader>
					<CardContent className="max-h-[500px] overflow-y-auto">
						{attendees.length === 0 ? (
							<div className="text-center py-10">
								<p className="text-muted-foreground">No attendees for this event yet</p>
							</div>
						) : (
							<ul className="space-y-2">
								{attendees.map((ticket) => (
									<li key={ticket.id} className="p-3 border rounded-md flex justify-between items-center">
										<div>
											<div className="font-medium">{ticket.attendeeName}</div>
											<div className="text-sm text-muted-foreground">{ticket.attendeeEmail}</div>
											<div className="text-xs text-muted-foreground mt-1">
												Code: {ticket.ticketCode}
											</div>
										</div>
										<div className="flex flex-col items-end">
											<Badge
												variant={ticket.checkedIn ? "default" : "outline"}
												className={ticket.checkedIn ? "bg-green-100 text-green-800" : ""}
											>
												{ticket.checkedIn ? "Checked In" : "Not Checked In"}
											</Badge>
											{ticket.checkedInAt && (
												<span className="text-xs text-muted-foreground mt-1">
													{formatTime(ticket.checkedInAt)}
												</span>
											)}
										</div>
									</li>
								))}
							</ul>
						)}
					</CardContent>
					<CardFooter>
						<div className="text-sm text-muted-foreground w-full text-center">
							{attendees.length > 0
								? `Last updated: ${new Date().toLocaleTimeString()}`
								: 'No attendees yet'
							}
						</div>
					</CardFooter>
				</Card>
			</div>
		</div>
	)
}
