import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState, useRef, useEffect } from 'react';
import { Button } from '~/components/ui/button';
import { Input } from '~/components/ui/input';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '~/components/ui/card';
import { Badge } from '~/components/ui/badge';
import { Skeleton } from '~/components/ui/skeleton';
import { toast } from 'sonner';
import { apiClient, apiRequest } from '~/lib/api';
import {
	Upload,
	Loader2,
	LogOut,
	Search,
	ImageIcon,
	VideoIcon,
	Copy,
	FolderOpen,
	X,
	CheckCircle,
	AlertCircle,
	RefreshCw,
	Grid,
	List,
	LayoutDashboard,
	FileText,
	Settings
} from 'lucide-react';
import { ProtectedRoute } from '~/components/protected-route';
import { useAuth } from '~/lib/contexts/auth-context';
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from '~/components/ui/select';
import {
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
} from '~/components/ui/tabs';
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from '~/components/ui/alert-dialog';
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from '~/components/ui/dialog';
import { Progress } from '~/components/ui/progress';
import { ScrollArea } from '~/components/ui/scroll-area';
import { Label } from '~/components/ui/label';
import { Image } from '~/components/ui/image';
import { useContentPlacements, useUpdateContentPlacement } from '~/lib/hooks/use-content-placement';
import type { ContentPlacement } from '~/lib/schemas/content-placement-schema';
import { MediaCard } from '~/components/media-card';
import { MediaListItem } from '~/components/media-list-item';
import { useQueryClient } from '@tanstack/react-query';
import { Video } from '~/components/ui/video';
import { TextContentEditor } from '~/components/ui/text-content-editor';

export interface MediaFile {
	key: string;
	url: string;
	lastModified: string;
	size: number;
	type: 'image' | 'video' | 'other';
	folder: string;
}

interface UploadFile {
	file: File;
	progress: number;
	status: 'pending' | 'uploading' | 'completed' | 'error';
	error?: string;
}

export const Route = createFileRoute('/admin/media')({
	component: AdminMediaPage,
});

function AdminMediaPage() {
	return (
		<ProtectedRoute>
			<AdminMediaContent />
		</ProtectedRoute>
	);
}

