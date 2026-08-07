<script lang="ts">
	import { onMount } from "svelte";
	import { apiClient } from "$lib/api";
	import { formatDate } from "$lib/utils";
	import type { VehicleSubmission } from "$lib/types";
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();

	let selectedEventId = $state("");
	let submissions = $state<VehicleSubmission[]>([]);
	let isLoading = $state(false);
	let errorMessage = $state("");
	let actionMessage = $state("");
	let activeEvent = $derived.by(
		() => data.events.find((event) => event.id === selectedEventId) ?? null,
	);

	async function loadSubmissions() {
		if (!selectedEventId) {
			submissions = [];
			return;
		}

		isLoading = true;
		errorMessage = "";

		try {
			const response = await apiClient.get<{
				submissions?: VehicleSubmission[];
			}>(`/admin/submissions/${selectedEventId}`);
			submissions = response.submissions ?? [];
		} catch (error) {
			submissions = [];
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to load submissions.";
		} finally {
			isLoading = false;
		}
	}

	async function approveSubmission(id: string) {
		const notes = window.prompt("Optional approval notes", "") ?? "";

		try {
			await apiClient.put(`/admin/submissions/${id}/approve`, { notes });
			actionMessage = "Submission approved.";
			await loadSubmissions();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to approve submission.";
		}
	}

	async function denySubmission(id: string) {
		const notes = window.prompt("Reason for denial");
		if (!notes) return;

		try {
			await apiClient.put(`/admin/submissions/${id}/deny`, { notes });
			actionMessage = "Submission denied.";
			await loadSubmissions();
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to deny submission.";
		}
	}

	onMount(() => {
		if (data.events[0]) {
			selectedEventId = data.events[0].id;
		}
		void loadSubmissions();
	});
</script>

<svelte:head>
	<title>Admin submissions · Euro Haus</title>
</svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Vehicle submissions</h1>
		<p class="mt-3 max-w-3xl text-sm leading-7">
			Review participant vehicle submissions for each event.
		</p>
	</div>
</header>

<section class="space-y-6">
	<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
		<label class="block text-sm">
			Event
			<select
				bind:value={selectedEventId}
				class="mt-2 w-full rounded-2xl border border-white/10 bg-black/20 px-4 py-3"
				onchange={() => void loadSubmissions()}
			>
				{#each data.events as event (event.id)}
					<option value={event.id}>{event.name}</option>
				{/each}
			</select>
		</label>
	</div>

	{#if activeEvent}
		<div class="grid gap-4 md:grid-cols-3">
			<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<p class="text-sm">Event</p>
				<p class="mt-2 font-medium">
					{activeEvent.name}
				</p>
			</div>
			<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<p class="text-sm">Date</p>
				<p class="mt-2 font-medium">
					{formatDate(activeEvent.date)}
				</p>
			</div>
			<div class="rounded-3xl border border-white/10 bg-white/5 p-5">
				<p class="text-sm">Count</p>
				<p class="mt-2 font-medium">
					{submissions.length} submissions
				</p>
			</div>
		</div>
	{/if}

	{#if actionMessage}
		<p
			class="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100"
		>
			{actionMessage}
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
			Loading submissions…
		</div>
	{:else if submissions.length === 0}
		<div
			class="rounded-3xl border border-dashed border-white/10 bg-white/5 p-8 text-sm"
		>
			No submissions found for this event.
		</div>
	{:else}
		<div class="space-y-4">
			{#each submissions as submission (submission.id)}
				<article
					class="rounded-3xl border border-white/10 bg-white/5 p-5"
				>
					<div
						class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"
					>
						<div>
							<p class="text-lg font-medium">
								{submission.vehicle_year}
								{submission.vehicle_make}
								{submission.vehicle_model}
							</p>
							<p class="mt-1 text-sm">
								{submission.participant_name} · {submission.participant_email}
							</p>
							<p class="mt-3 text-xs uppercase tracking-[0.2em]">
								Status: {submission.status}
							</p>
							{#if submission.review_notes}
								<p class="mt-3 text-sm">
									{submission.review_notes}
								</p>
							{/if}
						</div>

						<div class="flex flex-wrap gap-3">
							<button
								class="rounded-full border border-white/10 px-4 py-2 text-sm"
								onclick={() =>
									void approveSubmission(submission.id)}
							>
								Approve
							</button>
							<button
								class="rounded-full border border-destructive/30 px-4 py-2 text-sm text-destructive"
								onclick={() =>
									void denySubmission(submission.id)}
							>
								Deny
							</button>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
