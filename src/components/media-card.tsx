import { Card, CardContent } from '~/components/ui/card';
import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { Image } from '~/components/ui/image';
import {
	Trash2,
	Eye,
	Link2,
	CheckCircle,
	FileText
} from 'lucide-react';
import type { MediaFile } from '~/lib/services/gallery-service';
import type { ContentPlacement } from '~/lib/schemas/content-placement-schema';

interface MediaCardProps {
	file: MediaFile;
	onSelect: (file: MediaFile) => void;
	onDelete: (file: MediaFile) => void;
	contentPlacements?: ContentPlacement[];
	selectedPlacement?: ContentPlacement | null;
	onAssignToPlacement?: (file: MediaFile) => void;
}

export function MediaCard({
	file,
	onSelect,
	onDelete,
	contentPlacements,
	selectedPlacement,
	onAssignToPlacement
}: MediaCardProps) {
	const usedInPlacements = contentPlacements?.filter(p => p.mediaKey === file.key) || [];
	const isSelectedForPlacement = selectedPlacement?.mediaKey === file.key;

	return (
		<Card className={`group hover:shadow-lg transition-shadow ${isSelectedForPlacement ? 'ring-2 ring-primary' : ''}`}>
			<CardContent className="p-0">
				<div className="relative aspect-square">
					{file.type === 'image' ? (
						<Image
							src={file.url}
							alt={file.key.split('/').pop() || ''}
							className="w-full h-full object-cover rounded-t-lg"
						/>
					) : file.type === 'video' ? (
						<video
							src={file.url}
							className="w-full h-full object-cover rounded-t-lg"
							muted
						/>
					) : (
						<div className="w-full h-full bg-muted rounded-t-lg flex items-center justify-center">
							<FileText className="h-12 w-12 text-muted-foreground" />
						</div>
					)}

					{/* Usage indicators */}
					{usedInPlacements.length > 0 && (
						<div className="absolute top-2 left-2">
							<Badge variant="secondary" className="text-xs">
								<Link2 className="h-3 w-3 mr-1" />
								{usedInPlacements.length} placement{usedInPlacements.length > 1 ? 's' : ''}
							</Badge>
						</div>
					)}

					{/* Selected indicator */}
					{isSelectedForPlacement && (
						<div className="absolute top-2 right-2">
							<Badge variant="default" className="text-xs">
								<CheckCircle className="h-3 w-3 mr-1" />
								Selected
							</Badge>
						</div>
					)}

					{/* Action buttons */}
					<div className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
						<Button
							size="icon"
							variant="secondary"
							onClick={() => onSelect(file)}
						>
							<Eye className="h-4 w-4" />
						</Button>
						{selectedPlacement && (
							<Button
								size="sm"
								variant="default"
								onClick={() => onAssignToPlacement?.(file)}
							>
								Assign
							</Button>
						)}
						<Button
							size="icon"
							variant="destructive"
							onClick={() => onDelete(file)}
						>
							<Trash2 className="h-4 w-4" />
						</Button>
					</div>
				</div>

				<div className="p-3">
					<p className="text-sm font-medium truncate">
						{file.key.split('/').pop()}
					</p>
					<p className="text-xs text-muted-foreground">
						{formatFileSize(file.size)}
					</p>
				</div>
			</CardContent>
		</Card>
	);
}

function formatFileSize(bytes: number): string {
	if (bytes === 0) return '0 Bytes';
	const k = 1024;
	const sizes = ['Bytes', 'KB', 'MB', 'GB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
