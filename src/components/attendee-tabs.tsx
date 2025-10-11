import { Tabs, TabsList, TabsTrigger, TabsContent } from "~/components/ui/tabs";
import { Badge } from "~/components/ui/badge";
import { format } from "date-fns";
import { useState } from "react";

export function AttendeeTabs({ attendees }: { attendees: any[] }) {
	// Group attendees by ticket type
	const groupedAttendees = attendees.reduce((groups, ticket) => {
		const type = ticket.ticketType || 'General';
		if (!groups[type]) {
			groups[type] = [];
		}
		groups[type].push(ticket);
		return groups;
	}, {} as Record<string, typeof attendees>);

	const sortedTiers = Object.keys(groupedAttendees).sort((a, b) => a.localeCompare(b));
	const [activeTierTab, setActiveTierTab] = useState(sortedTiers[0] || '');

	if (sortedTiers.length === 0) {
		return null;
	}

	return (
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
								className="flex justify-between items-center p-3 border rounded-lg"
							>
								<div>
									<div className="font-medium">{ticket.attendeeName}</div>
									<div className="text-sm text-muted-foreground">
										{ticket.attendeeEmail}
									</div>
									<div className="text-xs text-muted-foreground mt-1">
										Code: {ticket.ticketCode}
									</div>
								</div>
								<div className="text-right">
									<Badge
										variant={ticket.checkedIn ? 'default' : 'outline'}
										className={ticket.checkedIn ? 'bg-green-100 text-green-800' : ''}
									>
										{ticket.checkedIn ? 'Checked In' : 'Not Checked In'}
									</Badge>
									{ticket.checkedInAt && (
										<div className="text-xs text-muted-foreground mt-1">
											{format(new Date(ticket.checkedInAt), 'pp')}
										</div>
									)}
								</div>
							</div>
						))}
					</div>
				</TabsContent>
			))}
		</Tabs>
	);
}
