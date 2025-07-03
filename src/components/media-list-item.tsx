import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { Image } from '~/components/ui/image';
import {
	Trash2,
	Eye,
	Link2,
	CheckCircle,
	FileText,
	ImageIcon,
	Video,
	FolderOpen
} from 'lucide-react';
import type { MediaFile } from '~/routes/admin/media';
import type { ContentPlacement } from '~/lib/schemas/content-placement-schema';
import { formatDistanceToNow } from 'date-fns';

interface MediaListItemProps {
	file: MediaFile;
	onSelect: (file: MediaFile) => void;
	onDelete: (file: MediaFile) => void;
	contentPlacements?: ContentPlacement[];
	selectedPlacement?: ContentPlacement | null;
	onAssignToPlacement?: (file: MediaFile) => void;
}

export function MediaListItem({
	file,
	onSelect,
	onDelete,
	contentPlacements,
	selectedPlacement,
	onAssignToPlacement
}: MediaListItemProps) {
	const usedInPlacements = contentPlacements?.filter(p => p.mediaKey === file.key) || [];
	const isSelectedForPlacement = selectedPlacement?.mediaKey === file.key;

	const getFileIcon = () => {
		switch (file.type) {
			case 'image':
				return <ImageIcon className="h-5 w-5 text-muted-foreground" />;
			case 'video':
				return <Video className="h-5 w-5 text-muted-foreground" />;
			default:
				return <FileText className="h-5 w-5 text-muted-foreground" />;
		}
	};

	return (
		<div className={`flex items-center gap-4 p-4 border rounded-lg hover:shadow-md transition-shadow ${isSelectedForPlacement ? 'ring-2 ring-primary' : ''
			}`}>
			{/* Preview */}
			<div className="w-16 h-16 flex-shrink-0">
				{file.type === 'image' ? (
					<Image
						src={file.url}
						alt={file.key.split('/').pop() || ''}
						className="w-full h-full object-cover rounded"
					/>
				) : file.type === 'video' ? (
					<video
						src={file.url}
						className="w-full h-full object-cover rounded"
						muted
					/>
				) : (
					<div className="w-full h-full bg-muted rounded flex items-center justify-center">
						{getFileIcon()}
					</div>
				)}
			</div>

			{/* File Info */}
			<div className="flex-1 min-w-0">
				<div className="flex items-center gap-2">
					<p className="font-medium truncate">{file.key.split('/').pop()}</p>
					{isSelectedForPlacement && (
						<Badge variant="default" className="text-xs">
							<CheckCircle className="h-3 w-3 mr-1" />
							Selected
						</Badge>
					)}
				</div>
				<div className="flex items-center gap-4 text-xs text-muted-foreground mt-1">
					<span className="flex items-center gap-1">
						{getFileIcon()}
						{file.type}
					</span>
					<span>{formatFileSize(file.size)}</span>
					<span>{formatDistanceToNow(new Date(file.lastModified), { addSuffix: true })}</span>
					{file.folder && (
						<span className="flex items-center gap-1">
							<FolderOpen className="h-3 w-3" />
							{file.folder}
						</span>
					)}
				</div>
				{usedInPlacements.length > 0 && (
					<div className="flex items-center gap-2 mt-2">
						<Badge variant="outline" className="text-xs">
							<Link2 className="h-3 w-3 mr-1" />
							Used in:
						</Badge>
						{usedInPlacements.map((placement, index) => (
							<span key={placement.id} className="text-xs text-muted-foreground">
								{placement.name}{index < usedInPlacements.length - 1 ? ', ' : ''}
							</span>
						))}
					</div>
				)}
			</div>

			{/* Actions */}
			<div className="flex items-center gap-2">
				<Button
					size="sm"
					variant="outline"
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
					size="sm"
					variant="outline"
					onClick={() => onDelete(file)}
				>
					<Trash2 className="h-4 w-4" />
				</Button>
			</div>
		</div>
	);
}

function formatFileSize(bytes: number): string {
	if (bytes === 0) return '0 Bytes';
	const k = 1024;
	const sizes = ['Bytes', 'KB', 'MB', 'GB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
