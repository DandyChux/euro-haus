import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs))
}

export interface ProcessedMetadata {
	[key: string]: any;
}

/**
 * Fetches externally stored metadata fields
 */
export async function fetchExternalMetadata(metadata: Record<string, string>): Promise<ProcessedMetadata> {
	const processed: ProcessedMetadata = {};

	for (const [key, value] of Object.entries(metadata)) {
		// Check if this field is stored externally
		if (key.endsWith('_external') && value === 'true') {
			const fieldName = key.replace('_external', '');
			const urlKey = `${fieldName}_url`;

			if (metadata[urlKey]) {
				try {
					// Fetch the external data
					const response = await fetch(metadata[urlKey]);
					if (response.ok) {
						const data = await response.json();
						processed[fieldName] = data;
					} else {
						console.warn(`Failed to fetch external metadata for ${fieldName}`);
						// Fall back to preview if available
						const previewKey = `${fieldName}_preview`;
						if (metadata[previewKey]) {
							processed[fieldName] = metadata[previewKey];
						}
					}
				} catch (error) {
					console.error(`Error fetching external metadata for ${fieldName}:`, error);
					// Fall back to preview if available
					const previewKey = `${fieldName}_preview`;
					if (metadata[previewKey]) {
						processed[fieldName] = metadata[previewKey];
					}
				}
			}
		} else if (!key.endsWith('_url') && !key.endsWith('_preview') && !key.endsWith('_external') && !key.endsWith('_truncated')) {
			// Regular metadata field
			processed[key] = value;
		}
	}

	return processed;
}
