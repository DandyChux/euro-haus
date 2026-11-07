<script lang="ts">
	import { onMount } from "svelte";
	import { toast } from "svelte-sonner";
	import apiClient from "$lib/api";
	import { Button } from "$lib/components/ui/button";
	import { Input } from "$lib/components/ui/input";
	import { Card } from "$lib/components/ui/card";
	import { Badge } from "$lib/components/ui/badge";
	import { formatDate } from "$lib/utils";

	interface EmailJobItem {
		id: number;
		submission_id?: string;
		email_type: string;
		recipient: string;
		status: string;
		attempts: number;
		last_error?: string;
		subject?: string;
		created_at: string;
		sent_at?: string;
	}

	interface EmailJobDetail extends EmailJobItem {
		message: {
			to: string[];
			subject: string;
			body_html: string;
			body_text: string;
			template_id?: string;
		};
	}

	let jobs = $state<EmailJobItem[]>([]);
	let loading = $state(false);
	let loadingMore = $state(false);
	let statusFilter = $state<string>("all");
	let searchTerm = $state("");

	// Pagination & lazy loading state
	let currentPage = $state(1);
	const pageSize = 25;
	let totalJobs = $state(0);
	let hasMore = $state(false);

	let selectedJob = $state<EmailJobDetail | null>(null);
	let loadingDetail = $state(false);
	let showModal = $state(false);

	let editRecipient = $state("");
	let editSubject = $state("");
	let editBodyHTML = $state("");
	let updateSubmission = $state(true);
	let previewMode = $state<"edit" | "preview">("edit");
	let resending = $state(false);

	let searchTimeout: ReturnType<typeof setTimeout>;

	async function loadEmails(page = 1, append = false) {
		if (append) {
			loadingMore = true;
		} else {
			loading = true;
		}

		try {
			const params = new URLSearchParams({
				page: String(page),
				limit: String(pageSize),
			});

			if (statusFilter !== "all") params.set("status", statusFilter);
			if (searchTerm.trim()) params.set("search", searchTerm.trim());

			const res = await apiClient.get<{
				jobs: EmailJobItem[];
				total?: number;
				has_more?: boolean;
			}>(`/admin/emails?${params.toString()}`);

			const newJobs = res.jobs ?? [];
			totalJobs =
				res.total ??
				(append ? jobs.length + newJobs.length : newJobs.length);
			hasMore = res.has_more ?? newJobs.length === pageSize;
			currentPage = page;

			if (append) {
				jobs = [...jobs, ...newJobs];
			} else {
				jobs = newJobs;
			}
		} catch (err) {
			console.error("Failed to load email jobs:", err);
			toast.error("Failed to load email jobs");
		} finally {
			loading = false;
			loadingMore = false;
		}
	}

	function handleSearchInput() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			void loadEmails(1, false);
		}, 300);
	}

	function loadNextPage() {
		if (hasMore && !loadingMore) {
			void loadEmails(currentPage + 1, true);
		}
	}

	async function openResendModal(job: EmailJobItem) {
		loadingDetail = true;
		showModal = true;
		previewMode = "edit";
		try {
			const res = await apiClient.get<EmailJobDetail>(
				`/admin/emails/${job.id}`,
			);
			selectedJob = res;
			editRecipient = res.recipient;
			editSubject = res.message?.subject ?? res.subject ?? "";
			editBodyHTML = res.message?.body_html ?? "";
			updateSubmission = Boolean(res.submission_id);
		} catch (err) {
			console.error(err);
			toast.error("Failed to load email details");
			showModal = false;
		} finally {
			loadingDetail = false;
		}
	}

	async function submitResend() {
		if (!selectedJob) return;
		if (!editRecipient.trim()) {
			toast.error("Recipient email is required");
			return;
		}

		resending = true;
		try {
			await apiClient.post(`/admin/emails/${selectedJob.id}/resend`, {
				recipient: editRecipient.trim(),
				subject: editSubject.trim(),
				body_html: editBodyHTML,
				update_submission: updateSubmission,
			});
			toast.success("New email row created and queued!");
			showModal = false;
			await loadEmails(1, false);
		} catch (err) {
			toast.error(
				err instanceof Error ? err.message : "Failed to queue resend",
			);
		} finally {
			resending = false;
		}
	}

	onMount(() => {
		void loadEmails(1, false);
	});
</script>

<svelte:head>
	<title>Admin Emails · Euro Haus</title>
</svelte:head>

