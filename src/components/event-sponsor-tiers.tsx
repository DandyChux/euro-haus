import { SponsorTier } from '~/lib/schemas/product-schema';
import { InfiniteMovingCards } from './ui/infinite-moving-cards';

interface EventSponsorTiersProps {
	sponsorTiers: SponsorTier[];
}

export function EventSponsorTiers({ sponsorTiers }: EventSponsorTiersProps) {
	// Sort tiers by display order
	const sortedTiers = [...sponsorTiers].sort((a, b) => (a.displayOrder || 0) - (b.displayOrder || 0));

	if (sortedTiers.length === 0) return null;
	console.log(sortedTiers)

	return (
		<div className="space-y-12">
			{sortedTiers.map((tier, index) => (
				<div key={index} className="space-y-4">
					<h3 className="text-2xl font-bold text-center">{tier.tierName}</h3>

					{tier.sponsors.length > 0 && (
						<InfiniteMovingCards
							items={tier.sponsors.map(sponsor => ({
								logoUrl: sponsor.logoUrl,
								name: sponsor.name,
								link: sponsor.link
							}))}
							variant='logo'
							direction="left"
							speed="normal"
							pauseOnHover={true}
							className="py-4"
						/>
					)}
				</div>
			))}
		</div>
	);
}
