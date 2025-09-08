import { apiClient } from '../api';
import type { ContentPlacement, UpdateContentPlacement } from '../schemas/content-placement-schema';

export const contentPlacementService = {
	async getAllPlacements(): Promise<ContentPlacement[]> {
		try {
			// This endpoint returns both static and dynamic placements
			const response = await apiClient.get<{ placements: ContentPlacement[] }>('/content-placements');
			return response.data.placements || [];
		} catch (error) {
			console.error('Failed to fetch content placements:', error);
			return [];
		}
	},

	async getPlacement(id: string): Promise<ContentPlacement | null> {
		try {
			const response = await apiClient.get<{ placement: ContentPlacement }>(`/content-placements/${id}`);
			return response.data.placement;
		} catch (error) {
			// Try to find in dynamic placements
			try {
				const dynamicResponse = await apiClient.get<{ placements: ContentPlacement[] }>('/content-placements');
				const placement = dynamicResponse.data.placements?.find(p => p.id === id);
				return placement || null;
			} catch (dynamicError) {
				console.error('Failed to fetch content placement:', error);
				return null;
			}
		}
	},

	async updatePlacement(id: string, data: UpdateContentPlacement): Promise<ContentPlacement> {
		try {
			const response = await apiClient.put<{ placement: ContentPlacement }>(
				`/content-placements/${id}`,
				data
			);
			return response.data.placement;
		} catch (error) {
			console.error('Failed to update content placement:', error);
			throw new Error('Failed to update content placement');
		}
	},

	async getPlacementsByMediaKey(mediaKey: string): Promise<ContentPlacement[]> {
		try {
			const response = await apiClient.get<{ placements: ContentPlacement[] }>(
				`/content-placements/by-media/${encodeURIComponent(mediaKey)}`
			);
			return response.data.placements;
		} catch (error) {
			console.error('Failed to fetch placements by media key:', error);
			return [];
		}
	},
};
