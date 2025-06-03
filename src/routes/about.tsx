import { createFileRoute } from '@tanstack/react-router'
import { Calendar, Users, Trophy, Heart, Mail, MapPin } from 'lucide-react'
import { Card, CardContent } from '~/components/ui/card'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbSeparator } from '~/components/ui/breadcrumb'
import { SocialSection } from '~/components/social-section'
import { NewsletterSubscription } from '~/components/newsletter-subscription'
import { ContactForm } from '~/components/contact-form'
import { Image } from '~/components/ui/image'

import LineupImage from '~/assets/PANA1748.jpg'

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

			<section className='relative bg-gradient-to-br from-primary/15 via-background to-secondary/15 py-16 md:py-24'>
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
									<Image
										src={LineupImage}
										alt=""
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
											className="rounded-full bg-primary/10 p-2 text-primary hover:bg-primary hover:text-white transition-colors"
										>
											<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" stroke-linejoin="round" className="lucide lucide-instagram-icon lucide-instagram"><rect width="20" height="20" x="2" y="2" rx="5" ry="5" /><path d="M16 11.37A4 4 0 1 1 12.63 8 4 4 0 0 1 16 11.37z" /><line x1="17.5" x2="17.51" y1="6.5" y2="6.5" /></svg>
										</a>
										<a
											href="https://facebook.com"
											className="rounded-full bg-secondary/10 p-2 text-secondary hover:bg-secondary hover:text-white transition-colors"
										>
											<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" stroke-linejoin="round" className="lucide lucide-facebook-icon lucide-facebook"><path d="M18 2h-3a5 5 0 0 0-5 5v3H7v4h3v8h4v-8h3l1-4h-4V7a1 1 0 0 1 1-1h3z" /></svg>
										</a>
										<a
											href="https://youtube.com"
											className="rounded-full bg-primary/10 p-2 text-primary hover:bg-primary hover:text-white transition-colors"
										>
											<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" stroke-linejoin="round" className="lucide lucide-youtube-icon lucide-youtube"><path d="M2.5 17a24.12 24.12 0 0 1 0-10 2 2 0 0 1 1.4-1.4 49.56 49.56 0 0 1 16.2 0A2 2 0 0 1 21.5 7a24.12 24.12 0 0 1 0 10 2 2 0 0 1-1.4 1.4 49.55 49.55 0 0 1-16.2 0A2 2 0 0 1 2.5 17" /><path d="m10 15 5-3-5-3z" /></svg>
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
