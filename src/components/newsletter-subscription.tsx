import { Button } from "./ui/button";
import { Input } from "./ui/input";

export function NewsletterSubscription() {
	return (
		<section className="py-12 md:py-16 lg:py-20">
			<div className="px-4 md:px-6">
				<div className="flex flex-col items-center justify-center space-y-4 text-center">
					<div className="space-y-2">
						<h2 className="text-3xl font-bold tracking-tighter sm:text-4xl md:text-5xl">Join Our Newsletter</h2>
						<p className="max-w-[700px] text-muted-foreground md:text-xl/relaxed lg:text-base/relaxed xl:text-xl/relaxed">
							Stay updated with the latest events, products, and community news.
						</p>
					</div>
					<div className="mx-auto w-full max-w-md space-y-2">
						<form className="flex space-x-2">
							<Input
								placeholder="Enter your email"
								type="email"
								required
							/>
							<Button type="submit">Subscribe</Button>
						</form>
						<p className="text-xs text-muted-foreground">
							By subscribing, you agree to our terms and privacy policy.
						</p>
					</div>
				</div>
			</div>
		</section>
	);
}
