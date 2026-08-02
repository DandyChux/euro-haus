<script lang="ts">
	import { goto } from "$app/navigation";
	import { toast } from "svelte-sonner";

	import apiClient from "$lib/api";
	import EventForm from "$lib/components/admin/event-form.svelte";
	import type { Event } from "$lib/schemas/event";

	import type { PageProps } from "./$types";

	let { data }: PageProps = $props();

	async function saveEvent(event: Event) {
		const response = await apiClient.post<{
			id: string;
		}>("/admin/events", event);

		toast.success("Event created.");

		await goto(`/admin/events/${response.id}`);
	}
</script>

<svelte:head>
	<title>New event · Euro Haus</title>
</svelte:head>

<div class="space-y-6">
	<header>
		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-2 text-3xl font-semibold">Create event</h1>
	</header>

	<EventForm data={{ form: data.form }} onsaved={saveEvent} />
</div>
