import { Card } from "~/components/ui/card"
import { MapPin, Clock, Phone, Mail, Globe, Info, Calendar, Navigation } from "lucide-react"
import { Button } from "~/components/ui/button"
import { Separator } from "~/components/ui/separator"

interface MapLocationProps {
	address: string
	mapUrl: string
	title?: string
	description?: string
	hours?: {
		day: string
		hours: string
		isToday?: boolean
	}[]
	phone?: string
	email?: string
	website?: string
	additionalInfo?: string[]
	eventDate?: string
	directionsUrl?: string
}

export function MapLocation({
	address,
	mapUrl,
	title = "Location Map",
	description,
	hours,
	phone,
	email,
	website,
	additionalInfo,
	eventDate,
	directionsUrl
}: MapLocationProps) {
	const handleGetDirections = () => {
		if (directionsUrl) {
			window.open(directionsUrl, '_blank')
		} else {
			// Default to Google Maps directions
			const encodedAddress = encodeURIComponent(address)
			window.open(`https://www.google.com/maps/dir/?api=1&destination=${encodedAddress}`, '_blank')
		}
	}

	return (
		<Card className="overflow-hidden shadow-neumorph hover:shadow-neumorph-hover transition-all">
			<div className="aspect-video w-full relative group">
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
				<div className="absolute inset-0 bg-gradient-to-t from-background/20 to-transparent pointer-events-none opacity-0 group-hover:opacity-100 transition-opacity" />
			</div>

			<div className="p-6 space-y-4 bg-gradient-to-br from-muted/30 to-muted/10">
				{/* Main Address Section */}
				<div className="space-y-3">
					<div className="flex items-start gap-3">
						<div className="p-2 bg-gradient-to-br from-primary to-secondary rounded-lg shadow-md">
							<MapPin className="h-5 w-5 text-primary-foreground" />
						</div>
						<div className="flex-1">
							<h4 className="font-semibold text-lg mb-1">{title}</h4>
							<p className="text-muted-foreground">{address}</p>
							{description && (
								<p className="text-sm text-muted-foreground mt-2">{description}</p>
							)}
						</div>
					</div>

					<Button
						onClick={handleGetDirections}
						className="w-full bg-gradient-to-r from-primary to-secondary hover:opacity-90 transition-all"
					>
						<Navigation className="mr-2 h-4 w-4" />
						Get Directions
					</Button>
				</div>

				{(hours || phone || email || website) && (
					<>
						<Separator className="bg-gradient-to-r from-transparent via-border to-transparent" />

						<div className="grid gap-4 sm:grid-cols-2">
							{/* Hours */}
							{hours && hours.length > 0 && (
								<div className="flex items-start gap-3 p-3 rounded-lg bg-gradient-to-r from-primary/5 to-transparent">
									<Clock className="h-4 w-4 text-primary mt-0.5" />
									<div className="flex-1">
										<p className="text-xs text-muted-foreground mb-1">Hours</p>
										<div className="space-y-1">
											{hours.map((schedule, idx) => (
												<div key={idx} className={`text-sm ${schedule.isToday ? 'font-semibold text-primary' : ''}`}>
													<span className="inline-block w-20 text-xs">{schedule.day}:</span>
													<span className="text-xs">{schedule.hours}</span>
												</div>
											))}
										</div>
									</div>
								</div>
							)}

							{/* Contact Info */}
							{(phone || email) && (
								<div className="space-y-2 p-3 rounded-lg bg-gradient-to-r from-secondary/5 to-transparent">
									{phone && (
										<a href={`tel:${phone}`} className="flex items-center gap-2 text-sm hover:text-primary transition-colors">
											<Phone className="h-4 w-4 text-secondary" />
											<span>{phone}</span>
										</a>
									)}
									{email && (
										<a href={`mailto:${email}`} className="flex items-center gap-2 text-sm hover:text-primary transition-colors">
											<Mail className="h-4 w-4 text-secondary" />
											<span>{email}</span>
										</a>
									)}
								</div>
							)}

							{/* Website */}
							{website && (
								<div className="p-3 rounded-lg bg-gradient-to-r from-chart-1/5 to-transparent">
									<a
										href={website}
										target="_blank"
										rel="noopener noreferrer"
										className="flex items-center gap-2 text-sm hover:text-primary transition-colors"
									>
										<Globe className="h-4 w-4 text-chart-1" />
										<span>Visit Website</span>
									</a>
								</div>
							)}
						</div>
					</>
				)}

				{/* Additional Info */}
				{additionalInfo && additionalInfo.length > 0 && (
					<>
						<Separator className="bg-gradient-to-r from-transparent via-border to-transparent" />
						<div className="space-y-2">
							<div className="flex items-center gap-2 text-sm font-medium">
								<Info className="h-4 w-4 text-muted-foreground" />
								<span>Good to Know</span>
							</div>
							<ul className="space-y-1.5 ml-6">
								{additionalInfo.map((info, idx) => (
									<li key={idx} className="text-sm text-muted-foreground flex items-start gap-2">
										<span className="text-primary mt-1.5 text-xs">•</span>
										<span>{info}</span>
									</li>
								))}
							</ul>
						</div>
					</>
				)}
			</div>
		</Card>
	)
}
