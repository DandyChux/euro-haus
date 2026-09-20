<script lang="ts">
	import { superForm } from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Card from "$lib/components/ui/card";
	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { apiClient } from "$lib/api";
	import { createUserSchema, updateProfileSchema } from "$lib/schemas/user";
	import { adminAuth } from "$lib/stores/auth.svelte";

	let profile = $state({ name: "", email: "" });
	let profileLoading = $state(true);

	const profileForm = superForm(
		{ name: "", email: "", password: "" },
		{
			SPA: true,
			validators: zod4Client(updateProfileSchema),
			async onUpdate({ form }) {
				if (!form.valid) return;
				try {
					const response = await apiClient.put<{
						name: string;
						email: string;
					}>("/admin/profile", form.data);
					profile = { name: response.name, email: response.email };
					$formProfileData.password = "";
					toast.success("Profile updated.");
				} catch (error) {
					toast.error(
						error instanceof Error
							? error.message
							: "Unable to update profile",
					);
				}
			},
		},
	);
	const {
		form: formProfileData,
		enhance: enhanceProfile,
		submitting: profileSubmitting,
	} = profileForm;

	async function loadProfile() {
		try {
			const response = await apiClient.get<{
				name: string;
				email: string;
			}>("/admin/profile");
			profile = { name: response.name, email: response.email };
			$formProfileData.name = response.name;
			$formProfileData.email = response.email;
		} catch {
			// The profile form remains usable even if loading fails.
		} finally {
			profileLoading = false;
		}
	}

	$effect(() => {
		if (adminAuth.isAuthenticated) void loadProfile();
	});

	const form = superForm(
		{
			name: "",
			email: "",
			password: "",
		},
		{
			SPA: true,
			validators: zod4Client(createUserSchema),

			async onUpdate({ form }) {
				if (!form.valid) return;

				try {
					const response = await apiClient.post<{ email: string }>(
						"/admin/users",
						form.data,
					);

					toast.success(`Created administrator ${response.email}`);
					$formData.name = "";
					$formData.email = "";
					$formData.password = "";
				} catch (error) {
					toast.error(
						error instanceof Error
							? error.message
							: "Unable to create administrator",
					);
				}
			},
		},
	);

	const { form: formData, enhance, submitting } = form;
</script>

<svelte:head>
	<title>Admin users · Euro Haus</title>
</svelte:head>

<section class="space-y-10">
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Administration</p>
		<h1 class="mt-2 text-3xl font-semibold">Users & profile</h1>
		<p class="mt-2 text-sm text-muted-foreground">
			Manage administrator access and your account details.
		</p>
	</div>

	<div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
		<Card.Root class="rounded-3xl border p-6">
			<div class="mb-6">
				<p class="text-sm uppercase tracking-[0.2em] text-muted-foreground">Your account</p>
				<h2 class="mt-2 text-xl font-semibold">Update your details</h2>
				<p class="mt-1 text-sm text-muted-foreground">Leave the password blank to keep it unchanged.</p>
			</div>
			<form method="POST" use:enhanceProfile class="space-y-6">
				<Form.Field form={profileForm} name="name"><Form.Control>{#snippet children({ props })}<Form.Label>Name</Form.Label><Input {...props} bind:value={$formProfileData.name} autocomplete="name" />{/snippet}</Form.Control><Form.FieldErrors /></Form.Field>
				<Form.Field form={profileForm} name="email"><Form.Control>{#snippet children({ props })}<Form.Label>Email</Form.Label><Input {...props} type="email" bind:value={$formProfileData.email} autocomplete="email" />{/snippet}</Form.Control><Form.FieldErrors /></Form.Field>
				<Form.Field form={profileForm} name="password"><Form.Control>{#snippet children({ props })}<Form.Label>New password</Form.Label><Input {...props} type="password" bind:value={$formProfileData.password} autocomplete="new-password" placeholder="Leave blank to keep current password" />{/snippet}</Form.Control><Form.FieldErrors /></Form.Field>
				<Button type="submit" disabled={$profileSubmitting || profileLoading}>{$profileSubmitting ? "Saving…" : "Save profile"}</Button>
			</form>
		</Card.Root>

		<Card.Root class="rounded-3xl border p-6">

		<Card.Root class="rounded-3xl border p-6">
		<form method="POST" use:enhance class="space-y-6">
			<Form.Field {form} name="name">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Name</Form.Label>
						<Input
							{...props}
							bind:value={$formData.name}
							autocomplete="name"
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="email">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Email</Form.Label>
						<Input
							{...props}
							type="email"
							bind:value={$formData.email}
							autocomplete="email"
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Form.Field {form} name="password">
				<Form.Control>
					{#snippet children({ props })}
						<Form.Label>Temporary password</Form.Label>
						<Input
							{...props}
							type="password"
							bind:value={$formData.password}
							autocomplete="new-password"
						/>
					{/snippet}
				</Form.Control>

				<Form.FieldErrors />
			</Form.Field>

			<Button type="submit" disabled={$submitting}>
				{$submitting ? "Creating…" : "Create administrator"}
			</Button>
		</form>
		</Card.Root>
	</div>
</section>
