import { Tabs, TabsList, TabsTrigger, TabsContent } from "~/components/ui/tabs";
import { Badge } from "~/components/ui/badge";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { format } from "date-fns";
import { useState } from "react";
import { Search, Check, Clock } from "lucide-react";

interface AttendeeTabsProps {
	attendees: any[];
	onManualCheckIn?: (ticketCode: string) => Promise<void>;
	checkingInTicket?: string | null;
}

export function AttendeeTabs({ attendees, onManualCheckIn, checkingInTicket }: AttendeeTabsProps) {
	const [searchQuery, setSearchQuery] = useState('');

	// Filter attendees based on search query
	const filteredAttendees = attendees.filter(ticket => {
		if (!searchQuery.trim()) return true;

		const query = searchQuery.toLowerCase();
		return (
			ticket.attendeeName?.toLowerCase().includes(query) ||
			ticket.attendeeEmail?.toLowerCase().includes(query) ||
			ticket.ticketCode?.toLowerCase().includes(query)
		);
	});

	// Group filtered attendees by ticket type
	const groupedAttendees = filteredAttendees.reduce((groups, ticket) => {
		const type = ticket.ticketType || 'General';
		if (!groups[type]) {
			groups[type] = [];
		}
		groups[type].push(ticket);
		return groups;
	}, {} as Record<string, typeof attendees>);

	const sortedTiers = Object.keys(groupedAttendees).sort((a, b) => a.localeCompare(b));
	const [activeTierTab, setActiveTierTab] = useState(sortedTiers[0] || '');

	// Update active tab if search filters it out
	if (sortedTiers.length > 0 && !sortedTiers.includes(activeTierTab)) {
		setActiveTierTab(sortedTiers[0]);
	}

	if (attendees.length === 0) {
		return null;
	}

	return (
		<div className="space-y-4">
			{/* Search Bar */}
			<div className="relative">
				<Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
				<Input
					type="text"
					placeholder="Search by name, email, or ticket code..."
					value={searchQuery}
					onChange={(e) => setSearchQuery(e.target.value)}
					className="pl-9"
				/>
			</div>

			{/* Results count */}
			{searchQuery && (
				<div className="text-sm text-muted-foreground">
					Found {filteredAttendees.length} of {attendees.length} attendees
				</div>
			)}

			{sortedTiers.length === 0 ? (
				<div className="text-center py-8 text-muted-foreground">
					No attendees match your search
				</div>
			) : (
				<Tabs value={activeTierTab} onValueChange={setActiveTierTab}>
					<div className="relative w-full overflow-x-auto">
						<TabsList className="w-max space-x-2">
							{sortedTiers.map(tier => {
								const tierAttendees = groupedAttendees[tier];
								const checkedInCount = tierAttendees.filter((t: any) => t.checkedIn).length;
								const totalCount = tierAttendees.length;
								return (
									<TabsTrigger key={tier} value={tier} className="w-auto">
										<div className="flex items-center justify-between w-full gap-4">
											<span>{tier}</span>
											<Badge
												variant={activeTierTab === tier ? 'default' : 'secondary'}
												className="ml-2 h-5 flex-shrink-0"
											>
												{checkedInCount} / {totalCount}
											</Badge>
										</div>
									</TabsTrigger>
								);
							})}
						</TabsList>
					</div>

					{sortedTiers.map(tier => (
						<TabsContent key={tier} value={tier}>
							<div className="max-h-80 overflow-y-auto space-y-2 pt-4">
								{groupedAttendees[tier].map((ticket: any) => (
									<div
										key={ticket.id}
										className="flex justify-between items-center p-3 border rounded-lg hover:bg-accent/50 transition-colors"
									>
										<div className="flex-1">
											<div className="font-medium">{ticket.attendeeName}</div>
											<div className="text-sm text-muted-foreground">
												{ticket.attendeeEmail}
											</div>
											<div className="text-xs text-muted-foreground mt-1">
												Code: {ticket.ticketCode}
											</div>
										</div>
										<div className="flex items-center gap-3">
											<div className="text-right">
												<Badge
													variant={ticket.checkedIn ? 'default' : 'outline'}
													className={ticket.checkedIn ? 'bg-green-100 text-green-800' : ''}
												>
													{ticket.checkedIn ? (
														<>
															<Check className="w-3 h-3 mr-1" />
															Checked In
														</>
													) : (
														'Not Checked In'
													)}
												</Badge>
												{ticket.checkedInAt && (
													<div className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
														<Clock className="w-3 h-3" />
														{format(new Date(ticket.checkedInAt), 'pp')}
													</div>
												)}
											</div>
											{!ticket.checkedIn && onManualCheckIn && (
												<Button
													size="sm"
													onClick={() => onManualCheckIn(ticket.ticketCode)}
													disabled={checkingInTicket === ticket.ticketCode}
												>
													{checkingInTicket === ticket.ticketCode ? (
														'Checking In...'
													) : (
														'Check In'
													)}
												</Button>
											)}
										</div>
									</div>
								))}
							</div>
						</TabsContent>
					))}
				</Tabs>
			)}
		</div>
	);
}
