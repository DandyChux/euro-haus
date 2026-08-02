<script lang="ts">
	import { onMount } from "svelte";
	import { toast } from "svelte-sonner";

	import apiClient, { ApiError } from "$lib/api";
	import type { MediaFile } from "$lib/schemas/media";

	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";
	import { Progress } from "$lib/components/ui/progress";

	interface EventFolder {
		name: string;
		path: string;
	}

	interface Props {
		currentEventSlug?: string;
	}

	interface MediaResponse {
		files?: MediaFile[];
	}

	interface FolderResponse {
		folders?: EventFolder[];
	}

	interface UploadResponse {
		success?: boolean;
		files?: MediaFile[];
		totalUploaded?: number;
		totalFailed?: number;
		errors?: string[];
		message?: string;
	}

	let { currentEventSlug }: Props = $props();

	let eventFolders = $state<EventFolder[]>([]);
	let selectedEventSlug = $state("");

	let selectedFiles = $state<File[]>([]);
	let previews = $state<string[]>([]);

	let existingFiles = $state<MediaFile[]>([]);

	let loadingFolders = $state(true);
	let loadingFiles = $state(false);
	let uploading = $state(false);

	let deletingKey = $state<string | null>(null);
	let uploadProgress = $state(0);

	let fileInput: HTMLInputElement;

	onMount(() => {
		selectedEventSlug = currentEventSlug ?? "";
		void loadFolders();
	});

	$effect(() => {
		if (selectedEventSlug) {
			void loadExistingFiles(selectedEventSlug);
		} else {
			existingFiles = [];
		}
	});

	async function loadFolders() {
		loadingFolders = true;

		try {
			const response = await apiClient.get<FolderResponse>(
				"/admin/events/folders",
			);

			eventFolders = response.folders ?? [];
		} catch (error) {
			console.error("Failed to load event folders:", error);
			toast.error("Unable to load event folders.");
		} finally {
			loadingFolders = false;
		}
	}

	async function loadExistingFiles(slug: string) {
		loadingFiles = true;

		try {
			const response = await apiClient.get<MediaResponse>("/media");

			const galleryPrefix = `events/${slug}/gallery/`;

			existingFiles = (response.files ?? []).filter((file) =>
				file.key.startsWith(galleryPrefix),
			);
		} catch (error) {
			console.error("Failed to load event gallery:", error);
			toast.error("Unable to load the event gallery.");
		} finally {
			loadingFiles = false;
		}
	}

	function addFiles(files: FileList | File[]) {
		const incomingFiles = Array.from(files);

		const validFiles = incomingFiles.filter(
			(file) =>
				file.type.startsWith("image/") ||
				file.type.startsWith("video/"),
		);

		if (validFiles.length !== incomingFiles.length) {
			toast.error(
				"Some files were skipped. Only images and videos are allowed.",
			);
		}

		for (const file of validFiles) {
			selectedFiles.push(file);
			previews.push(URL.createObjectURL(file));
		}
	}

	function handleFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;

		if (input.files) {
			addFiles(input.files);
		}

		// Allows selecting the same file again later.
		input.value = "";
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();

		if (event.dataTransfer?.files) {
			addFiles(event.dataTransfer.files);
		}
	}

	function removeSelectedFile(index: number) {
		const preview = previews[index];

		if (preview) {
			URL.revokeObjectURL(preview);
		}

		selectedFiles = selectedFiles.filter(
			(_, fileIndex) => fileIndex !== index,
		);

		previews = previews.filter((_, fileIndex) => fileIndex !== index);
	}

	function clearSelectedFiles() {
		for (const preview of previews) {
			URL.revokeObjectURL(preview);
		}

		selectedFiles = [];
		previews = [];
	}

	async function uploadFiles() {
		if (!selectedEventSlug) {
			toast.error("Please select an event folder.");
			return;
		}

		if (selectedFiles.length === 0) {
			toast.error("Please select files to upload.");
			return;
		}

		uploading = true;
		uploadProgress = 0;

		try {
			const body = new FormData();

			body.append("eventSlug", selectedEventSlug);

			for (const file of selectedFiles) {
				// The Go batch endpoint expects repeated "files" fields.
				body.append("files", file, file.name);
			}

			const response = await apiClient.upload<UploadResponse>(
				"/admin/events/gallery/upload",
				body,
				({ percent }) => {
					uploadProgress = percent;
				},
			);

			const uploadedCount =
				response.totalUploaded ?? response.files?.length ?? 0;

			const failedCount = response.totalFailed ?? 0;

			if (failedCount > 0) {
				toast.warning(
					`Uploaded ${uploadedCount} file(s); ${failedCount} failed.`,
				);
			} else {
				toast.success(`Uploaded ${uploadedCount} file(s).`);
			}

			clearSelectedFiles();
			await loadExistingFiles(selectedEventSlug);
		} catch (error) {
			console.error("Failed to upload event gallery files:", error);

			if (error instanceof ApiError && error.status === 0) {
				toast.error(error.message);
			} else {
				toast.error("Failed to upload files.");
			}
		} finally {
			uploading = false;
		}
	}

	async function deleteFile(file: MediaFile) {
		const filename = file.key.split("/").pop() ?? "this file";

		if (!confirm(`Delete ${filename}?`)) {
			return;
		}

		deletingKey = file.key;

		try {
			await apiClient.delete("/admin/media/delete", {
				key: file.key,
			});

			toast.success("File deleted.");

			await loadExistingFiles(selectedEventSlug);
		} catch (error) {
			console.error("Failed to delete gallery file:", error);
			toast.error("Failed to delete file.");
		} finally {
			deletingKey = null;
		}
	}