<div class="space-y-6">
	<header
		class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
	>
		<div>
			<p class="text-sm uppercase tracking-[0.3em] text-muted">Admin</p>
			<h1 class="mt-2 text-3xl font-semibold">Email Queue & Resend</h1>
			<p class="mt-2 text-sm text-muted">
				Search, inspect, and re-queue all system emails from the queue.
			</p>
		</div>

		<div class="flex items-center gap-3">
			<span class="text-xs text-muted">
				Total records: <strong class="text-foreground"
					>{totalJobs}</strong
				>
			</span>

			<Button
				onclick={() => void loadEmails(1, false)}
				disabled={loading}
				variant="outline"
			>
				{loading ? "Refreshing…" : "Refresh"}
			</Button>
		</div>
	</header>

	<div class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex flex-wrap gap-2">
			{#each ["all", "sent", "failed", "pending", "processing"] as status (status)}
				<Button
					type="button"
					variant={statusFilter === status ? "default" : "outline"}
					class="capitalize"
					onclick={() => {
						statusFilter = status;
						void loadEmails(1, false);
					}}
				>
					{status}
				</Button>
			{/each}
		</div>

		<div class="flex w-full max-w-sm gap-2">
			<Input
				placeholder="Search recipient, submission ID, subject, content…"
				bind:value={searchTerm}
				oninput={handleSearchInput}
				onkeydown={(e) => {
					if (e.key === "Enter") void loadEmails(1, false);
				}}
			/>
			<Button
				onclick={() => void loadEmails(1, false)}
				variant="secondary">Search</Button
			>
		</div>
	</div>

	{#if jobs.length === 0}
		<Card class="p-8 text-center text-muted">
			{loading ? "Loading email records…" : "No email jobs found."}
		</Card>
	{:else}
		<div class="grid gap-3">
			{#each jobs as job (job.id)}
				<Card
					class="flex flex-col justify-between gap-4 p-4 md:flex-row md:items-center"
				>
					<div class="space-y-1">
						<div class="flex items-center gap-2">
							<Badge
								variant={job.status === "sent"
									? "default"
									: job.status === "failed"
										? "destructive"
										: "outline"}
								class="text-xs uppercase"
							>
								{job.status}
							</Badge>
							<span class="font-medium">{job.recipient}</span>
							<span class="text-xs text-muted">#{job.id}</span>
						</div>

						<p class="text-sm text-muted">
							{job.subject || job.email_type}
						</p>

						{#if job.last_error}
							<p class="text-xs text-destructive">
								Error: {job.last_error}
							</p>
						{/if}

						<p class="text-xs text-muted">
							Created: {formatDate(job.created_at, {
								dateStyle: "short",
								timeStyle: "short",
							})}
							{#if job.submission_id}
								· Submission: <span class="font-mono"
									>{job.submission_id.slice(0, 8)}…</span
								>
							{/if}
						</p>
					</div>

					<Button
						type="button"
						variant="outline"
						onclick={() => openResendModal(job)}
					>
						Edit & Resend
					</Button>
				</Card>
			{/each}
		</div>

		<!-- Pagination / Lazy Loading Footer -->
		<div
			class="flex flex-col items-center justify-between gap-4 border-t pt-4 sm:flex-row"
		>
			<p class="text-xs text-muted">
				Showing <strong class="text-foreground">{jobs.length}</strong>
				of <strong class="text-foreground">{totalJobs}</strong> jobs
			</p>

			{#if hasMore}
				<Button
					variant="outline"
					disabled={loadingMore}
					onclick={loadNextPage}
				>
					{loadingMore ? "Loading more…" : "Load more emails"}
				</Button>
			{/if}
		</div>
	{/if}
</div>

{#if showModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
	>
		<Card
			class="max-h-[90vh] w-full max-w-2xl overflow-y-auto space-y-4 p-6"
		>
			<div class="flex items-start justify-between border-b pb-3">
				<div>
					<h2 class="text-lg font-semibold">
						Resend Email #{selectedJob?.id}
					</h2>
					<p class="text-xs text-muted">
						Replicates all columns, creates a fresh pending row, and
						commits for the worker.
					</p>
				</div>
				<Button
					type="button"
					variant="ghost"
					onclick={() => (showModal = false)}>✕</Button
				>
			</div>

			{#if loadingDetail}
				<p class="py-8 text-center text-muted">
					Loading email content…
				</p>
			{:else if selectedJob}
				<div class="space-y-4">
					<div>
						<label
							for="email-recipient"
							class="text-xs font-semibold uppercase text-muted"
							>Recipient Email</label
						>
						<Input
							id="email-recipient"
							bind:value={editRecipient}
							class="mt-1"
							placeholder="participant@example.com"
						/>
					</div>

					<div>
						<label
							for="email-subject"
							class="text-xs font-semibold uppercase text-muted"
							>Subject</label
						>
						<Input
							id="email-subject"
							bind:value={editSubject}
							class="mt-1"
							placeholder="Email subject"
						/>
					</div>

					{#if selectedJob.submission_id}
						<label class="flex items-center gap-2 text-sm">
							<input
								type="checkbox"
								bind:checked={updateSubmission}
								class="rounded"
							/>
							<span
								>Also update email on linked submission & ticket</span
							>
						</label>
					{/if}

					<div>
						<div class="flex items-center justify-between mb-1">
							<label
								for="email-body-html"
								class="text-xs font-semibold uppercase text-muted"
								>Email HTML Body</label
							>
							<div class="flex gap-1 text-xs">
								<button
									type="button"
									class="px-2 py-0.5 rounded {previewMode ===
									'edit'
										? 'bg-primary text-primary-foreground'
										: 'bg-muted'}"
									onclick={() => (previewMode = "edit")}
								>
									Source
								</button>
								<button
									type="button"
									class="px-2 py-0.5 rounded {previewMode ===
									'preview'
										? 'bg-primary text-primary-foreground'
										: 'bg-muted'}"
									onclick={() => (previewMode = "preview")}
								>
									Preview
								</button>
							</div>
						</div>

						{#if previewMode === "edit"}
							<textarea
								id="email-body-html"
								bind:value={editBodyHTML}
								rows={10}
								class="w-full rounded-md border bg-black/20 p-2 font-mono text-xs focus:outline-none"
							></textarea>
						{:else}
							<div
								class="max-h-80 overflow-y-auto rounded-md border bg-white p-4 text-black"
							>
								{@html editBodyHTML}
							</div>
						{/if}
					</div>

					<div class="flex justify-end gap-2 border-t pt-4">
						<Button
							type="button"
							variant="outline"
							onclick={() => (showModal = false)}
						>
							Cancel
						</Button>
						<Button onclick={submitResend} disabled={resending}>
							{resending
								? "Duplicating & Queueing…"
								: "Queue & Send"}
						</Button>
					</div>
				</div>
			{/if}
		</Card>
	</div>
{/if}
