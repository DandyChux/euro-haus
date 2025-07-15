import { Card } from "~/components/ui/card"

interface MapLocationProps {
	address: string
	mapUrl: string
	title?: string
	description?: string
}

export function MapLocation({
	address,
	mapUrl,
	title = "Location Map",
	description
}: MapLocationProps) {
	return (
		<Card className="overflow-hidden">
			<div className="aspect-video w-full">
				<iframe
					src={mapUrl}
					width="100%"
					height="100%"
					style={{ border: 0 }}
					allowFullScreen={false}
					loading="lazy"
					referrerPolicy="no-referrer-when-downgrade"
					title={title}
					className="h-full w-full"
				></iframe>
			</div>
			<div className="p-4 bg-muted/50">
				<p className="text-sm text-muted-foreground">
					{description || `Located at ${address}`}
				</p>
			</div>
		</Card>
	)
}