</script>

<div class="space-y-6">
	<Card class="space-y-4 p-5">
		<div>
			<h2 class="text-lg font-semibold">Upload media</h2>

			<p class="text-sm text-muted-foreground">
				Select an event folder and upload images or videos to its
				gallery.
			</p>
		</div>

		<label class="space-y-2 text-sm font-medium">
			<span>Event folder</span>

			<select
				class="w-full rounded-md border bg-background px-3 py-2"
				bind:value={selectedEventSlug}
				disabled={loadingFolders || uploading}
			>
				<option value="">Choose an event…</option>

				{#each eventFolders as folder (folder.name)}
					<option value={folder.name}>
						{folder.name}
					</option>
				{/each}
			</select>
		</label>

		<div
			role="button"
			tabindex="0"
			class="cursor-pointer rounded-lg border-2 border-dashed p-8 text-center transition-colors hover:border-primary"
			onclick={() => fileInput?.click()}
			onkeydown={(event) => {
				if (event.key === "Enter" || event.key === " ") {
					fileInput?.click();
				}
			}}
			ondragover={(event) => event.preventDefault()}
			ondrop={handleDrop}
		>
			<p class="text-3xl" aria-hidden="true">↥</p>

			<p class="text-sm text-muted-foreground">
				Click to upload or drag and drop
			</p>

			<p class="mt-1 text-xs text-muted-foreground">
				Images and videos only
			</p>
		</div>

		<input
			bind:this={fileInput}
			type="file"
			multiple
			accept="image/*,video/*"
			class="hidden"
			onchange={handleFileChange}
			disabled={uploading}
		/>

		{#if selectedFiles.length > 0}
			<div class="space-y-3">
				<div class="flex items-center justify-between">
					<h3 class="text-sm font-medium">
						Selected files ({selectedFiles.length})
					</h3>

					<Button
						type="button"
						variant="ghost"
						size="sm"
						onclick={clearSelectedFiles}
						disabled={uploading}
					>
						Clear all
					</Button>
				</div>

				<div class="grid grid-cols-2 gap-4 md:grid-cols-4">
					{#each selectedFiles as file, index (file.name + file.lastModified + index)}
						<div class="group relative min-w-0">
							<div
								class="aspect-square overflow-hidden rounded-lg bg-muted"
							>
								{#if file.type.startsWith("image/")}
									<img
										src={previews[index]}
										alt={file.name}
										class="h-full w-full object-cover"
									/>
								{:else}
									<div
										class="flex h-full items-center justify-center text-3xl"
										aria-label="Video file"
									>
										▶
									</div>
								{/if}
							</div>

							<button
								type="button"
								class="absolute right-2 top-2 rounded-full bg-destructive px-2 py-1 text-xs text-destructive-foreground"
								onclick={() => removeSelectedFile(index)}
								disabled={uploading}
								aria-label={`Remove ${file.name}`}
							>
								×
							</button>

							<p class="mt-1 truncate text-xs" title={file.name}>
								{file.name}
							</p>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if uploading}
			<div class="space-y-2">
				<div class="flex justify-between text-sm">
					<span>Uploading…</span>
					<span>{uploadProgress}%</span>
				</div>

				<Progress value={uploadProgress} />
			</div>
		{/if}

		<Button
			type="button"
			class="w-full"
			onclick={uploadFiles}
			disabled={!selectedEventSlug ||
				selectedFiles.length === 0 ||
				uploading}
		>
			{uploading
				? "Uploading…"
				: `Upload ${
						selectedFiles.length
							? `${selectedFiles.length} file(s)`
							: "files"
					}`}
		</Button>
	</Card>

	{#if selectedEventSlug}
		<Card class="space-y-4 p-5">
			<div>
				<h2 class="text-lg font-semibold">Existing gallery</h2>

				<p class="text-sm text-muted-foreground">
					{existingFiles.length} file(s) in
					{selectedEventSlug}/gallery
				</p>
			</div>

			{#if loadingFiles}
				<p class="py-8 text-center text-sm text-muted-foreground">
					Loading gallery…
				</p>
			{:else if existingFiles.length === 0}
				<p class="py-8 text-center text-sm text-muted-foreground">
					No files in this gallery yet.
				</p>
			{:else}
				<div
					class="grid grid-cols-2 gap-4 md:grid-cols-4 lg:grid-cols-6"
				>
					{#each existingFiles as file (file.key)}
						<div class="group relative min-w-0">
							<div
								class="aspect-square overflow-hidden rounded-lg bg-muted"
							>
								{#if file.type === "image"}
									<img
										src={file.url}
										alt={file.key}
										class="h-full w-full object-cover"
										loading="lazy"
									/>
								{:else if file.type === "video"}
									<video
										src={file.url}
										class="h-full w-full object-cover"
										controls
										preload="metadata"
									>
										<track kind="captions" />
									</video>
								{:else}
									<div
										class="flex h-full items-center justify-center text-3xl"
										aria-label="Other media file"
									>
										📄
									</div>
								{/if}
							</div>

							<button
								type="button"
								class="absolute right-2 top-2 rounded-full bg-destructive px-2 py-1 text-xs text-destructive-foreground"
								onclick={() => deleteFile(file)}
								disabled={deletingKey !== null}
								aria-label={`Delete ${file.key}`}
							>
								×
							</button>

							<p class="mt-1 truncate text-xs" title={file.key}>
								{file.key.split("/").pop()}
							</p>
						</div>
					{/each}
				</div>
			{/if}
		</Card>
	{/if}
</div>
