import { createClient } from '@sanity/client'
import imageUrlBuilder from '@sanity/image-url'
import type { SanityImageSource } from '@sanity/image-url/lib/types/types'

export const sanityClient = createClient({
	projectId: import.meta.env.VITE_SANITY_PROJECT_ID,
	dataset: import.meta.env.VITE_SANITY_DATASET || 'production',
	useCdn: true, // set to false for fresh data
	apiVersion: '2024-01-01', // use current date
})

const builder = imageUrlBuilder(sanityClient)

export function urlFor(source: SanityImageSource) {
	return builder.image(source)
}

// Type for Sanity Event
export interface SanityEvent {
	_id: string
	_createdAt: string
	_updatedAt: string
	title: string
	slug: { current: string }
	date: string
	description: string
	longDescription?: any[]
	location: {
		name: string
		address?: string
		city?: string
		state?: string
		coordinates?: { lat: number; lng: number }
	}
	price: number
	image: SanityImageSource & { alt?: string }
	gallery?: (SanityImageSource & { alt?: string })[]
	capacity?: number
	availableSpots?: number
	organizer?: string
	tags?: string[]
	status?: 'upcoming' | 'ongoing' | 'completed' | 'cancelled' | 'soldout'
	agenda?: { time: string; activity: string }[]
	includes?: string[]
	featured?: boolean
	registrationLink?: string
}
