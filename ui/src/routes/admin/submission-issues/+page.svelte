<script lang="ts">
	import { onMount } from "svelte";
	import { apiClient } from "$lib/api";
	import { formatDate } from "$lib/utils";

	interface IssueSubmission {
		id: string;
		eventId: string;
		eventSlug: string;
		participantName: string;
		participantEmail: string;
		vehicleYear: string;
		vehicleMake: string;
		vehicleModel: string;
		status: string;
		submittedAt: string;
		issues?: string[];
		emailSent?: boolean;
		checkoutSessionId?: string;
		paymentIntentId?: string;
		ticketId?: string;
	}

	let submissions = $state<IssueSubmission[]>([]);
	let isLoading = $state(true);
	let errorMessage = $state("");
	let statusMessage = $state("");

	async function loadIssues() {
		isLoading = true;
		errorMessage = "";

		try {
			const response = await apiClient.get<{
				submissions?: IssueSubmission[];
			}>("/admin/submissions/issues");
			submissions = response.submissions ?? [];
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to load submission issues.";
		} finally {
			isLoading = false;
		}
	}

	async function resendEmail(id: string) {
		try {
			await apiClient.post(`/admin/submissions/${id}/resend-email`, {});
			statusMessage = `Resent approval email for ${id}.`;
		} catch (error) {
			errorMessage =
				error instanceof Error
					? error.message
					: "Unable to resend email.";
		}
	}

	onMount(() => {
		void loadIssues();
	});
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
			onclick={() => void loadIssues()}
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

	{#if isLoading}
		<div class="rounded-3xl border border-white/10 bg-white/5 p-8 text-sm">
			Loading issue list…
		</div>
	{:else if submissions.length === 0}
		<div
			class="rounded-3xl border border-dashed border-white/10 bg-white/5 p-8 text-sm"
		>
			No submission issues detected.
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
								{submission.vehicleYear}
								{submission.vehicleMake}
								{submission.vehicleModel}
							</p>
							<p class="mt-1 text-sm">
								{submission.participantName} · {submission.participantEmail}
							</p>
							<p class="mt-2 text-sm">
								{submission.eventSlug} · {formatDate(
									submission.submittedAt,
								)}
							</p>

							<div class="mt-4 flex flex-wrap gap-2">
								{#each submission.issues ?? [] as issue (issue)}
									<span
										class="rounded-full border border-amber-500/30 px-3 py-1 text-xs text-amber-100"
									>
										{issue}
									</span>
								{/each}
							</div>
						</div>

						<div class="flex flex-wrap gap-3">
							<button
								class="rounded-full border border-white/10 px-4 py-2 text-sm"
								onclick={() => void resendEmail(submission.id)}
							>
								Resend email
							</button>
							<a
								href={`/checkout/recover?submission=${submission.id}`}
								class="rounded-full border border-white/10 px-4 py-2 text-sm"
							>
								Open recovery route
							</a>
						</div>
					</div>
				</article>
			{/each}
		</div>
	{/if}
</section>
