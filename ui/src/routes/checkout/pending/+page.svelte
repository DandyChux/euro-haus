<script lang="ts">
	import { formatDate } from "$lib/utils";
	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Submission pending · Euro Haus</title>
</svelte:head>

<section class="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
	<div class="rounded-3xl border border-white/10 bg-white/5 p-8">
		<h1 class="text-3xl font-semibold text-white">Submission received</h1>
		<p class="mt-4 text-base leading-7">
			Your vehicle submission is in review. You’ll get an email once the
			team has approved or denied it.
		</p>

		{#if data.submission}
			<div
				class="mt-8 space-y-5 rounded-2xl border border-white/10 bg-black/20 p-5 text-sm"
			>
				<div>
					<p class="">Vehicle</p>
					<p class="mt-1 text-white">
						{data.submission.vehicle_year}
						{data.submission.vehicle_make}
						{data.submission.vehicle_model}
					</p>
				</div>

				<div>
					<p class="">Participant</p>
					<p class="mt-1 text-white">
						{data.submission.participant_name}
					</p>
					<p>{data.submission.participant_email}</p>
				</div>

				<div>
					<p class="">Reference</p>
					<p class="mt-1 font-mono text-white">
						{data.submission.id}
					</p>
				</div>

				<div>
					<p class="">Submitted</p>
					<p class="mt-1 text-white">
						{formatDate(data.submission.submitted_at)}
					</p>
				</div>
			</div>
		{/if}

		<div class="mt-8 flex flex-wrap gap-3">
			<a
				href="/events"
				class="rounded-full bg-white px-5 py-3 text-sm font-medium"
				>View events</a
			>
			<a
				href="/"
				class="rounded-full border border-white/10 px-5 py-3 text-sm font-medium text-white"
				>Return home</a
			>
		</div>
	</div>
</section>
