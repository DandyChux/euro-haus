<script lang="ts">
	import { onMount, untrack } from "svelte";
	import { superForm, type SuperValidated } from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";

	import { apiClient } from "$lib/api";
	import type {
		EventAttendee,
		EventCheckIn,
		TicketInfo,
	} from "$lib/schemas/event";

	import { eventCheckInSchema } from "$lib/schemas/event";
	import { Html5Qrcode } from "html5-qrcode";

	type ScanMode = "camera" | "manual";

	interface Props {
		eventId: string;
		eventName: string;
		data: SuperValidated<EventCheckIn>;
	}

	let { eventId, eventName, data }: Props = $props();

	const form = superForm(
		untrack(() => data),
		{
			SPA: true,
			validators: zod4Client(eventCheckInSchema),

			async onUpdate({ form }) {
				if (!form.valid) return;

				await checkInTicket(form.data.code);
				$formData.code = "";
			},
		},
	);

	const { form: formData, enhance, submitting } = form;

	let scanMode = $state<ScanMode>("camera");
	let attendees = $state<EventAttendee[]>([]);
	let lastCheckedIn = $state<TicketInfo | null>(null);

	let isLoading = $state(true);
	let isChecking = $state(false);

	let scanner: Html5Qrcode | null = null;
	let isScanning = $state(false);
	let cameraError = $state("");

	let checkedInCount = $derived(
		attendees.filter((attendee) => attendee.checked_in).length,
	);

	let remainingCount = $derived(attendees.length - checkedInCount);

	async function loadAttendees() {
		isLoading = true;

		try {
			const response = await apiClient.get<{
				attendees?: EventAttendee[];
				stats: { total: number; checked_in: number };
			}>(`/admin/events/${eventId}/attendees`);

			attendees = response.attendees ?? [];
		} catch (error) {
			console.error("Loading attendees failed:", error);
			toast.error("Unable to load event attendees.");
		} finally {
			isLoading = false;
		}
	}

	async function checkInTicket(code: string) {
		if (isChecking) return;

		let token = code.trim();

		// If a full check-in URL was scanned (from camera/email link), extract the token parameter
		if (token.includes("ticket=")) {
			try {
				const url = new URL(token);
				token = url.searchParams.get("ticket") || token;
			} catch {
				const match = token.match(/ticket=([A-Za-z0-9_-]+)/);
				if (match) token = match[1];
			}
		}

		if (!token) return;

		isChecking = true;

		try {
			const ticket = await apiClient.post<TicketInfo>(
				"/events/ticket/validate",
				{ token },
			);

			if (!ticket.valid) {
				toast.error("Invalid ticket code.");
				return;
			}

			if (ticket.event_id !== eventId) {
				toast.error("This ticket is for a different event.");
				return;
			}

			if (ticket.checked_in) {
				lastCheckedIn = ticket;
				toast.warning("This ticket has already been checked in.");
				return;
			}

			const checkedIn = await apiClient.post<TicketInfo>(
				"/admin/events/ticket/check-in",
				{ token },
			);

			lastCheckedIn = checkedIn;
			toast.success(`Checked in: ${checkedIn.customer_name}`);
			await loadAttendees();
		} catch (error) {
			console.error("Ticket check-in failed:", error);
			toast.error("Unable to check in ticket.");
		} finally {
			isChecking = false;
		}
	}

	async function startCamera() {
		cameraError = "";

		try {
			if (!scanner) {
				scanner = new Html5Qrcode("qr-reader");
			}

			isScanning = true;

			await scanner.start(
				{ facingMode: "environment" },
				{
					fps: 10,
					qrbox: { width: 250, height: 250 },
					aspectRatio: 1.0,
				},
				async (decodedText) => {
					if (isChecking) return;
					await checkInTicket(decodedText);
				},
				() => {
					// Frame parse misses are normal during active search
				},
			);
		} catch (error) {
			console.error("Starting QR scanner failed:", error);
			cameraError =
				"Unable to access camera. Please check camera permissions in your browser.";
			isScanning = false;
		}
	}

	async function stopCamera() {
		if (scanner && isScanning) {
			try {
				await scanner.stop();
				scanner.clear();
			} catch (err) {
				console.error("Error stopping scanner:", err);
			}
			isScanning = false;
			scanner = null;
		}
	}

	function setMode(mode: ScanMode) {
		if (mode !== "camera") {
			void stopCamera();
		}

		scanMode = mode;

		if (mode === "camera") {
			void startCamera();
		}
	}

	onMount(() => {
		void loadAttendees();
		void startCamera();

		return () => {
			void stopCamera();
		};
	});
</script>

<svelte:head>
	<title>{eventName} check-in · Admin</title>
</svelte:head>

<section class="space-y-6">
	<div class="grid gap-4 md:grid-cols-3">
		<Card class="p-4">
			<p class="text-sm text-muted">Total tickets</p>
			<p class="mt-2 text-3xl font-semibold">
				{isLoading ? "—" : attendees.length}
			</p>
		</Card>

		<Card class="p-4">
			<p class="text-sm text-muted">Checked in</p>
			<p class="mt-2 text-3xl font-semibold text-emerald-600">
				{isLoading ? "—" : checkedInCount}
			</p>
		</Card>

		<Card class="p-4">
			<p class="text-sm text-muted">Remaining</p>
			<p class="mt-2 text-3xl font-semibold">
				{isLoading ? "—" : remainingCount}
			</p>
		</Card>
	</div>

	<div class="flex gap-2">
		<Button
			type="button"
			variant={scanMode === "camera" ? "default" : "outline"}
			onclick={() => setMode("camera")}
		>
			Scan QR code
		</Button>

		<Button
			type="button"
			variant={scanMode === "manual" ? "default" : "outline"}
			onclick={() => setMode("manual")}
		>
			Enter code
		</Button>
	</div>

	<Card class="p-6">
		{#if scanMode === "camera"}
			<div class="space-y-4">
				{#if cameraError}
					<p role="alert" class="text-sm text-destructive">
						{cameraError}
					</p>
				{/if}

				<div
					id="qr-reader"
					class="mx-auto w-full max-w-md overflow-hidden rounded-2xl bg-black"
				></div>

				<p class="text-center text-sm text-muted">
					Point the camera at the ticket QR code.
				</p>
			</div>
		{:else}
			<form
				method="POST"
				use:enhance
				class="flex flex-col gap-3 sm:flex-row sm:items-end"
			>
				<Form.Field {form} name="code" class="flex-1">
					<Form.Control>
						{#snippet children({ props })}
							<Form.Label>Ticket code</Form.Label>
							<Input
								{...props}
								bind:value={$formData.code}
								autocomplete="off"
								autofocus
							/>
						{/snippet}
					</Form.Control>
					<Form.FieldErrors />
				</Form.Field>

				<Button type="submit" disabled={$submitting || isChecking}>
					Check in
				</Button>
			</form>
		{/if}
	</Card>

	{#if lastCheckedIn}
		<Card class="border-emerald-500/40 p-4">
			<p class="font-semibold">
				{lastCheckedIn.customer_name}
			</p>

			<p class="text-sm text-muted">
				{lastCheckedIn.ticket_type}
				·
				{lastCheckedIn.ticket_code}
			</p>

			{#if lastCheckedIn.checked_in_at}
				<p class="mt-1 text-sm">
					Checked in at
					{new Date(lastCheckedIn.checked_in_at).toLocaleString()}
				</p>
			{/if}
		</Card>
	{/if}
</section>