function AdminMediaContent() {
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const { logout } = useAuth();
	const fileInputRef = useRef<HTMLInputElement>(null);

	const [activeTab, setActiveTab] = useState('browse');
	const [mediaFiles, setMediaFiles] = useState<MediaFile[]>([]);
	const [filteredFiles, setFilteredFiles] = useState<MediaFile[]>([]);
	const [isRefreshing, setIsRefreshing] = useState(false);
	const [searchQuery, setSearchQuery] = useState('');
	const [filterType, setFilterType] = useState<'all' | 'image' | 'video'>('all');
	const [filterFolder, setFilterFolder] = useState<'all' | string>('all');
	const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
	const [selectedFile, setSelectedFile] = useState<MediaFile | null>(null);
	const [fileToDelete, setFileToDelete] = useState<MediaFile | null>(null);
	const [isDeleting, setIsDeleting] = useState(false);
	const [uploadFiles, setUploadFiles] = useState<UploadFile[]>([]);
	const [isUploading, setIsUploading] = useState(false);
	const [folders, setFolders] = useState<string[]>(['images', 'videos', 'products', 'events']);
	const [selectedPlacement, setSelectedPlacement] = useState<ContentPlacement | null>(null);
	const [editingTextPlacement, setEditingTextPlacement] = useState<ContentPlacement | null>(null);

	// Content placement hooks
	const { data: contentPlacements, isLoading: isLoadingPlacements } = useContentPlacements();
	console.log(contentPlacements)
	const updatePlacement = useUpdateContentPlacement();

	const handleEditMedia = (placement: ContentPlacement) => {
		setSelectedPlacement(placement);
		setActiveTab('browse');
		const newUrl = new URL(window.location.href);
		newUrl.searchParams.set('mode', 'select');
		newUrl.searchParams.set('placementId', placement.id);
		window.history.pushState({}, '', newUrl.toString());
	}

	const handleEditTextContent = (placement: ContentPlacement) => {
		setEditingTextPlacement(placement);
	};

	const handleTextEditComplete = () => {
		setEditingTextPlacement(null);
		queryClient.invalidateQueries({ queryKey: ['content-placements'] });
	};

	// Fetch media files from backend
	const fetchMediaFiles = async () => {
		setIsRefreshing(true);
		try {
			const response = await apiRequest<{ files: MediaFile[], total: number }>({
				method: 'GET',
				url: '/admin/media'
			});
			const files = response.files || [];
			setMediaFiles(files);
			filterMediaFiles(files, searchQuery, filterType, filterFolder);

			// Extract unique folders
			const uniqueFolders = [...new Set(files.map((f: MediaFile) => f.folder).filter(Boolean))];
			if (uniqueFolders.length > 0) {
				setFolders(uniqueFolders);
			}
		} catch (error) {
			console.error('Error fetching media:', error);
			toast.error('Failed to fetch media files');
		} finally {
			setIsRefreshing(false);
		}
	};

	// Filter media files
	const filterMediaFiles = (files: MediaFile[], query: string, type: string, folder: string) => {
		let filtered = files;

		// Filter by type
		if (type !== 'all') {
			filtered = filtered.filter(f => f.type === type);
		}

		// Filter by folder
		if (folder !== 'all') {
			filtered = filtered.filter(f => f.folder === folder);
		}

		// Filter by search query
		if (query) {
			filtered = filtered.filter(f =>
				f.key.toLowerCase().includes(query.toLowerCase())
			);
		}

		setFilteredFiles(filtered);
	};

	// Handle file selection
	const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
		const files = Array.from(event.target.files || []);
		const newUploadFiles: UploadFile[] = files.map(file => ({
			file,
			progress: 0,
			status: 'pending' as const
		}));
		setUploadFiles(prev => [...prev, ...newUploadFiles]);
		setActiveTab('upload');
	};

	// Upload single file
	const uploadFile = async (uploadFile: UploadFile, folder: string) => {
		const formData = new FormData();
		formData.append('file', uploadFile.file);
		formData.append('folder', folder);

		return new Promise((resolve, reject) => {
			const xhr = new XMLHttpRequest();

			// Update status to uploading
			setUploadFiles(prev => prev.map(f =>
				f === uploadFile ? { ...f, status: 'uploading', progress: 0 } : f
			));

			// Track upload progress
			xhr.upload.addEventListener('progress', (event) => {
				if (event.lengthComputable) {
					const percentComplete = Math.round((event.loaded / event.total) * 100);
					console.log(`Upload progress for ${uploadFile.file.name}: ${percentComplete}%`);

					setUploadFiles(prev => prev.map(f =>
						f === uploadFile ? { ...f, progress: percentComplete } : f
					));
				}
			});

			// Handle successful response
			xhr.addEventListener('load', () => {
				if (xhr.status === 200) {
					try {
						const response = JSON.parse(xhr.responseText);
						console.log('Upload completed:', response);

						// Update status to completed
						setUploadFiles(prev => prev.map(f =>
							f === uploadFile ? { ...f, status: 'completed', progress: 100 } : f
						));

						resolve(response);
					} catch (error) {
						console.error('Failed to parse response:', error);
						setUploadFiles(prev => prev.map(f =>
							f === uploadFile ? { ...f, status: 'error', error: 'Invalid server response' } : f
						));
						reject(new Error('Invalid server response'));
					}
				} else {
					console.error('Upload failed with status:', xhr.status);
					setUploadFiles(prev => prev.map(f =>
						f === uploadFile ? { ...f, status: 'error', error: `Upload failed (${xhr.status})` } : f
					));
					reject(new Error(`Upload failed with status ${xhr.status}`));
				}
			});

			// Handle network errors
			xhr.addEventListener('error', () => {
				console.error('Upload network error');
				setUploadFiles(prev => prev.map(f =>
					f === uploadFile ? { ...f, status: 'error', error: 'Network error' } : f
				));
				reject(new Error('Network error'));
			});

			// Handle upload abort
			xhr.addEventListener('abort', () => {
				console.log('Upload aborted');
				setUploadFiles(prev => prev.map(f =>
					f === uploadFile ? { ...f, status: 'error', error: 'Upload cancelled' } : f
				));
				reject(new Error('Upload cancelled'));
			});

			// Get the API URL
			const apiUrl = import.meta.env.VITE_API_URL || '';
			const uploadUrl = `${apiUrl}/admin/media/upload`;

			// Open the request
			xhr.open('POST', uploadUrl);

			// Set authorization header
			const token = localStorage.getItem('accessToken');
			if (token) {
				xhr.setRequestHeader('Authorization', `Bearer ${token}`);
			}

			// Send the request
			xhr.send(formData);
		});
	};

	// Handle batch upload
	const handleUpload = async (folder: string) => {
		setIsUploading(true);
		const pendingFiles = uploadFiles.filter(f => f.status === 'pending');

		try {
			// Upload files one by one
			for (const file of pendingFiles) {
				await uploadFile(file, folder);
			}

			toast.success(`Successfully uploaded ${pendingFiles.length} files`);
			fetchMediaFiles(); // Refresh the list
		} catch (error) {
			toast.error('Some files failed to upload');
		} finally {
			setIsUploading(false);
		}
	};

	// Delete file
	const handleDeleteFile = async () => {
		if (!fileToDelete) return;

		setIsDeleting(true);
		try {
			await apiClient.delete('/admin/media/delete', {
				data: { key: fileToDelete.key }
			});
			toast.success('File deleted successfully');
			fetchMediaFiles();
			setFileToDelete(null);
		} catch (error) {
			console.error('Error deleting file:', error);
			toast.error('Failed to delete file');
		} finally {
			setIsDeleting(false);
		}
	};

	// Copy URL to clipboard
	const copyToClipboard = (url: string) => {
		navigator.clipboard.writeText(url);
		toast.success('URL copied to clipboard');
	};

	// Format file size
	const formatFileSize = (bytes: number) => {
		if (bytes === 0) return '0 Bytes';
		const k = 1024;
		const sizes = ['Bytes', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	};

	// Get file type icon
	const getFileIcon = (type: string) => {
		return type === 'image' ? ImageIcon : type === 'video' ? VideoIcon : FolderOpen;
	};

	// Handle assigning media to content placement
	const handleAssignToPlacement = async (file: MediaFile) => {
		if (!selectedPlacement) return;

		try {
			await updatePlacement.mutateAsync({
				id: selectedPlacement.id,
				data: {
					mediaUrl: file.url,
					mediaKey: file.key,
				}
			});

			setSelectedPlacement(null);
			setActiveTab('content');
			toast.success(`Updated ${selectedPlacement.name} with new media`);
		} catch (error) {
			console.error('Failed to assign media:', error);
			toast.error('Failed to update content placement');
		}
	};

	// Initialize
	useEffect(() => {
		fetchMediaFiles();
	}, []);

	// Update filters
	useEffect(() => {
		filterMediaFiles(mediaFiles, searchQuery, filterType, filterFolder);
	}, [searchQuery, filterType, filterFolder, mediaFiles]);

	const handleLogout = async () => {
		await logout();
		navigate({ to: '/admin/login' });
	};

	return (
		<div className="min-h-screen bg-background">
			<div className="max-w-7xl mx-auto p-6">
				<div className="flex justify-between items-center mb-8">
					<div>
						<h1 className="text-3xl font-bold">Media Management</h1>
						<p className="text-muted-foreground">Manage images and videos for your site</p>
					</div>
					<div className="flex gap-2">
						<Button variant="outline" onClick={() => navigate({ to: '/admin' })}>
							<LayoutDashboard className="h-4 w-4 mr-2" />
							Dashboard
						</Button>
						<Button variant="outline" onClick={() => navigate({ to: '/admin/products' })}>
							Products
						</Button>
						<Button variant="outline" onClick={() => navigate({ to: '/' })}>
							Back to Site
						</Button>
						<Button variant="ghost" size="icon" onClick={handleLogout} title="Logout">
							<LogOut className="h-4 w-4" />
						</Button>
					</div>
				</div>

				<Tabs value={activeTab} onValueChange={setActiveTab}>
					<TabsList className="mb-8">
						<TabsTrigger value="browse">
							<Grid className="h-4 w-4 mr-2" />
							Browse Media
						</TabsTrigger>
						<TabsTrigger value="upload">
							<Upload className="h-4 w-4 mr-2" />
							Upload Files ({uploadFiles.filter(f => f.status === 'pending').length})
						</TabsTrigger>
						<TabsTrigger value='content'>
							Content Management
						</TabsTrigger>
					</TabsList>

					{/* Browse Media Tab */}
					<TabsContent value="browse">
						<Card>
							<CardHeader>
								{/* Selection Mode Banner */}
								{selectedPlacement && (
									<div className="mb-4 p-4 bg-primary/10 rounded-lg border border-primary/20">
										<div className="flex items-center justify-between">
											<div className="flex items-center gap-2">
												<Settings className="h-5 w-5 text-primary" />
												<div>
													<p className="font-medium">Selection Mode</p>
													<p className="text-sm text-muted-foreground">
														Choose media for: <span className="font-medium">{selectedPlacement.name}</span>
													</p>
												</div>
											</div>
											<Button
												variant="outline"
												size="sm"
												onClick={() => {
													setSelectedPlacement(null);
													navigate({ to: '/admin/media', replace: true });
												}}
											>
												<X className="h-4 w-4 mr-2" />
												Exit Selection
											</Button>
										</div>
									</div>
								)}

								<div className="flex justify-between items-start">
									<div>
										<CardTitle>Media Library</CardTitle>
										<CardDescription>Browse and manage your uploaded files</CardDescription>
									</div>
									<div className="flex gap-2">
										<Button
											variant="outline"
											size="sm"
											onClick={() => setViewMode(viewMode === 'grid' ? 'list' : 'grid')}
										>
											{viewMode === 'grid' ? <List className="h-4 w-4" /> : <Grid className="h-4 w-4" />}
										</Button>
										<Button
											variant="outline"
											size="sm"
											onClick={fetchMediaFiles}
											disabled={isRefreshing}
										>
											{isRefreshing ? (
												<Loader2 className="h-4 w-4 animate-spin" />
											) : (
												<RefreshCw className="h-4 w-4" />
											)}
										</Button>
									</div>
								</div>
							</CardHeader>
							<CardContent>
								{/* Filters */}
								<div className="flex gap-4 mb-6">
									<div className="flex-1">
										<div className="relative">
											<Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
											<Input
												placeholder="Search files..."
												value={searchQuery}
												onChange={(e) => setSearchQuery(e.target.value)}
												className="pl-10"
											/>
										</div>
									</div>
									<Select value={filterType} onValueChange={(value) => setFilterType(value as 'all' | 'image' | 'video')}>
										<SelectTrigger className="w-[150px]">
											<SelectValue placeholder="Filter by type" />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="all">All Files</SelectItem>
											<SelectItem value="image">Images</SelectItem>
											<SelectItem value="video">Videos</SelectItem>
										</SelectContent>
									</Select>
									<Select value={filterFolder} onValueChange={setFilterFolder}>
										<SelectTrigger className="w-[150px]">
											<SelectValue placeholder="Filter by folder" />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="all">All Folders</SelectItem>
											{folders.map(folder => (
												<SelectItem key={folder} value={folder}>{folder}</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>

								{/* Upload button */}
								<div className="mb-6">
									<input
										ref={fileInputRef}
										type="file"
										multiple
										accept="image/*,video/*"
										onChange={handleFileSelect}
										className="hidden"
									/>
									<Button className='cursor-pointer' onClick={() => fileInputRef.current?.click()}>
										<Upload className="h-4 w-4 mr-2" />
										Upload Files
									</Button>
								</div>

								{/* Media Grid/List */}
								{isRefreshing ? (
									<div className={viewMode === 'grid' ? 'grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4' : 'space-y-2'}>
										{[...Array(8)].map((_, i) => (
											<Skeleton key={i} className={viewMode === 'grid' ? 'aspect-square' : 'h-16'} />
										))}
									</div>
								) : filteredFiles.length === 0 ? (
									<div className="text-center py-12">
										<p className="text-muted-foreground mb-4">No media files found</p>
										<Button onClick={() => fileInputRef.current?.click()}>
											<Upload className="h-4 w-4 mr-2" />
											Upload Your First File
										</Button>
									</div>
								) : viewMode === 'grid' ? (
									<div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
										{filteredFiles.map((file) => (
											<MediaCard
												key={file.key}
												file={file}
												onSelect={setSelectedFile}
												onDelete={setFileToDelete}
												contentPlacements={contentPlacements}
												selectedPlacement={selectedPlacement}
												onAssignToPlacement={handleAssignToPlacement}
											/>
										))}
									</div>
								) : (
									<div className="space-y-2">
										{filteredFiles.map((file) => (
											<MediaListItem
												key={file.key}
												file={file}
												onSelect={setSelectedFile}
												onDelete={setFileToDelete}
												contentPlacements={contentPlacements}
												selectedPlacement={selectedPlacement}
												onAssignToPlacement={handleAssignToPlacement}
											/>
										))}
									</div>
								)}
							</CardContent>
						</Card>
					</TabsContent>

					{/* Upload Tab */}
					<TabsContent value="upload">
						<Card>
							<CardHeader>
								<CardTitle>Upload Files</CardTitle>
								<CardDescription>
									Upload images and videos to your media library
								</CardDescription>
							</CardHeader>
							<CardContent>
								{uploadFiles.length === 0 ? (
									<div className="text-center py-12 border-2 border-dashed rounded-lg">
										<Upload className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
										<p className="text-muted-foreground mb-4">No files selected</p>
										<input
											ref={fileInputRef}
											type="file"
											multiple
											accept="image/*,video/*"
											onChange={handleFileSelect}
											className="hidden"
										/>
										<Button className='cursor-pointer' onClick={() => fileInputRef.current?.click()}>
											Select Files
										</Button>
									</div>
								) : (
									<div className="space-y-4">
										<div className="flex justify-between items-center mb-4">
											<p className="text-sm text-muted-foreground">
												{uploadFiles.filter(f => f.status === 'pending').length} files ready to upload
											</p>
											<Button
												variant="outline"
												size="sm"
												onClick={() => setUploadFiles([])}
											>
												Clear All
											</Button>
										</div>

										<ScrollArea className="h-[400px] pr-4">
											<div className="space-y-2">
												{uploadFiles.map((file, index) => (
													<div key={index} className="flex items-center gap-4 p-4 border rounded-lg">
														<div className="flex-1">
															<p className="font-medium">{file.file.name}</p>
															<p className="text-sm text-muted-foreground">
																{formatFileSize(file.file.size)}
															</p>
															{file.status === 'uploading' && (
																<Progress value={file.progress} className="mt-2 h-2" />
															)}
															{file.error && (
																<p className="text-sm text-destructive mt-1">{file.error}</p>
															)}
														</div>
														<div className="flex items-center gap-2">
															{file.status === 'pending' && (
																<Badge variant="outline">Pending</Badge>
															)}
															{file.status === 'uploading' && (
																<Badge variant="secondary">
																	<Loader2 className="h-3 w-3 mr-1 animate-spin" />
																	{file.progress}%
																</Badge>
															)}
															{file.status === 'completed' && (
																<Badge variant="default">
																	<CheckCircle className="h-3 w-3 mr-1" />
																	Uploaded
																</Badge>
															)}
															{file.status === 'error' && (
																<Badge variant="destructive">
																	<AlertCircle className="h-3 w-3 mr-1" />
																	Failed
																</Badge>
															)}
															{file.status === 'pending' && (
																<Button
																	size="sm"
																	variant="ghost"
																	onClick={() => setUploadFiles(prev => prev.filter((_, i) => i !== index))}
																>
																	<X className="h-4 w-4" />
																</Button>
															)}
														</div>
													</div>
												))}
											</div>
										</ScrollArea>

										{uploadFiles.some(f => f.status === 'pending') && (
											<div className="pt-4">
												<Select defaultValue="images">
													<SelectTrigger className="w-full mb-4">
														<SelectValue placeholder="Select upload folder" />
													</SelectTrigger>
													<SelectContent>
														<Label>
															Select Upload Folder
														</Label>
														{folders.map(folder => (
															<SelectItem key={folder} value={folder}>{folder}</SelectItem>
														))}
													</SelectContent>
												</Select>
												<Button
													className="w-full"
													onClick={() => {
														const folder = document.querySelector('[data-value]')?.getAttribute('data-value') || 'images';
														handleUpload(folder);
													}}
													disabled={isUploading}
												>
													{isUploading ? (
														<>
															<Loader2 className="mr-2 h-4 w-4 animate-spin" />
															Uploading...
														</>
													) : (
														<>
															<Upload className="mr-2 h-4 w-4" />
															Upload All Files
														</>
													)}
												</Button>
											</div>
										)}
									</div>
								)}
							</CardContent>
						</Card>
					</TabsContent>

					{/* Content Management Tab */}
					<TabsContent value="content">
						<Card>
							<CardHeader>
								<CardTitle>Content Placements</CardTitle>
								<CardDescription>
									Manage media content used across your site. Media is automatically discovered as you browse pages.
								</CardDescription>
							</CardHeader>
							<CardContent>
								{isLoadingPlacements ? (
									<div className="space-y-4">
										{[...Array(3)].map((_, i) => (
											<Skeleton key={i} className="h-20 w-full" />
										))}
									</div>
								) : (
									<div className="space-y-6">
										{/* Group placements by page */}
										{contentPlacements && contentPlacements.length > 0 ? (
											Object.entries(
												contentPlacements.reduce((acc, placement) => {
													const page = placement.page || 'Other';
													if (!acc[page]) acc[page] = [];
													acc[page].push(placement);
													return acc;
												}, {} as Record<string, typeof contentPlacements>)
											).map(([page, placements]) => (
												<div key={page} className="space-y-4">
													<h3 className="text-lg font-semibold flex items-center gap-2">
														<FileText className="h-5 w-5" />
														{page} Page
													</h3>
													<div className="grid gap-4 pl-6">
														{placements.map((placement) => (
															<Card key={placement.id} className="p-4 border rounded-lg space-y-2 hover:shadow-md transition-shadow">
																<CardHeader className="pb-2">
																	<CardTitle className="text-lg">{placement.name}</CardTitle>
																	<CardDescription>{placement.description}</CardDescription>
																</CardHeader>
																<CardContent>
																	{/* Display content based on type */}
																	{placement.type === 'text' ? (
																		<div className="bg-muted p-4 rounded-md mb-2 relative max-h-48 overflow-auto">
																			{placement.html ? (
																				<div dangerouslySetInnerHTML={{ __html: placement.textContent || placement.defaultText || '' }} />
																			) : (
																				<p>{placement.textContent || placement.defaultText || ''}</p>
																			)}
																		</div>
																	) : placement.type === 'image' ? (
																		<Image
																			src={placement.mediaUrl || ''}
																			alt={placement.name}
																			className="h-32 w-44 object-cover rounded-md mb-2"
																		/>
																	) : (
																		<Video
																			src={placement.mediaUrl}
																			className="h-32 w-44 object-cover rounded"
																			muted
																		/>
																	)}
																</CardContent>
																<CardFooter className='flex-col space-y-4'>
																	<Button
																		onClick={() => placement.type === 'text'
																			? handleEditTextContent(placement)
																			: handleEditMedia(placement)
																		}
																		variant="outline"
																		className="w-full"
																	>
																		<Settings className="h-4 w-4 mr-2" />
																		{placement.type === 'text' ? 'Edit Text' : 'Change Media'}
																	</Button>
																	<p className="text-xs text-muted-foreground">
																		Last updated: {new Date(placement.updatedAt).toLocaleDateString()}
																		{placement.updatedBy && ` by ${placement.updatedBy}`}
																	</p>
																</CardFooter>
															</Card>
														))}
													</div>
												</div>
											))
										) : (
											<div className="text-center py-8 text-muted-foreground">
												<p>No content placements found.</p>
												<p className="text-sm mt-2">
													Browse your site to automatically discover media placements.
												</p>
											</div>
										)}
									</div>
								)}
							</CardContent>
						</Card>
					</TabsContent>

				</Tabs>

				{/* File Preview Dialog */}
				<Dialog open={!!selectedFile} onOpenChange={(open) => !open && setSelectedFile(null)}>
					<DialogContent className="max-w-4xl">
						<DialogHeader>
							<DialogTitle>{selectedFile?.key.split('/').pop()}</DialogTitle>
							<DialogDescription>
								{selectedFile && (
									<div className="flex gap-4 text-sm">
										<span>{formatFileSize(selectedFile.size)}</span>
										<span>{new Date(selectedFile.lastModified).toLocaleDateString()}</span>
										<Badge variant="outline">{selectedFile.folder}</Badge>
									</div>
								)}
							</DialogDescription>
						</DialogHeader>
						<div className="mt-4">
							{selectedFile?.type === 'image' ? (
								<img
									src={selectedFile.url}
									alt={selectedFile.key}
									className="w-full rounded-lg"
								/>
							) : selectedFile?.type === 'video' ? (
								<video
									src={selectedFile.url}
									controls
									className="w-full rounded-lg"
								/>
							) : null}
							<div className="mt-4 flex gap-2">
								<Input value={selectedFile?.url || ''} readOnly />
								<Button
									variant="outline"
									onClick={() => selectedFile && copyToClipboard(selectedFile.url)}
								>
									<Copy className="h-4 w-4" />
								</Button>
							</div>
						</div>
					</DialogContent>
				</Dialog>

				{/* Delete Confirmation Dialog */}
				<AlertDialog open={!!fileToDelete} onOpenChange={(open) => !open && setFileToDelete(null)}>
					<AlertDialogContent>
						<AlertDialogHeader>
							<AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
							<AlertDialogDescription>
								This will permanently delete the file "{fileToDelete?.key.split('/').pop()}".
								This action cannot be undone.
							</AlertDialogDescription>
						</AlertDialogHeader>
						<AlertDialogFooter>
							<AlertDialogCancel>Cancel</AlertDialogCancel>
							<AlertDialogAction
								onClick={handleDeleteFile}
								disabled={isDeleting}
								className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
							>
								{isDeleting ? (
									<>
										<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										Deleting...
									</>
								) : (
									'Delete File'
								)}
							</AlertDialogAction>
						</AlertDialogFooter>
					</AlertDialogContent>
				</AlertDialog>

				<Dialog open={!!editingTextPlacement} onOpenChange={(open) => !open && setEditingTextPlacement(null)}>
					<DialogContent className="sm:max-w-xl">
						<DialogHeader>
							<DialogTitle>Edit Text Content</DialogTitle>
							<DialogDescription>
								Update the text for "{editingTextPlacement?.name}"
							</DialogDescription>
						</DialogHeader>

						<TextContentEditor
							placementId={editingTextPlacement?.id || ''}
							currentText={editingTextPlacement?.textContent || ''}
							defaultText={editingTextPlacement?.defaultText || ''}
							placementName={editingTextPlacement?.name || ''}
							isHtml={editingTextPlacement?.html}
							onSave={handleTextEditComplete}
							onCancel={() => setEditingTextPlacement(null)}
						/>
					</DialogContent>
				</Dialog>
			</div>
		</div>
	);
}
