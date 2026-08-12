<script lang="ts">
	import { invalidateAll } from "$app/navigation";
	import SubmissionIssuesManager from "$lib/components/admin/submission-issues-manager.svelte";

	let { data } = $props();

	async function refresh() {
		await invalidateAll();
	}
</script>

<svelte:head>
	<title>Admin submission issues · Euro Haus</title>
</svelte:head>

<div class="space-y-6">
	<header
		class="flex flex-col gap-4 rounded-3xl border p-6 lg:flex-row lg:items-end lg:justify-between"
	>
		<div>
			<p class="text-sm uppercase tracking-[0.3em]">Admin</p>

			<h1 class="mt-2 text-3xl font-semibold">Submission issues</h1>

			<p class="mt-3 max-w-3xl text-sm leading-7">
				Inspect Stripe state, repair missing payment references, retry
				capture and ticket creation, recreate orphaned checkouts, and
				resend failed emails.
			</p>
		</div>
	</header>

	<SubmissionIssuesManager
		submissions={data.issues ?? []}
		onRefresh={refresh}
	/>
</div>
