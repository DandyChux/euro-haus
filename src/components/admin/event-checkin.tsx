import React, { useState, useEffect } from 'react';
import { QrReader } from 'react-qr-reader';
import { Card, CardContent, CardHeader, CardTitle } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Alert, AlertDescription } from '~/components/ui/alert';
import { Check, X, Search, Camera, Keyboard } from 'lucide-react';
import { ticketService } from '~/lib/services/ticket-service';
import { toast } from 'sonner';

interface EventCheckInProps {
	eventId: string;
	eventName: string;
}

export function EventCheckIn({ eventId, eventName }: EventCheckInProps) {
	const [scanMode, setScanMode] = useState<'camera' | 'manual'>('camera');
	const [manualCode, setManualCode] = useState('');
	const [isChecking, setIsChecking] = useState(false);
	const [lastCheckedIn, setLastCheckedIn] = useState<any>(null);
	const [stats, setStats] = useState({
		total: 0,
		checkedIn: 0,
		remaining: 0,
	});

	useEffect(() => {
		fetchEventStats();
	}, [eventId]);

	const fetchEventStats = async () => {
		try {
			const attendees = await ticketService.getEventAttendees(eventId);
			const checkedIn = attendees.filter(a => a.checkedIn).length;
			setStats({
				total: attendees.length,
				checkedIn,
				remaining: attendees.length - checkedIn,
			});
		} catch (error) {
			console.error('Failed to fetch event stats:', error);
		}
	};

	const handleScan = async (data: string | null) => {
		if (data && !isChecking) {
			await checkInTicket(data);
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
			// First validate the ticket
			const ticket = await ticketService.validateTicket(ticketCode);

			if (!ticket) {
				toast.error('Invalid ticket code');
				return;
			}

			if (ticket.eventId !== eventId) {
				toast.error('This ticket is for a different event');
				return;
			}

			if (ticket.checkedIn) {
				toast.warning(`Already checked in at ${new Date(ticket.checkedInAt!).toLocaleString()}`);
				setLastCheckedIn({ ...ticket, alreadyCheckedIn: true });
				return;
			}

			// Check in the ticket
			const checkedInTicket = await ticketService.checkInTicket(ticketCode);

			toast.success(`Checked in: ${checkedInTicket.attendeeName}`);
			setLastCheckedIn(checkedInTicket);

			// Update stats
			setStats(prev => ({
				...prev,
				checkedIn: prev.checkedIn + 1,
				remaining: prev.remaining - 1,
			}));

		} catch (error) {
			toast.error('Failed to check in ticket');
		} finally {
			setIsChecking(false);
		}
	};

	return (
		<div className="space-y-6">
			{/* Stats */}
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
							<QrReader
								onResult={(result, error) => {
									if (result) {
										handleScan(result.getText());
									}
								}}
								constraints={{ facingMode: 'environment' }}
								className="w-full"
							/>
						</div>
					) : (
						<form onSubmit={handleManualSubmit} className="flex gap-2">
							<Input
								value={manualCode}
								onChange={(e) => setManualCode(e.target.value)}
								placeholder="Enter ticket code..."
								className="flex-1"
								disabled={isChecking}
							/>
							<Button type="submit" disabled={isChecking || !manualCode.trim()}>
								<Search className="w-4 h-4 mr-2" />
								Check In
							</Button>
						</form>
					)}
				</CardContent>
			</Card>

			{/* Last Check-in Result */}
			{lastCheckedIn && (
				<Alert className={lastCheckedIn.alreadyCheckedIn ? 'border-yellow-500' : 'border-green-500'}>
					<div className="flex items-start gap-2">
						{lastCheckedIn.alreadyCheckedIn ? (
							<X className="w-5 h-5 text-yellow-500" />
						) : (
							<Check className="w-5 h-5 text-green-500" />
						)}
						<div className="flex-1">
							<AlertDescription>
								<div className="font-semibold">{lastCheckedIn.attendeeName}</div>
								<div className="text-sm text-gray-600">
									Ticket: {lastCheckedIn.ticketType} • Code: {lastCheckedIn.ticketCode}
								</div>
								{lastCheckedIn.alreadyCheckedIn && (
									<div className="text-sm text-yellow-600 mt-1">
										Already checked in at {new Date(lastCheckedIn.checkedInAt).toLocaleString()}
									</div>
								)}
							</AlertDescription>
						</div>
					</div>
				</Alert>
			)}
		</div>
	);
}
