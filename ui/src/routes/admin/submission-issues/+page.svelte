<script lang="ts">
	import { apiClient } from "$lib/api";
	import { formatDate } from "$lib/utils";
	import * as Item from "$lib/components/ui/item";
	import { invalidateAll } from "$app/navigation";
	import Badge from "$lib/components/ui/badge/badge.svelte";

	let { data } = $props();

	let statusMessage = $state("");
	let errorMessage = $state("");
	let isResending = $state<string | null>(null);

	async function resendEmail(id: string) {
		isResending = id;
		statusMessage = "";
		errorMessage = "";

		try {
			await apiClient.post(`/admin/submissions/${id}/resend-email`, {});
			statusMessage = `Resent approval email for ${id}.`;
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to resend email.";
		} finally {
			isResending = null;
		}
	}

	async function refresh() {
		statusMessage = "";
		errorMessage = "";
		await invalidateAll();
	}
</script>

<svelte:head>
	<title>Admin submission issues · Euro Haus</title>
</svelte:head>

<header
	class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
>
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Submission issues</h1>
		<p class="mt-3 max-w-3xl text-sm leading-7">
			Review problematic submissions and retry the approval-email flow.
			Manual payment-link recreation can be added in a later pass.
		</p>
	</div>
</header>

<section class="space-y-6">
	<div class="flex justify-end">
		<button
			class="rounded-full border border-white/10 px-4 py-2 text-sm"
			onclick={() => void refresh()}
		>
			Refresh
		</button>
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

	{#if data.submissions.length === 0}
		<div
			class="rounded-3xl border border-dashed border-white/10 bg-white/5 p-8 text-sm"
		>
			No submission issues detected.
		</div>
	{:else}
		<Item.Group class="flex flex-col gap-4">
			{#each data.submissions as submission, index (submission.id)}
				<Item.Root variant="outline" class="bg-white/5">
					<Item.Content class="gap-3">
						<Item.Header>
							<Item.Title>
								{submission.vehicle_year}
								{submission.vehicle_make}
								{submission.vehicle_model}
							</Item.Title>
							<Item.Description>
								{submission.participant_name} · {submission.participant_email}
							</Item.Description>
							<Item.Description>
								{submission.event_slug} · {formatDate(
									submission.submitted_at,
								)}
							</Item.Description>
						</Item.Header>

						{#if submission.issues?.length}
							<div class="flex flex-wrap gap-2">
								{#each submission.issues as issue (issue)}
									<Badge
										class="text-xs border-accent/30 text-accent hover:border-accent/50 hover:text-accent"
										variant="circle"
									>
										{issue}
									</Badge>
								{/each}
							</div>
						{/if}
					</Item.Content>

					<Item.Actions class="flex-wrap">
						<button
							class="rounded-full border border-white/10 px-4 py-2 text-sm"
							disabled={isResending === submission.id}
							onclick={() => void resendEmail(submission.id)}
						>
							{isResending === submission.id
								? "Resending…"
								: "Resend email"}
						</button>

						<a
							href={`/checkout/recover?submission=${submission.id}`}
							class="rounded-full border border-white/10 px-4 py-2 text-sm"
						>
							Open recovery route
						</a>
					</Item.Actions>
				</Item.Root>

				{#if index !== data.submissions.length - 1}
					<Item.Separator />
				{/if}
			{/each}
		</Item.Group>
	{/if}
</section>
