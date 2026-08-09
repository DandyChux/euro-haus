<script lang="ts">
	import { onMount } from "svelte";
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import type { VehicleSubmission } from "$lib/schemas/event";

	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";
	import { Textarea } from "$lib/components/ui/textarea";
	import { formatDate } from "$lib/utils";

	interface SubmissionsResponse {
		submissions?: VehicleSubmission[];
	}

	interface Props {
		eventName: string;
		submissions: VehicleSubmission[];
		loading?: boolean;
		errorMessage?: string;
		onSubmissionUpdated?: () => void | Promise<void>;
	}

	let {
		eventName,
		submissions = [],
		loading = false,
		errorMessage = "",
		onSubmissionUpdated,
	}: Props = $props();

	let selectedSubmission = $state<VehicleSubmission | null>(null);

	let activeTab = $state<"pending" | "reviewed">("pending");
	let currentImageIndex = $state(0);

	let processing = $state(false);

	let action = $state<"approve" | "deny" | null>(null);
	let actionNotes = $state("");

	const pendingSubmissions = $derived(
		submissions.filter((submission) => submission.status === "pending"),
	);

	const reviewedSubmissions = $derived(
		submissions.filter((submission) => submission.status !== "pending"),
	);

	const visibleSubmissions = $derived(
		activeTab === "pending" ? pendingSubmissions : reviewedSubmissions,
	);

	function openSubmission(submission: VehicleSubmission) {
		selectedSubmission = submission;
		currentImageIndex = 0;
		action = null;
		actionNotes = "";
	}

	function closeSubmission() {
		selectedSubmission = null;
		action = null;
		actionNotes = "";
	}

	function showPreviousImage() {
		if (!selectedSubmission) return;

		currentImageIndex =
			currentImageIndex === 0
				? selectedSubmission.images.length - 1
				: currentImageIndex - 1;
	}

	function showNextImage() {
		if (!selectedSubmission) return;

		currentImageIndex =
			currentImageIndex === selectedSubmission.images.length - 1
				? 0
				: currentImageIndex + 1;
	}

	async function submitAction() {
		if (!selectedSubmission || !action) return;

		if (action === "deny" && !actionNotes.trim()) {
			toast.error("A reason is required when denying a submission.");
			return;
		}

		processing = true;

		try {
			const endpoint =
				action === "approve"
					? `/admin/submissions/${selectedSubmission.id}/approve`
					: `/admin/submissions/${selectedSubmission.id}/deny`;

			await apiClient.put(endpoint, {
				notes: actionNotes.trim(),
			});

			toast.success(
				action === "approve"
					? "Submission approved."
					: "Submission denied.",
			);

			closeSubmission();
			await onSubmissionUpdated?.();
		} catch (error) {
			console.error("Failed to update submission:", error);
			toast.error(
				action === "approve"
					? "Failed to approve submission."
					: "Failed to deny submission.",
			);
		} finally {
			processing = false;
		}
	}
</script>

