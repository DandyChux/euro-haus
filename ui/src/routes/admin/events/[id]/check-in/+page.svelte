<script lang="ts">
	import { goto } from "$app/navigation";

	import EventCheckIn from "$lib/components/admin/event-checkin.svelte";

	let { data } = $props();
</script>

<svelte:head>
	<title>Check in · Euro Haus</title>
</svelte:head>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4">
		<div>
			<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
			<h1 class="mt-2 text-3xl font-semibold">Check in for event</h1>
		</div>

		<a href="/admin/events" class="rounded-full border px-4 py-2 text-sm">
			Back to events
		</a>
	</header>

	<EventCheckIn
		eventId={data.event.id}
		eventName={data.event.name}
		data={data.form}
	/>
</div>
