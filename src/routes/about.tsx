import { createFileRoute } from '@tanstack/react-router'
import { Calendar, Users, Trophy, Heart, Mail, Phone, MapPin } from 'lucide-react'
import { Card, CardContent } from '~/components/ui/card'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbSeparator } from '~/components/ui/breadcrumb'
import { SocialSection } from '~/components/social-section'
import { NewsletterSubscription } from '~/components/newsletter-subscription'
import { ContactForm } from '~/components/contact-form'

export const Route = createFileRoute('/about')({
	component: RouteComponent,
})

function RouteComponent() {
	return (
		<>
			{/* Breadcrumb */}
			<div className="container px-4 py-6 md:px-6">
				<Breadcrumb>
					<BreadcrumbList>
						<BreadcrumbItem>
							<BreadcrumbLink href="/" className="text-primary hover:text-primary/80">
								Home
							</BreadcrumbLink>
						</BreadcrumbItem>
						<BreadcrumbSeparator />
						<BreadcrumbItem>
							<BreadcrumbLink href="#" isCurrentPage>
								About
							</BreadcrumbLink>
						</BreadcrumbItem>
					</BreadcrumbList>
				</Breadcrumb>
			</div>

			<section className='relative bg-gradient-to-br from-primary/10 via-background to-secondary/10 py-16 md:py-24'>
				<div className='px-4 md:px-6'>
					<div className='mx-auto max-w-4xl text-center'>
						<h1 className='text-4xl font-bold tracking-tight text-secondary md:text-6xl lg:text-7xl'>
							About Euro Haus
						</h1>
						<p className='mt-6 text-lg text-muted-foreground md:text-xl'>
							Building a community around car enthusiasm since 2015. We're more than just a brand - we're a family of passionate automative enthusiasts dedicated to celebrating European car culture.
						</p>
					</div>
				</div>
			</section>

			{/* Our Story Section */}
			<section className="py-16 md:py-20">
				<div className="px-4 md:px-6">
					<div className="mx-auto max-w-4xl">
						<div className="text-center mb-12">
							<h2 className="text-3xl font-bold tracking-tight text-secondary md:text-4xl">Our Story</h2>
							<p className="mt-4 text-muted-foreground">
								From humble beginnings to a thriving community of car enthusiasts
							</p>
						</div>

						<div className="prose prose-lg max-w-none">
							<div className="grid gap-8 md:grid-cols-2 md:gap-12 items-center">
								<div>
									<h3 className="text-2xl font-bold text-primary mb-4">The Beginning</h3>
									<p className="text-muted-foreground leading-relaxed">
										Euro Haus was born in 2015 from a simple idea: to create a space where European car enthusiasts
										could come together, share their passion, and celebrate the engineering excellence that defines
										brands like BMW, Audi, Mercedes-Benz, Porsche, and Volkswagen.
									</p>
									<p className="text-muted-foreground leading-relaxed mt-4">
										What started as a small group of friends meeting in parking lots has evolved into one of the most
										respected communities in the automotive world, hosting events across the country and bringing
										together thousands of like-minded enthusiasts.
									</p>
								</div>
								<div className="relative">
									<img
										src="/placeholder.svg?height=400&width=600"
										alt="Euro Haus founding members"
										className="rounded-lg shadow-lg"
									/>
									<div className="absolute inset-0 bg-gradient-to-t from-primary/20 to-transparent rounded-lg"></div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</section>

			{/* Mission & Values Section */}
			<section className="bg-muted py-16 md:py-20">
				<div className="px-4 md:px-6">
					<div className="mx-auto max-w-6xl">
						<div className="text-center mb-12">
							<h2 className="text-3xl font-bold tracking-tight text-secondary md:text-4xl">Mission & Values</h2>
							<p className="mt-4 text-muted-foreground">The principles that guide everything we do</p>
						</div>

						<div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
							<Card className="border-primary/20">
								<CardContent className="p-6">
									<div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary mb-4">
										<Users className="h-6 w-6" />
									</div>
									<h3 className="text-xl font-bold text-secondary mb-2">Community First</h3>
									<p className="text-muted-foreground">
										We believe that the best experiences come from genuine connections. Our community is built on
										mutual respect, shared passion, and the joy of automotive culture.
									</p>
								</CardContent>
							</Card>

							<Card className="border-primary/20">
								<CardContent className="p-6">
									<div className="flex h-12 w-12 items-center justify-center rounded-lg bg-secondary/10 text-secondary mb-4">
										<Heart className="h-6 w-6" />
									</div>
									<h3 className="text-xl font-bold text-secondary mb-2">Passion Driven</h3>
									<p className="text-muted-foreground">
										Every event, product, and interaction is fueled by our genuine love for European automotive
										excellence and the culture that surrounds it.
									</p>
								</CardContent>
							</Card>

							<Card className="border-primary/20">
								<CardContent className="p-6">
									<div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary mb-4">
										<Trophy className="h-6 w-6" />
									</div>
									<h3 className="text-xl font-bold text-secondary mb-2">Excellence</h3>
									<p className="text-muted-foreground">
										We strive for excellence in everything we do, from the quality of our events to the products we
										offer and the experiences we create.
									</p>
								</CardContent>
							</Card>
						</div>
					</div>
				</div>
			</section>

			{/* Timeline Section */}
			<section className="py-16 md:py-20">
				<div className="px-4 md:px-6">
					<div className="mx-auto max-w-4xl">
						<div className="text-center mb-12">
							<h2 className="text-3xl font-bold tracking-tight text-secondary md:text-4xl">Our Journey</h2>
							<p className="mt-4 text-muted-foreground">Key milestones in the Euro Haus story</p>
						</div>

						<div className="relative">
							<div className="absolute left-4 top-0 bottom-0 w-0.5 bg-primary/30 md:left-1/2 md:-translate-x-0.5"></div>

							<div className="space-y-12">
								<div className="relative flex items-center md:justify-start">
									<div className="absolute left-0 flex h-8 w-8 items-center justify-center rounded-full bg-primary text-white md:left-1/2 md:-translate-x-1/2">
										<Calendar className="h-4 w-4" />
									</div>
									<div className="ml-12 md:ml-0 md:w-1/2 md:pr-8">
										<div className="rounded-lg bg-background p-6 shadow-sm border border-primary/20">
											<h3 className="text-xl font-bold text-secondary">2015 - The Foundation</h3>
											<p className="mt-2 text-muted-foreground">
												Euro Haus was founded with the first official meet in Orlando, bringing together 50 European
												car enthusiasts for a weekend of passion and community.
											</p>
										</div>
									</div>
								</div>

								<div className="relative flex items-center md:justify-end">
									<div className="absolute left-0 flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-white md:left-1/2 md:-translate-x-1/2">
										<Users className="h-4 w-4" />
									</div>
									<div className="ml-12 md:ml-0 md:w-1/2 md:pl-8">
										<div className="rounded-lg bg-background p-6 shadow-sm border border-secondary/20">
											<h3 className="text-xl font-bold text-secondary">2017 - Going National</h3>
											<p className="mt-2 text-muted-foreground">
												Expanded to multiple cities across the United States, hosting our first multi-state rally with
												over 200 participants from coast to coast.
											</p>
										</div>
									</div>
								</div>

								<div className="relative flex items-center md:justify-start">
									<div className="absolute left-0 flex h-8 w-8 items-center justify-center rounded-full bg-primary text-white md:left-1/2 md:-translate-x-1/2">
										<Trophy className="h-4 w-4" />
									</div>
									<div className="ml-12 md:ml-0 md:w-1/2 md:pr-8">
										<div className="rounded-lg bg-background p-6 shadow-sm border border-primary/20">
											<h3 className="text-xl font-bold text-secondary">2019 - Merchandise Launch</h3>
											<p className="mt-2 text-muted-foreground">
												Launched our first line of premium merchandise, allowing our community to represent Euro Haus
												with pride wherever they go.
											</p>
										</div>
									</div>
								</div>

								<div className="relative flex items-center md:justify-end">
									<div className="absolute left-0 flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-white md:left-1/2 md:-translate-x-1/2">
										<Heart className="h-4 w-4" />
									</div>
									<div className="ml-12 md:ml-0 md:w-1/2 md:pl-8">
										<div className="rounded-lg bg-background p-6 shadow-sm border border-secondary/20">
											<h3 className="text-xl font-bold text-secondary">2021 - Digital Evolution</h3>
											<p className="mt-2 text-muted-foreground">
												Adapted to the digital age with virtual events and online community building, keeping our
												family connected during challenging times.
											</p>
										</div>
									</div>
								</div>

								<div className="relative flex items-center md:justify-start">
									<div className="absolute left-0 flex h-8 w-8 items-center justify-center rounded-full bg-primary text-white md:left-1/2 md:-translate-x-1/2">
										<Calendar className="h-4 w-4" />
									</div>
									<div className="ml-12 md:ml-0 md:w-1/2 md:pr-8">
										<div className="rounded-lg bg-background p-6 shadow-sm border border-primary/20">
											<h3 className="text-xl font-bold text-secondary">2024 - Looking Forward</h3>
											<p className="mt-2 text-muted-foreground">
												Today, Euro Haus continues to grow with thousands of members nationwide, planning our biggest
												events yet and expanding our community reach.
											</p>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</section>

			{/* Contact Section */}
			<section className="py-16 md:py-20">
				<div className="px-4 md:px-6">
					<div className="mx-auto max-w-6xl">
						<div className="text-center mb-12">
							<h2 className="text-3xl font-bold tracking-tight text-secondary md:text-4xl">Get in Touch</h2>
							<p className="mt-4 text-muted-foreground">
								Have questions, ideas, or want to collaborate? We'd love to hear from you.
							</p>
						</div>

						<div className="grid gap-12 lg:grid-cols-2">
							{/* Contact Info */}
							<div>
								<h3 className="text-2xl font-bold text-primary mb-6">Contact Information</h3>
								<div className="space-y-6">
									<div className="flex items-start space-x-4">
										<div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
											<Mail className="h-5 w-5" />
										</div>
										<div>
											<h4 className="font-semibold">Email</h4>
											<p className="text-muted-foreground">info@eurohaus.com</p>
											<p className="text-sm text-muted-foreground">We'll get back to you within 24 hours</p>
										</div>
									</div>

									<div className="flex items-start space-x-4">
										<div className="flex h-10 w-10 items-center justify-center rounded-lg bg-secondary/10 text-secondary">
											<Phone className="h-5 w-5" />
										</div>
										<div>
											<h4 className="font-semibold">Phone</h4>
											<p className="text-muted-foreground">(555) 123-EURO</p>
											<p className="text-sm text-muted-foreground">Monday - Friday, 9AM - 6PM EST</p>
										</div>
									</div>

									<div className="flex items-start space-x-4">
										<div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
											<MapPin className="h-5 w-5" />
										</div>
										<div>
											<h4 className="font-semibold">Headquarters</h4>
											<p className="text-muted-foreground">Orlando, Florida</p>
											<p className="text-sm text-muted-foreground">Where it all began</p>
										</div>
									</div>
								</div>

								<div className="mt-8">
									<h4 className="font-semibold mb-4">Follow Us</h4>
									<div className="flex space-x-4">
										<a
											href="https://instagram.com"
											className="rounded-full bg-primary/10 p-3 text-primary hover:bg-primary hover:text-white transition-colors"
										>
											<svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
												<path d="M12.017 0C5.396 0 .029 5.367.029 11.987c0 6.62 5.367 11.987 11.988 11.987s11.987-5.367 11.987-11.987C24.004 5.367 18.637.001 12.017.001zM8.449 16.988c-1.297 0-2.448-.49-3.323-1.297C4.198 14.864 3.708 13.713 3.708 12.416s.49-2.448 1.418-3.323c.875-.875 2.026-1.297 3.323-1.297s2.448.422 3.323 1.297c.928.875 1.418 2.026 1.418 3.323s-.49 2.448-1.418 3.275c-.875.807-2.026 1.297-3.323 1.297zm7.83-9.608c-.49 0-.928-.422-.928-.928 0-.49.422-.928.928-.928.49 0 .928.422.928.928 0 .49-.422.928-.928.928zm-3.323 9.608c-2.448 0-4.474-1.959-4.474-4.407 0-2.448 1.959-4.474 4.474-4.474s4.407 1.959 4.407 4.474c0 2.448-1.959 4.407-4.407 4.407z" />
											</svg>
										</a>
										<a
											href="https://facebook.com"
											className="rounded-full bg-secondary/10 p-3 text-secondary hover:bg-secondary hover:text-white transition-colors"
										>
											<svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
												<path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
											</svg>
										</a>
										<a
											href="https://youtube.com"
											className="rounded-full bg-primary/10 p-3 text-primary hover:bg-primary hover:text-white transition-colors"
										>
											<svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
												<path d="M23.498 6.186a3.016 3.016 0 0 0-2.122-2.136C19.505 3.545 12 3.545 12 3.545s-7.505 0-9.377.505A3.017 3.017 0 0 0 .502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 0 0 2.122 2.136c1.871.505 9.376.505 9.376.505s7.505 0 9.377-.505a3.015 3.015 0 0 0 2.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z" />
											</svg>
										</a>
									</div>
								</div>
							</div>

							{/* Contact Form */}
							<ContactForm />
						</div>
					</div>
				</div>
			</section>

			<NewsletterSubscription />

			<SocialSection />
		</>
	)
}
