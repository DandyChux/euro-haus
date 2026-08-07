<script lang="ts">
	import { onMount } from "svelte";
	import { apiClient } from "$lib/api";
	import { formatDate } from "$lib/utils";
	import type { MediaFile } from "$lib/schemas/media";

	let files = $state<MediaFile[]>([]);
	let search = $state("");
	let typeFilter = $state<"all" | "image" | "video" | "other">("all");
	let folderFilter = $state("all");
	let uploadFolder = $state("images");
	let isLoading = $state(true);
	let isUploading = $state(false);
	let statusMessage = $state("");
	let errorMessage = $state("");

	let folders = $derived([
		"all",
		...new Set(files.map((file) => file.folder).filter(Boolean)),
	]);

	let filteredFiles = $derived.by(() => {
		return files.filter((file) => {
			if (typeFilter !== "all" && file.type !== typeFilter) return false;
			if (folderFilter !== "all" && file.folder !== folderFilter)
				return false;
			if (!search.trim()) return true;
			return file.key.toLowerCase().includes(search.toLowerCase());
		});
	});

	async function loadMedia() {
		isLoading = true;
		errorMessage = "";

		try {
			const response = await fetch("/api/media");
			const payload = (await response.json()) as { files?: MediaFile[] };
			files = payload.files ?? [];
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to load media.";
		} finally {
			isLoading = false;
		}
	}

	async function uploadFiles(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const selected = Array.from(input.files ?? []);

		if (!selected.length) return;

		isUploading = true;
		errorMessage = "";
		statusMessage = "";

		try {
			for (const file of selected) {
				const formData = new FormData();
				formData.append("file", file);
				formData.append("folder", uploadFolder);

				await apiClient.post("/admin/media/upload", formData);
			}

			statusMessage = `Uploaded ${selected.length} file(s).`;
			input.value = "";
			await loadMedia();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to upload files.";
		} finally {
			isUploading = false;
		}
	}

	async function deleteFile(file: MediaFile) {
		if (!window.confirm(`Delete ${file.key}?`)) return;

		try {
			await apiClient.delete("/admin/media/delete", {
				body: { key: file.key },
			});
			statusMessage = `Deleted ${file.key}.`;
			await loadMedia();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to delete file.";
		}
	}

	onMount(() => {
		void loadMedia();
	});
</script>

<svelte:head>
	<title>Admin media · Euro Haus</title>
</svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Media library</h1>
		<p class="mt-3 max-w-3xl text-sm leading-7">
			Browse, filter, upload, and delete media files. Content-placement
			editing can be added in a later pass.
		</p>
	</div>
</header>

<section class="space-y-6">
	<div
		class="grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-5 md:grid-cols-[12rem_minmax(0,1fr)]"
	>
		<select
			bind:value={uploadFolder}
			class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		>
			<option value="images">images</option>
			<option value="videos">videos</option>
			<option value="products">products</option>
			<option value="events">events</option>
		</select>

		<label
			class="flex cursor-pointer items-center justify-center rounded-2xl border border-dashed border-white/20 px-4 py-3 text-sm hover:border-white/40"
		>
			<input type="file" multiple class="hidden" onchange={uploadFiles} />
			{isUploading ? "Uploading…" : "Choose files to upload"}
		</label>
	</div>

	<div
		class="grid gap-4 rounded-3xl border border-white/10 bg-white/5 p-5 md:grid-cols-[minmax(0,1fr)_10rem_10rem]"
	>
		<input
			bind:value={search}
			placeholder="Search by key"
			class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		/>

		<select
			bind:value={typeFilter}
			class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		>
			<option value="all">All types</option>
			<option value="image">Images</option>
			<option value="video">Videos</option>
			<option value="other">Other</option>
		</select>

		<select
			bind:value={folderFilter}
			class="rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
		>
			{#each folders as folder (folder)}
				<option value={folder}>{folder}</option>
			{/each}
		</select>
	</div>

	{#if statusMessage}
		<p
			class="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100"
		>
			{statusMessage}
		</p>
	{/if}

	{#if errorMessage}
		<p
			class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
		>
			{errorMessage}
		</p>
	{/if}

	{#if isLoading}
		<div class="rounded-3xl border border-white/10 bg-white/5 p-8 text-sm">
			Loading media…
		</div>
	{:else}
		<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
			{#each filteredFiles as file (file.key)}
				<article
					class="overflow-hidden rounded-3xl border border-white/10 bg-white/5"
				>
					<div class="aspect-4/3 bg-zinc-900">
						{#if file.type === "image"}
							<img
								src={file.url}
								alt={file.key}
								class="h-full w-full object-cover"
							/>
						{:else}
							<div
								class="flex h-full items-center justify-center text-sm"
							>
								{file.type.toUpperCase()}
							</div>
						{/if}
					</div>

					<div class="space-y-3 p-5">
						<p class="line-clamp-2 text-sm">
							{file.key}
						</p>
						<p class="text-xs uppercase tracking-[0.2em]">
							{file.folder} · {file.type}
						</p>
						<p class="text-xs">
							{formatDate(file.last_modified)}
						</p>

						<div class="flex flex-wrap gap-3">
							<a
								href={file.url}
								target="_blank"
								rel="noreferrer"
								class="rounded-full border border-white/10 px-4 py-2 text-sm"
							>
								Open
							</a>
							<button
								class="rounded-full border border-destructive/30 px-4 py-2 text-sm text-destructive"
								onclick={() => void deleteFile(file)}
							>
								Delete
							</button>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