{#if loading}
	<div
		class="flex min-h-64 items-center justify-center text-muted-foreground"
	>
		Loading submissions…
	</div>
{:else}
	<div class="space-y-6">
		<div>
			<h2 class="text-2xl font-bold">Vehicle submissions</h2>

			<p class="text-muted-foreground">
				Review and approve participant vehicles for {eventName}.
			</p>
		</div>

		{#if errorMessage}
			<p
				role="alert"
				class="rounded-xl border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
			>
				{errorMessage}
			</p>
		{/if}

		<div class="flex gap-2 border-b">
			<Button
				type="button"
				variant={activeTab === "pending" ? "default" : "ghost"}
				onclick={() => (activeTab = "pending")}
			>
				Pending review ({pendingSubmissions.length})
			</Button>

			<Button
				type="button"
				variant={activeTab === "reviewed" ? "default" : "ghost"}
				onclick={() => (activeTab = "reviewed")}
			>
				Reviewed ({reviewedSubmissions.length})
			</Button>
		</div>

		{#if visibleSubmissions.length === 0}
			<Card class="p-8 text-center text-muted-foreground">
				No {activeTab} submissions.
			</Card>
		{:else}
			<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{#each visibleSubmissions as submission (submission.id)}
					<button
						type="button"
						class="text-left"
						onclick={() => openSubmission(submission)}
					>
						<Card
							class="h-full space-y-4 p-4 transition-shadow hover:shadow-lg"
						>
							<div class="flex items-start justify-between gap-3">
								<div>
									<h3 class="font-semibold">
										{submission.vehicle_year}
										{submission.vehicle_make}
										{submission.vehicle_model}
									</h3>

									<p class="text-sm text-muted-foreground">
										{submission.participant_name}
									</p>
								</div>

								<span
									class="rounded-full border px-2 py-1 text-xs capitalize"
								>
									{submission.status}
								</span>
							</div>

							{#if submission.images[0]}
								<img
									src={submission.images[0]}
									alt={`${submission.vehicle_make} ${submission.vehicle_model}`}
									class="aspect-video w-full rounded-lg object-cover"
								/>
							{:else}
								<div
									class="flex aspect-video items-center justify-center rounded-lg bg-muted text-sm text-muted-foreground"
								>
									No vehicle images
								</div>
							{/if}

							<p class="text-sm text-muted-foreground">
								Submitted
								{formatDate(submission.submitted_at, {
									dateStyle: "medium",
								})}
							</p>
						</Card>
					</button>
				{/each}
			</div>
		{/if}
	</div>
{/if}

{#if selectedSubmission}
	<div
		class="fixed inset-0 z-50 overflow-y-auto bg-background/80 p-4 backdrop-blur-sm"
	>
		<div class="mx-auto max-w-4xl">
			<Card class="space-y-6 p-5">
				<div class="flex items-start justify-between gap-4">
					<div>
						<h2 class="text-xl font-semibold">
							{selectedSubmission.vehicle_year}
							{selectedSubmission.vehicle_make}
							{selectedSubmission.vehicle_model}
						</h2>

						<p class="text-sm text-muted-foreground">
							Submitted by
							{selectedSubmission.participant_name}
						</p>
					</div>

					<Button
						type="button"
						variant="ghost"
						onclick={closeSubmission}
					>
						Close
					</Button>
				</div>

				{#if selectedSubmission.images.length > 0}
					<div class="space-y-3">
						<img
							src={selectedSubmission.images[currentImageIndex]}
							alt={`Vehicle image ${currentImageIndex + 1}`}
							class="max-h-[28rem] w-full rounded-lg object-contain"
						/>

						{#if selectedSubmission.images.length > 1}
							<div class="flex items-center justify-center gap-3">
								<Button
									type="button"
									variant="outline"
									onclick={showPreviousImage}
								>
									Previous
								</Button>

								<span class="text-sm">
									{currentImageIndex + 1} /
									{selectedSubmission.images.length}
								</span>

								<Button
									type="button"
									variant="outline"
									onclick={showNextImage}
								>
									Next
								</Button>
							</div>
						{/if}
					</div>
				{/if}

				<div class="grid gap-6 md:grid-cols-2">
					<div>
						<h3 class="font-semibold">Participant</h3>

						<p>{selectedSubmission.participant_name}</p>

						<a
							class="text-sm text-primary hover:underline"
							href={`mailto:${selectedSubmission.participant_email}`}
						>
							{selectedSubmission.participant_email}
						</a>

						{#if selectedSubmission.participant_phone}
							<p>{selectedSubmission.participant_phone}</p>
						{/if}
					</div>

					<div>
						<h3 class="font-semibold">Vehicle</h3>

						<p>
							{selectedSubmission.vehicle_year}
							{selectedSubmission.vehicle_make}
							{selectedSubmission.vehicle_model}
						</p>

						<p class="text-sm text-muted-foreground">
							Submitted
							{formatDate(selectedSubmission.submitted_at, {
								dateStyle: "medium",
							})}
						</p>
					</div>
				</div>

				{#if selectedSubmission.vehicle_description}
					<div>
						<h3 class="font-semibold">Description</h3>

						<p
							class="whitespace-pre-wrap text-sm text-muted-foreground"
						>
							{selectedSubmission.vehicle_description}
						</p>
					</div>
				{/if}

				{#if selectedSubmission.vehicle_modifications}
					<div>
						<h3 class="font-semibold">Modifications</h3>

						<p
							class="whitespace-pre-wrap text-sm text-muted-foreground"
						>
							{selectedSubmission.vehicle_modifications}
						</p>
					</div>
				{/if}

				{#if selectedSubmission.review_notes}
					<div
						class="rounded-xl border border-primary/30 bg-primary/5 p-4"
					>
						<h3 class="font-semibold">Review notes</h3>

						<p class="whitespace-pre-wrap text-sm">
							{selectedSubmission.review_notes}
						</p>
					</div>
				{/if}

				{#if selectedSubmission.status === "pending"}
					{#if action}
						<div class="space-y-3 rounded-xl border p-4">
							<h3 class="font-semibold">
								{action === "approve" ? "Approve" : "Deny"}
								submission
							</h3>

							<p class="text-sm text-muted-foreground">
								{action === "approve"
									? "Approval notes are optional."
									: "A reason is required when denying a submission."}
							</p>

							<Textarea
								bind:value={actionNotes}
								placeholder={action === "approve"
									? "Optional approval notes"
									: "Reason for denial"}
								rows={4}
							/>

							<div class="flex justify-end gap-2">
								<Button
									type="button"
									variant="ghost"
									onclick={() => {
										action = null;
										actionNotes = "";
									}}
									disabled={processing}
								>
									Cancel
								</Button>

								<Button
									type="button"
									variant={action === "deny"
										? "destructive"
										: "default"}
									onclick={submitAction}
									disabled={processing ||
										(action === "deny" &&
											!actionNotes.trim())}
								>
									{processing
										? "Processing…"
										: action === "approve"
											? "Approve"
											: "Deny"}
								</Button>
							</div>
						</div>
					{:else}
						<div class="flex justify-end gap-2">
							<Button
								type="button"
								variant="outline"
								onclick={() => (action = "deny")}
							>
								Deny
							</Button>

							<Button
								type="button"
								onclick={() => (action = "approve")}
							>
								Approve
							</Button>
						</div>
					{/if}
				{/if}
			</Card>
		</div>
	</div>
{/if}
