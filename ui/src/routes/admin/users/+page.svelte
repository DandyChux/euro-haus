<script lang="ts">
	import { superForm } from "sveltekit-superforms";
	import { zod4Client } from "sveltekit-superforms/adapters";
	import { toast } from "svelte-sonner";

	import * as Card from "$lib/components/ui/card";
	import * as Form from "$lib/components/ui/form";
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { apiClient } from "$lib/api";
	import { createUserSchema } from "$lib/schemas/user";

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

<section class="space-y-6">
	<div>
		<p class="text-sm uppercase tracking-[0.3em]">Administration</p>
		<h1 class="mt-2 text-3xl font-semibold">Create administrator</h1>
		<p class="mt-2 text-sm text-muted-foreground">
			Add another administrator account.
		</p>
	</div>

	<Card.Root class="max-w-2xl rounded-3xl border p-6">
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
</section>
