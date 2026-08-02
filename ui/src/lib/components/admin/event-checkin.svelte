<script lang="ts">
	import { onMount, untrack } from "svelte";
	import {
		superForm,
		type Infer,
		type SuperValidated,
	} from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";

	import { apiClient } from "$lib/api";
	import type {
		EventAttendee,
		Ticket,
		EventCheckIn,
		TicketInfo,
	} from "$lib/schemas/event";

	import { eventCheckInSchema } from "$lib/schemas/event";

	type ScanMode = "camera" | "manual";

	interface Props {
		eventId: string;
		eventName: string;
		data: {
			form: SuperValidated<EventCheckIn>;
		};
	}

	let { eventId, eventName, data }: Props = $props();

	const form = superForm(
		untrack(() => data.form),
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

	let videoElement = $state<HTMLVideoElement | undefined>();

	let stream = $state<MediaStream | null>(null);
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

		const token = code.trim();

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

		if (!videoElement) return;

		const BarcodeDetectorConstructor = (
			window as typeof window & {
				BarcodeDetector?: new () => {
					detect(
						source: HTMLVideoElement,
					): Promise<Array<{ rawValue: string }>>;
				};
			}
		).BarcodeDetector;

		if (!BarcodeDetectorConstructor) {
			cameraError = "QR scanning is not supported in this browser.";
			return;
		}

		try {
			stream = await navigator.mediaDevices.getUserMedia({
				video: {
					facingMode: "environment",
				},
				audio: false,
			});

			videoElement.srcObject = stream;
			await videoElement.play();

			const detector = new BarcodeDetectorConstructor();

			while (scanMode === "camera" && stream) {
				const codes = await detector.detect(videoElement);

				const code = codes[0]?.rawValue;

				if (code) {
					await checkInTicket(code);
				}

				await new Promise((resolve) => setTimeout(resolve, 500));
			}
		} catch (error) {
			console.error("Starting QR scanner failed:", error);

			cameraError = "Unable to access the camera.";
		}
	}

	function stopCamera() {
		stream?.getTracks().forEach((track) => track.stop());

		stream = null;

		if (videoElement) {
			videoElement.srcObject = null;
		}
	}

	function setMode(mode: ScanMode) {
		if (mode !== "camera") {
			stopCamera();
		}

		scanMode = mode;

		if (mode === "camera") {
			void startCamera();
		}
	}

	onMount(() => {
		void loadAttendees();
		void startCamera();

		return stopCamera;
	});
</script>

<svelte:head>
	<title>{eventName} check-in · Admin</title>
</svelte:head>

<section class="space-y-6">
	<div class="grid gap-4 md:grid-cols-3">
		<Card class="p-4">
			<p class="text-sm text-muted-foreground">Total tickets</p>

			<p class="mt-2 text-3xl font-semibold">
				{isLoading ? "—" : attendees.length}
			</p>
		</Card>

		<Card class="p-4">
			<p class="text-sm text-muted-foreground">Checked in</p>

			<p class="mt-2 text-3xl font-semibold text-emerald-600">
				{isLoading ? "—" : checkedInCount}
			</p>
		</Card>

		<Card class="p-4">
			<p class="text-sm text-muted-foreground">Remaining</p>

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

				<video
					bind:this={videoElement}
					muted
					playsinline
					class="mx-auto aspect-video w-full max-w-md rounded-2xl bg-black object-cover"
				></video>

				<p class="text-center text-sm text-muted-foreground">
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

			<p class="text-sm text-muted-foreground">
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
