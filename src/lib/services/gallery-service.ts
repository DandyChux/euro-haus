import { apiClient } from '../api';
import type { EventProduct } from './stripe-service';

export interface MediaFile {
	key: string;
	url: string;
	lastModified: string;
	size: number;
	type: 'image' | 'video' | 'other';
	folder: string;
}

export interface MediaResponse {
	files: MediaFile[];
	total: number;
}

export interface EventGallery {
	eventSlug: string;
	eventName?: string;
	images: MediaFile[];
}

export interface EventFolder {
	name: string;
	path: string;
}

export interface EventFoldersResponse {
	folders: EventFolder[];
	total: number;
}

export interface UploadResponse {
	success: boolean;
	file: MediaFile;
	message: string;
}

export interface BatchUploadResponse {
	success: boolean;
	files: MediaFile[];
	totalUploaded: number;
	totalFailed: number;
	errors: string[];
	message: string;
}


export const galleryService = {
	/**
	 * Fetches all media files from DigitalOcean Spaces
	 */
	async getAllMedia(): Promise<MediaResponse> {
		const response = await apiClient.get<MediaResponse>('/media');

		if (response.status !== 200) {
			console.error('Failed to fetch media files:', response.statusText);
			throw new Error('Failed to fetch media files');
		}

		return response.data || [];
	},

	/**
	 * Fetches gallery images for all events
	 * Filters media files from the events/{slug}/gallery/ folders
	 */
	async getAllEventGalleries(events: EventProduct[]): Promise<EventGallery[]> {
		try {
			const allMedia = await this.getAllMedia();

			// Filter images from events/{slug}/gallery/ folders
			const eventGalleries: EventGallery[] = [];

			for (const event of events) {
				const galleryPrefix = `events/${event.slug}/gallery/`;
				const eventImages = allMedia.files.filter(file =>
					file.key.startsWith(galleryPrefix) &&
					file.type === 'image'
				);

				if (eventImages.length > 0) {
					eventGalleries.push({
						eventSlug: event.slug,
						eventName: event.title,
						images: eventImages.sort((a, b) =>
							new Date(b.lastModified).getTime() - new Date(a.lastModified).getTime()
						)
					});
				}
			}

			return eventGalleries;
		} catch (error) {
			console.error('Failed to fetch event galleries:', error);
			return [];
		}
	},

	/**
	 * Fetches gallery images for a specific event
	 */
	async getEventGallery(eventSlug: string): Promise<MediaFile[]> {
		try {
			const allMedia = await this.getAllMedia();
			const galleryPrefix = `events/${eventSlug}/gallery/`;

			const eventImages = allMedia.files.filter(file =>
				file.key.startsWith(galleryPrefix) &&
				file.type === 'image'
			);

			// Sort by last modified date (newest first)
			return eventImages.sort((a, b) =>
				new Date(b.lastModified).getTime() - new Date(a.lastModified).getTime()
			);
		} catch (error) {
			console.error(`Failed to fetch gallery for event ${eventSlug}:`, error);
			return [];
		}
	},

	/**
		 * Fetches all event folders from the events/ directory in Spaces
		 */
	async getEventFolders(): Promise<EventFolder[]> {
		try {
			const response = await apiClient.get<EventFoldersResponse>('/admin/events/folders');
			return response.data.folders || [];
		} catch (error) {
			console.error('Failed to fetch event folders:', error);
			return [];
		}
	},

	/**
	 * Uploads a file to an event's gallery folder
	 */
	async uploadToEventGallery(eventSlug: string, file: File): Promise<MediaFile | null> {
		try {
			const formData = new FormData();
			formData.append('file', file);
			formData.append('eventSlug', eventSlug);

			const response = await apiClient.post<UploadResponse>(
				'/admin/events/gallery/upload',
				formData,
				{
					headers: {
						'Content-Type': 'multipart/form-data',
					},
				}
			);

			return response.data.file;
		} catch (error) {
			console.error('Failed to upload file to event gallery:', error);
			throw error;
		}
	},

	/**
		 * Uploads multiple files in a single batch request
		 */
	async uploadMultipleToEventGallery(
		eventSlug: string,
		files: File[],
		onProgress?: (progress: number) => void
	): Promise<MediaFile[]> {
		const formData = new FormData();
		formData.append('eventSlug', eventSlug);

		// Append all files with the same field name "files"
		files.forEach(file => {
			formData.append('files', file);
		});

		const response = await apiClient.post<BatchUploadResponse>(
			'/admin/events/gallery/upload',
			formData,
			{
				headers: {
					'Content-Type': 'multipart/form-data',
				},
				onUploadProgress: (progressEvent) => {
					if (onProgress && progressEvent.total) {
						const percentComplete = (progressEvent.loaded / progressEvent.total) * 100;
						onProgress(percentComplete);
					}
				},
			}
		);

		if (response.data.errors && response.data.errors.length > 0) {
			console.warn('Some files failed to upload:', response.data.errors);
		}

		return response.data.files;
	},

	/**
		 * Uploads files sequentially (fallback method)
		 */
	async uploadSequentialToEventGallery(
		eventSlug: string,
		files: File[],
		onProgress?: (progress: number) => void
	): Promise<MediaFile[]> {
		const uploadedFiles: MediaFile[] = [];

		for (let i = 0; i < files.length; i++) {
			try {
				const file = await this.uploadToEventGallery(eventSlug, files[i]);
				if (file) {
					uploadedFiles.push(file);
				}

				if (onProgress) {
					onProgress(((i + 1) / files.length) * 100);
				}
			} catch (error) {
				console.error(`Failed to upload file ${files[i].name}:`, error);
			}
		}

		return uploadedFiles;
	},

	/**
	 * Gets preview images (first N images) for an event gallery
	 */
	getPreviewImages(images: MediaFile[], count: number = 3): MediaFile[] {
		return images.slice(0, count);
	}
};
