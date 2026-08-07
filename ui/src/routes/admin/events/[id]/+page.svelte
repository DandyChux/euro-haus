<script lang="ts">
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import EventForm from "$lib/components/admin/event-form.svelte";
	import type { Event } from "$lib/schemas/event";

	let { data } = $props();

	async function saveEvent(event: Event) {
		await apiClient.put(`/admin/events/${data.event.id}`, event);

		toast.success("Event updated.");

		await goto("/admin/events");
	}
</script>

<svelte:head>
	<title>Edit event · Euro Haus</title>
</svelte:head>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4">
		<div>
			<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
			<h1 class="mt-2 text-3xl font-semibold">Edit event</h1>
		</div>

		<a href="/admin/events" class="rounded-full border px-4 py-2 text-sm">
			Back to events
		</a>
	</header>

	<EventForm data={{ form: data.form }} onsaved={saveEvent} />
</div>
