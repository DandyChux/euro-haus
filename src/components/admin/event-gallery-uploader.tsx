import { useState, useRef, useCallback } from 'react';
import { Button } from '~/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '~/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { Progress } from '~/components/ui/progress';
import { Image } from '~/components/ui/image';
import { galleryService } from '~/lib/services/gallery-service';
import { Upload, X, Loader2, ImageIcon, Video, Trash2 } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { apiClient } from '~/lib/api';

interface EventGalleryUploaderProps {
	currentEventSlug?: string;
}

export function EventGalleryUploader({ currentEventSlug }: EventGalleryUploaderProps) {
	const [selectedEventSlug, setSelectedEventSlug] = useState<string>(currentEventSlug || '');
	const [selectedFiles, setSelectedFiles] = useState<File[]>([]);
	const [uploadProgress, setUploadProgress] = useState(0);
	const [previews, setPreviews] = useState<string[]>([]);
	const fileInputRef = useRef<HTMLInputElement>(null);
	const queryClient = useQueryClient();

	// Fetch event folders
	const { data: eventFolders = [], isLoading: loadingFolders } = useQuery({
		queryKey: ['eventFolders'],
		queryFn: () => galleryService.getEventFolders(),
	});

	// Fetch existing gallery images for selected event
	const { data: existingImages = [], isLoading: loadingImages, refetch: refetchImages } = useQuery({
		queryKey: ['eventGallery', selectedEventSlug],
		queryFn: () => galleryService.getEventGallery(selectedEventSlug),
		enabled: !!selectedEventSlug,
	});

	// Upload mutation
	const uploadMutation = useMutation({
		mutationFn: async (files: File[]) => {
			return galleryService.uploadMultipleToEventGallery(
				selectedEventSlug,
				files,
				setUploadProgress
			);
		},
		onSuccess: (uploadedFiles) => {
			toast.success(`Successfully uploaded ${uploadedFiles.length} file(s)`);
			setSelectedFiles([]);
			setPreviews([]);
			setUploadProgress(0);
			refetchImages();
			queryClient.invalidateQueries({ queryKey: ['eventGallery', selectedEventSlug] });
		},
		onError: (error) => {
			toast.error('Failed to upload files');
			console.error(error);
		},
	});

	// Delete mutation
	const deleteMutation = useMutation({
		mutationFn: async (fileKey: string) => {
			await apiClient.delete('/admin/media/delete', {
				data: { key: fileKey },
			});
		},
		onSuccess: () => {
			toast.success('File deleted successfully');
			refetchImages();
			queryClient.invalidateQueries({ queryKey: ['eventGallery', selectedEventSlug] });
		},
		onError: (error) => {
			toast.error('Failed to delete file');
			console.error(error);
		},
	});

	const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
		const files = Array.from(e.target.files || []);
		if (files.length === 0) return;

		// Validate file types
		const validFiles = files.filter(file => {
			const isImage = file.type.startsWith('image/');
			const isVideo = file.type.startsWith('video/');
			return isImage || isVideo;
		});

		if (validFiles.length !== files.length) {
			toast.error('Some files were skipped. Only images and videos are allowed.');
		}

		setSelectedFiles(prev => [...prev, ...validFiles]);

		// Generate previews
		validFiles.forEach(file => {
			const reader = new FileReader();
			reader.onloadend = () => {
				setPreviews(prev => [...prev, reader.result as string]);
			};
			reader.readAsDataURL(file);
		});

		// Reset input
		if (e.target) {
			e.target.value = '';
		}
	}, []);

	const handleRemoveFile = useCallback((index: number) => {
		setSelectedFiles(prev => prev.filter((_, i) => i !== index));
		setPreviews(prev => prev.filter((_, i) => i !== index));
	}, []);

	const handleUpload = useCallback(() => {
		if (!selectedEventSlug) {
			toast.error('Please select an event folder');
			return;
		}

		if (selectedFiles.length === 0) {
			toast.error('Please select files to upload');
			return;
		}

		uploadMutation.mutate(selectedFiles);
	}, [selectedEventSlug, selectedFiles, uploadMutation]);

	const handleDelete = useCallback((fileKey: string) => {
		if (confirm('Are you sure you want to delete this file?')) {
			deleteMutation.mutate(fileKey);
		}
	}, [deleteMutation]);

	const isUploading = uploadMutation.isPending;

	return (
		<div className="space-y-6">
			{/* Upload Section */}
			<Card>
				<CardHeader>
					<CardTitle>Upload Media</CardTitle>
					<CardDescription>
						Select an event folder and upload images or videos to its gallery.
					</CardDescription>
				</CardHeader>
				<CardContent className="space-y-4">
					{/* Event Folder Selector */}
					<div className="space-y-2">
						<label className="text-sm font-medium">Select Event Folder</label>
						<Select
							value={selectedEventSlug}
							onValueChange={setSelectedEventSlug}
							disabled={loadingFolders || isUploading}
						>
							<SelectTrigger>
								<SelectValue placeholder="Choose an event..." />
							</SelectTrigger>
							<SelectContent>
								{eventFolders.map((folder) => (
									<SelectItem key={folder.name} value={folder.name}>
										{folder.name}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* File Upload Area */}
					<div className="space-y-2">
						<label className="text-sm font-medium">Select Files</label>
						<div
							className="border-2 border-dashed rounded-lg p-8 text-center cursor-pointer hover:border-primary transition-colors"
							onClick={() => fileInputRef.current?.click()}
						>
							<Upload className="w-12 h-12 mx-auto mb-4 text-muted-foreground" />
							<p className="text-sm text-muted-foreground mb-2">
								Click to upload or drag and drop
							</p>
							<p className="text-xs text-muted-foreground">
								Images (JPG, PNG, GIF, WebP) and Videos (MP4, WebM, MOV)
							</p>
						</div>
						<input
							ref={fileInputRef}
							type="file"
							multiple
							accept="image/*,video/*"
							className="hidden"
							onChange={handleFileSelect}
							disabled={isUploading}
						/>
					</div>

					{/* Selected Files Preview */}
					{selectedFiles.length > 0 && (
						<div className="space-y-2">
							<label className="text-sm font-medium">
								Selected Files ({selectedFiles.length})
							</label>
							<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
								{selectedFiles.map((file, index) => (
									<div key={index} className="relative group">
										<div className="aspect-square rounded-lg overflow-hidden bg-muted">
											{file.type.startsWith('image/') ? (
												<img
													src={previews[index]}
													alt={file.name}
													className="w-full h-full object-cover"
												/>
											) : (
												<div className="w-full h-full flex items-center justify-center">
													<Video className="w-8 h-8 text-muted-foreground" />
												</div>
											)}
										</div>
										<button
											onClick={() => handleRemoveFile(index)}
											className="absolute top-2 right-2 p-1 bg-destructive text-destructive-foreground rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
											disabled={isUploading}
										>
											<X className="w-4 h-4" />
										</button>
										<p className="text-xs mt-1 truncate" title={file.name}>
											{file.name}
										</p>
									</div>
								))}
							</div>
						</div>
					)}

					{/* Upload Progress */}
					{isUploading && (
						<div className="space-y-2">
							<div className="flex justify-between text-sm">
								<span>Uploading...</span>
								<span>{Math.round(uploadProgress)}%</span>
							</div>
							<Progress value={uploadProgress} />
						</div>
					)}

					{/* Upload Button */}
					<Button
						onClick={handleUpload}
						disabled={!selectedEventSlug || selectedFiles.length === 0 || isUploading}
						className="w-full"
					>
						{isUploading ? (
							<>
								<Loader2 className="mr-2 w-4 h-4 animate-spin" />
								Uploading...
							</>
						) : (
							<>
								<Upload className="mr-2 w-4 h-4" />
								Upload {selectedFiles.length > 0 ? `${selectedFiles.length} file(s)` : ''}
							</>
						)}
					</Button>
				</CardContent>
			</Card>

			{/* Existing Gallery Section */}
			{selectedEventSlug && (
				<Card>
					<CardHeader>
						<CardTitle>Existing Gallery</CardTitle>
						<CardDescription>
							{existingImages.length} file(s) in {selectedEventSlug}/gallery
						</CardDescription>
					</CardHeader>
					<CardContent>
						{loadingImages ? (
							<div className="flex justify-center py-8">
								<Loader2 className="w-8 h-8 animate-spin" />
							</div>
						) : existingImages.length === 0 ? (
							<div className="text-center py-8 text-muted-foreground">
								<ImageIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
								<p>No files in gallery yet</p>
							</div>
						) : (
							<div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
								{existingImages.map((file) => (
									<div key={file.key} className="relative group">
										<div className="aspect-square rounded-lg overflow-hidden bg-muted">
											{file.type === 'image' ? (
												<Image
													src={file.url}
													alt={file.key}
													className="w-full h-full object-cover"
												/>
											) : (
												<div className="w-full h-full flex items-center justify-center">
													<Video className="w-8 h-8 text-muted-foreground" />
												</div>
											)}
										</div>
										<button
											onClick={() => handleDelete(file.key)}
											className="absolute top-2 right-2 p-1.5 bg-destructive text-destructive-foreground rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
											disabled={deleteMutation.isPending}
										>
											<Trash2 className="w-3 h-3" />
										</button>
										<p className="text-xs mt-1 truncate" title={file.key}>
											{file.key.split('/').pop()}
										</p>
									</div>
								))}
							</div>
						)}
					</CardContent>
				</Card>
			)}
		</div>
	);
}
