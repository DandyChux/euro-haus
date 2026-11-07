<script lang="ts">
	import { onMount, untrack } from "svelte";
	import { superForm, type SuperValidated } from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { Card } from "$lib/components/ui/card";
	import { Badge } from "$lib/components/ui/badge";

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

	let scanner = $state<Html5Qrcode | null>(null);
	let isScanning = $state(false);
	let cameraError = $state("");
	let lastScannedCode = $state("");
	let fileInputRef = $state<HTMLInputElement | null>(null);

	// Attendee search and filter state
	let attendeeSearch = $state("");
	let attendeeFilter = $state<"all" | "checked_in" | "pending">("all");

	let checkedInCount = $derived(
		attendees.filter((attendee) => attendee.checked_in).length,
	);

	let remainingCount = $derived(attendees.length - checkedInCount);

	let filteredAttendees = $derived.by(() => {
		const q = attendeeSearch.trim().toLowerCase();
		return attendees
			.filter((a) => {
				if (attendeeFilter === "checked_in" && !a.checked_in)
					return false;
				if (attendeeFilter === "pending" && a.checked_in) return false;

				if (!q) return true;

				const name = (
					a.customer_name ||
					((a as Record<string, unknown>).name as string) ||
					""
				).toLowerCase();
				const email = (
					a.customer_email ||
					((a as Record<string, unknown>).email as string) ||
					""
				).toLowerCase();
				const token = (a.token || "").toLowerCase();
				const tier = (a.ticket_type || "").toLowerCase();

				return (
					name.includes(q) ||
					email.includes(q) ||
					token.includes(q) ||
					tier.includes(q)
				);
			})
			.sort((a, b) => {
				// Show checked in attendees sorted by checked_in_at desc (most recent first)
				if (a.checked_in && b.checked_in) {
					const timeA = a.checked_in_at
						? new Date(a.checked_in_at).getTime()
						: 0;
					const timeB = b.checked_in_at
						? new Date(b.checked_in_at).getTime()
						: 0;
					return timeB - timeA;
				}
				// Checked-in first if looking at all, or un-checked
				if (a.checked_in !== b.checked_in) {
					return a.checked_in ? -1 : 1;
				}
				// Both pending: sort alphabetically by name
				const nameA = (
					a.customer_name ||
					((a as Record<string, unknown>).name as string) ||
					""
				).toLowerCase();
				const nameB = (
					b.customer_name ||
					((b as Record<string, unknown>).name as string) ||
					""
				).toLowerCase();
				return nameA.localeCompare(nameB);
			});
	});

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

	function cleanToken(code: string): string {
		let token = code.trim();
		if (token.includes("ticket=")) {
			try {
				const url = new URL(token);
				token = url.searchParams.get("ticket") || token;
			} catch {
				const match = token.match(/ticket=([A-Za-z0-9_-]+)/);
				if (match) token = match[1];
			}
		}
		return token;
	}

	async function checkInTicket(code: string) {
		if (isChecking) return;

		const token = cleanToken(code);
		if (!token) return;

		if (token === lastScannedCode && lastCheckedIn?.ticket_code === token) {
			return;
		}

		isChecking = true;
		lastScannedCode = token;

		if (typeof navigator !== "undefined" && navigator.vibrate) {
			navigator.vibrate([100, 50, 100]);
		}

		try {
			if (scanner && isScanning) {
				try {
					scanner.pause(true);
				} catch {}
			}

			const ticket = await apiClient.post<TicketInfo>(
				"/events/ticket/validate",
				{ token },
			);

			if (!ticket.valid) {
				toast.error("Invalid ticket code: " + token);
				return;
			}

			if (ticket.event_id !== eventId) {
				toast.error("Ticket is for a different event.");
				return;
			}

			if (ticket.checked_in) {
				lastCheckedIn = ticket;
				toast.warning(`Already checked in: ${ticket.customer_name}`);
				return;
			}

			const checkedIn = await apiClient.post<TicketInfo>(
				"/admin/events/ticket/check-in",
				{ token },
			);

			lastCheckedIn = checkedIn;
			toast.success(
				`Checked in: ${checkedIn.customer_name} (${checkedIn.ticket_type})`,
			);
			await loadAttendees();
		} catch (error) {
			console.error("Ticket check-in failed:", error);
			toast.error("Unable to check in ticket.");
		} finally {
			isChecking = false;
			setTimeout(() => {
				if (scanner && isScanning) {
					try {
						scanner.resume();
					} catch {}
				}
			}, 1500);
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
					fps: 15,
					qrbox: (
						viewfinderWidth: number,
						viewfinderHeight: number,
					) => {
						const minEdge = Math.min(
							viewfinderWidth,
							viewfinderHeight,
						);
						const size = Math.floor(minEdge * 0.8);
						return { width: size, height: size };
					},
					aspectRatio: 1.0,
					experimentalFeatures: {
						useBarCodeDetectorIfSupported: true,
					},
				},
				async (decodedText) => {
					await checkInTicket(decodedText);
				},
				() => {},
			);
		} catch (error) {
			console.error("Starting QR scanner failed:", error);
			cameraError =
				"Unable to access camera. Please allow camera permissions in your browser.";
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

	async function handleImageFile(event: Event) {
		const target = event.target as HTMLInputElement;
		const file = target.files?.[0];
		if (!file) return;

		try {
			toast.info("Scanning image…");
			const tempScanner = new Html5Qrcode("qr-file-processor");
			const result = await tempScanner.scanFile(file, true);
			await checkInTicket(result);
		} catch (err) {
			console.error("File scan failed:", err);
			toast.error(
				"No QR code detected in this photo. Try again or enter code manually.",
			);
		} finally {
			if (target) target.value = "";
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
	<!-- Top Statistics -->
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

	<!-- Mode Switcher -->
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

	<!-- Check-In Scanner / Input Card -->
	<Card class="p-6">
		{#if scanMode === "camera"}
			<div class="space-y-4">
				{#if cameraError}
					<p role="alert" class="text-sm text-destructive">
						{cameraError}
					</p>
				{/if}

				<!-- Video Viewfinder Container -->
				<div
					class="relative mx-auto w-full max-w-md overflow-hidden rounded-2xl bg-black"
				>
					<div id="qr-reader" class="w-full"></div>

					{#if isChecking}
						<div
							class="absolute inset-0 flex items-center justify-center bg-black/60 text-white backdrop-blur-xs"
						>
							<Badge class="p-3 text-sm font-semibold"
								>Checking in ticket…</Badge
							>
						</div>
					{/if}
				</div>

				<p class="text-center text-sm text-muted">
					Point camera at the QR code. It will auto-detect and check
					in immediately.
				</p>

				<div class="flex justify-center pt-2">
					<input
						type="file"
						accept="image/*"
						capture="environment"
						bind:this={fileInputRef}
						class="hidden"
						onchange={handleImageFile}
					/>
					<Button
						type="button"
						variant="outline"
						size="sm"
						onclick={() => fileInputRef?.click()}
					>
						📷 Take Photo / Upload QR
					</Button>
				</div>

				<div id="qr-file-processor" class="hidden"></div>
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

	<!-- Last Checked In Card -->
	{#if lastCheckedIn}
		<Card class="border-emerald-500/40 bg-emerald-500/5 p-4">
			<div class="flex items-center justify-between">
				<div>
					<p class="font-semibold text-lg text-emerald-400">
						✓ Checked In: {lastCheckedIn.customer_name}
					</p>
					<p class="text-sm text-muted">
						{lastCheckedIn.ticket_type} ·
						<span class="font-mono"
							>{lastCheckedIn.ticket_code}</span
						>
					</p>
				</div>
				{#if lastCheckedIn.checked_in_at}
					<span class="text-xs text-muted">
						{new Date(
							lastCheckedIn.checked_in_at,
						).toLocaleTimeString([], {
							hour: "2-digit",
							minute: "2-digit",
						})}
					</span>
				{/if}
			</div>
		</Card>
	{/if}

	<!-- Attendees Ticket List with Search & Status Filter -->
	<div class="space-y-4 pt-4">
		<div
			class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center"
		>
			<div>
				<h2 class="text-xl font-bold">Attendees & Tickets</h2>
				<p class="text-xs text-muted">
					Real-time attendee check-in roster
				</p>
			</div>

			<div class="flex gap-1 rounded-xl border bg-muted/20 p-1 text-xs">
				<button
					type="button"
					class="rounded-lg px-3 py-1 font-medium transition-colors {attendeeFilter ===
					'all'
						? 'bg-primary text-primary-foreground'
						: 'text-muted-foreground'}"
					onclick={() => (attendeeFilter = "all")}
				>
					All ({attendees.length})
				</button>
				<button
					type="button"
					class="rounded-lg px-3 py-1 font-medium transition-colors {attendeeFilter ===
					'checked_in'
						? 'bg-primary text-primary-foreground'
						: 'text-muted-foreground'}"
					onclick={() => (attendeeFilter = "checked_in")}
				>
					Checked In ({checkedInCount})
				</button>
				<button
					type="button"
					class="rounded-lg px-3 py-1 font-medium transition-colors {attendeeFilter ===
					'pending'
						? 'bg-primary text-primary-foreground'
						: 'text-muted-foreground'}"
					onclick={() => (attendeeFilter = "pending")}
				>
					Pending ({remainingCount})
				</button>
			</div>
		</div>

		<!-- Search Input -->
		<Input
			placeholder="Search by customer name, email, ticket code, or tier…"
			bind:value={attendeeSearch}
			class="w-full"
		/>

		<!-- Attendee Cards Grid -->
		{#if isLoading}
			<Card class="p-8 text-center text-muted">
				Loading attendee records…
			</Card>
		{:else if filteredAttendees.length === 0}
			<Card class="p-8 text-center text-muted">
				{attendeeSearch
					? "No attendees matching your search."
					: "No attendees registered for this event."}
			</Card>
		{:else}
			<div class="grid gap-2">
				{#each filteredAttendees as attendee (attendee.token)}
					<Card
						class="flex items-center justify-between p-4 transition-all {attendee.checked_in
							? 'border-white/5 bg-white/2 opacity-60 dark:bg-black/30'
							: 'border-white/10 bg-white/5'}"
					>
						<div class="flex items-center gap-3">
							<!-- Status Icon / Question Mark or Checkmark -->
							{#if attendee.checked_in}
								<div
									class="flex h-8 w-8 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400 font-bold"
								>
									✓
								</div>
							{:else}
								<div
									class="flex h-8 w-8 items-center justify-center rounded-full bg-amber-500/20 text-amber-400 font-bold"
								>
									?
								</div>
							{/if}

							<div>
								<div class="flex items-center gap-2">
									<p
										class="font-medium {attendee.checked_in
											? 'text-muted-foreground line-through'
											: 'text-foreground'}"
									>
										{attendee.customer_name ||
											(
												attendee as Record<
													string,
													unknown
												>
											).name ||
											"Unnamed Attendee"}
									</p>
									<Badge
										variant={attendee.checked_in
											? "outline"
											: "secondary"}
										class="text-xs"
									>
										{attendee.ticket_type || "General"}
									</Badge>
								</div>

								<p class="text-xs text-muted">
									<span class="font-mono"
										>{attendee.token}</span
									>
									{#if attendee.customer_email || (attendee as Record<string, unknown>).email}
										· {attendee.customer_email ||
											(
												attendee as Record<
													string,
													unknown
												>
											).email}
									{/if}
								</p>
							</div>
						</div>

						<!-- Check In Status or Action Button -->
						<div class="flex items-center gap-3">
							{#if attendee.checked_in}
								<div class="text-right">
									<span
										class="text-xs font-semibold text-emerald-400"
										>Checked In</span
									>
									{#if attendee.checked_in_at}
										<p class="text-[11px] text-muted">
											{new Date(
												attendee.checked_in_at,
											).toLocaleTimeString([], {
												hour: "2-digit",
												minute: "2-digit",
											})}
										</p>
									{/if}
								</div>
							{:else}
								<Button
									size="sm"
									variant="outline"
									disabled={isChecking}
									onclick={() =>
										checkInTicket(attendee.token)}
								>
									Check In
								</Button>
							{/if}
						</div>
					</Card>
				{/each}
			</div>
		{/if}
	</div>
</section>
