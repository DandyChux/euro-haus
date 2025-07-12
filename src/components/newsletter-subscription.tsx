import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";

import { Button } from "./ui/button";
import { Input } from "./ui/input";
import {
	Form,
	FormControl,
	FormField,
	FormItem,
	FormMessage,
} from "./ui/form";
import { apiClient } from "~/lib/api";

// Define schema for form validation
const formSchema = z.object({
	email: z
		.string()
		.min(1, { message: "Email is required" })
		.email({ message: "Invalid email address" }),
	firstName: z.string().optional(),
	lastName: z.string().optional(),
});

type FormValues = z.infer<typeof formSchema>;

export function NewsletterSubscription() {

	// Initialize form with validation
	const form = useForm<FormValues>({
		resolver: zodResolver(formSchema),
		defaultValues: {
			email: "",
			firstName: "",
			lastName: "",
		},
	});

	// Handle form submission
	const onSubmit = async (data: FormValues) => {

		try {
			await apiClient.post("/newsletter/subscribe", data);

			form.reset();

			toast('Successfully Subscribed', {
				description: "Thank you for joining our mailing list!",
			});
		} catch (error) {
			console.error("Newsletter subscription error:", error);
			toast('Subscription Failed', {
				description: error instanceof Error
					? error.message
					: "There was an error subscribing to the newsletter.",
			})
		}
	};

	return (
		<section className="py-12 md:py-16 lg:py-20" id="mailing-list">
			<div className="px-4 md:px-6">
				<div className="flex flex-col items-center justify-center space-y-4 text-center">
					<div className="space-y-2">
						<h2 className="text-3xl font-bold tracking-tighter sm:text-4xl md:text-5xl">
							Join Our Mailing List
						</h2>
						<p className="max-w-[700px] text-muted-foreground md:text-xl/relaxed lg:text-base/relaxed xl:text-xl/relaxed">
							Stay updated with the latest events, products, and community news.
						</p>
					</div>

					<div className="mx-auto w-full max-w-md space-y-4">
						<Form {...form}>
							<form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
								{/* <div className="grid grid-cols-2 gap-3">
									<FormField
										control={form.control}
										name="firstName"
										render={({ field }) => (
											<FormItem>
												<FormControl>
													<Input
														placeholder="First Name"
														disabled={form.formState.isSubmitting}
														{...field}
													/>
												</FormControl>
												<FormMessage />
											</FormItem>
										)}
									/>

									<FormField
										control={form.control}
										name="lastName"
										render={({ field }) => (
											<FormItem>
												<FormControl>
													<Input
														placeholder="Last Name"
														disabled={form.formState.isSubmitting}
														{...field}
													/>
												</FormControl>
												<FormMessage />
											</FormItem>
										)}
									/>
								</div> */}

								{/* Email field and submit button */}
								<div className="flex space-x-2">
									<FormField
										control={form.control}
										name="email"
										render={({ field }) => (
											<FormItem className="flex-1">
												<FormControl>
													<Input
														placeholder="Enter your email"
														type="email"
														disabled={form.formState.isSubmitting}
														{...field}
													/>
												</FormControl>
												<FormMessage />
											</FormItem>
										)}
									/>

									<Button
										type="submit"
										disabled={form.formState.isSubmitting}
									>
										{form.formState.isSubmitting ? "Subscribing..." : form.formState.isSubmitSuccessful ? "Subscribed!" : "Subscribe"}
									</Button>
								</div>
							</form>
						</Form>

						<p className="text-xs text-muted-foreground">
							By subscribing, you agree to our terms and privacy policy.
						</p>
					</div>
				</div>
			</div>
		</section>
	);
}
