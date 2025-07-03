import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { contentPlacementService } from '../services/content-placement-service';
import type { UpdateContentPlacement } from '../schemas/content-placement-schema';
import { toast } from 'sonner';

export function useContentPlacements() {
	return useQuery({
		queryKey: ['content-placements'],
		queryFn: contentPlacementService.getAllPlacements,
		staleTime: 5 * 60 * 1000, // 5 minutes
	});
}

export function useContentPlacement(id: string) {
	return useQuery({
		queryKey: ['content-placement', id],
		queryFn: () => contentPlacementService.getPlacement(id),
		enabled: !!id,
	});
}

export function useUpdateContentPlacement() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: ({ id, data }: { id: string; data: UpdateContentPlacement }) =>
			contentPlacementService.updatePlacement(id, data),
		onSuccess: (data) => {
			queryClient.invalidateQueries({ queryKey: ['content-placements'] });
			queryClient.invalidateQueries({ queryKey: ['content-placement', data.id] });
			toast.success('Content placement updated successfully');
		},
		onError: (error) => {
			toast.error('Failed to update content placement');
			console.error('Update placement error:', error);
		},
	});
}

export function useContentPlacementByMedia(mediaKey: string) {
	return useQuery({
		queryKey: ['content-placements-by-media', mediaKey],
		queryFn: () => contentPlacementService.getPlacementsByMediaKey(mediaKey),
		enabled: !!mediaKey,
	});
}
