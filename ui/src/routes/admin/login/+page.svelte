<script lang="ts">
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { adminAuth, restoreAdminSession } from "$lib/stores/auth.svelte";
	import { loginAdmin, validateAdminSession } from "$lib/auth";
	import { Input } from "$lib/components/ui/input";
	import { resolve } from "$app/paths";

	let email = $state("");
	let password = $state("");
	let errorMessage = $state("");

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		errorMessage = "";

		const success = await loginAdmin(email, password);

		if (success) {
			await goto("/admin");
			return;
		}

		errorMessage = adminAuth.lastError || "Unable to sign in.";
	}

	onMount(() => {
		void (async () => {
			restoreAdminSession();
			const valid = await validateAdminSession();
			if (valid) {
				await goto("/admin");
			}
		})();
	});
</script>

<svelte:head>
	<title>Admin login · Euro Haus</title>
</svelte:head>

<section class="flex min-h-screen items-center justify-center px-4 py-12">
	<div
		class="w-full max-w-md rounded-3xl border border-white/10 bg-white/5 p-8"
	>
		<a
			href={resolve("/")}
			class="mb-10 inline-flex items-center gap-2 text-xs uppercase tracking-[0.2em] text-muted-foreground transition-colors hover:text-foreground"
		>
			<span aria-hidden="true">←</span>
			Back to Euro Haus
		</a>

		<p class="text-sm uppercase tracking-[0.3em]">Admin</p>
		<h1 class="mt-3 text-3xl font-semibold">Sign in</h1>
		<p class="mt-2 text-sm">Sign in with your administrator account.</p>

		<form class="mt-8 space-y-4" onsubmit={submit}>
			<label class="block text-sm">
				Email
				<Input
					bind:value={email}
					type="email"
					autocomplete="username"
					required
					class="mt-2"
					placeholder="admin@example.com"
				/>
			</label>

			<label class="block text-sm">
				Password
				<Input
					bind:value={password}
					type="password"
					autocomplete="current-password"
					required
					class="mt-2"
					placeholder="Enter your password"
				/>
			</label>

			{#if errorMessage}
				<p
					role="alert"
					class="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
				>
					{errorMessage}
				</p>
			{/if}

			<button
				type="submit"
				class="w-full rounded-full bg-white px-5 py-3 text-sm font-medium disabled:opacity-60"
				disabled={adminAuth.checking ||
					email.trim().length === 0 ||
					password.length === 0}
			>
				{adminAuth.checking ? "Signing in…" : "Sign in"}
			</button>
		</form>
	</div>
</section>
