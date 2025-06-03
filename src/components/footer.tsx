import { Facebook, Instagram, Mail, Phone, Youtube } from 'lucide-react';
import { Image } from './ui/image';

const socialLinks = [
	{
		name: 'Facebook',
		url: '#',
		icon: '/facebook.svg',
	},
	{
		name: 'Instagram',
		url: '#',
		icon: '/instagram.svg',
	},
	{
		name: 'YouTube',
		url: '#',
		icon: '/youtube.svg',
	},
];

export function Footer() {
	return (
		<footer className="bg-muted py-12">
			<div className="px-4 md:px-6">
				<div className="grid grid-cols-1 gap-8 md:grid-cols-2 lg:grid-cols-4">
					<div>
						<h3 className="text-lg font-bold">Euro Haus</h3>
						<p className="mt-2 text-sm text-muted-foreground">Building a community around car enthusiasm since 2015.</p>
						<div className="mt-4 flex space-x-3">
							<a href="https://instagram.com" className="text-muted-foreground hover:text-primary">
								<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" stroke-linejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-brand-instagram"><path stroke="none" d="M0 0h24v24H0z" fill="none" /><path d="M4 8a4 4 0 0 1 4 -4h8a4 4 0 0 1 4 4v8a4 4 0 0 1 -4 4h-8a4 4 0 0 1 -4 -4z" /><path d="M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0" /><path d="M16.5 7.5v.01" /></svg>
								<span className="sr-only">Instagram</span>
							</a>
							<a href="https://facebook.com" className="text-muted-foreground hover:text-primary">
								<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" stroke-linejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-brand-facebook"><path stroke="none" d="M0 0h24v24H0z" fill="none" /><path d="M7 10v4h3v7h4v-7h3l1 -4h-4v-2a1 1 0 0 1 1 -1h3v-4h-3a5 5 0 0 0 -5 5v2h-3" /></svg>
								<span className="sr-only">Facebook</span>
							</a>
							<a href="https://youtube.com" className="text-muted-foreground hover:text-primary">
								<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" stroke-linejoin="round" className="icon icon-tabler icons-tabler-outline icon-tabler-brand-youtube"><path stroke="none" d="M0 0h24v24H0z" fill="none" /><path d="M2 8a4 4 0 0 1 4 -4h12a4 4 0 0 1 4 4v8a4 4 0 0 1 -4 4h-12a4 4 0 0 1 -4 -4v-8z" /><path d="M10 9l5 3l-5 3z" /></svg>
								<span className="sr-only">YouTube</span>
							</a>
						</div>
					</div>
					<div>
						<h3 className="text-lg font-bold">Shop</h3>
						<ul className="mt-2 space-y-2 text-sm">
							<li>
								<a href="/shop/apparel" className="text-muted-foreground hover:text-foreground">
									Apparel
								</a>
							</li>
							<li>
								<a href="/shop/accessories" className="text-muted-foreground hover:text-foreground">
									Accessories
								</a>
							</li>
							<li>
								<a href="/shop/tickets" className="text-muted-foreground hover:text-foreground">
									Event Tickets
								</a>
							</li>
							<li>
								<a href="/shop/gift-cards" className="text-muted-foreground hover:text-foreground">
									Gift Cards
								</a>
							</li>
						</ul>
					</div>
					<div>
						<h3 className="text-lg font-bold">Community</h3>
						<ul className="mt-2 space-y-2 text-sm">
							<li>
								<a href="/events" className="text-muted-foreground hover:text-foreground">
									Events
								</a>
							</li>
							<li>
								<a href="/community/forum" className="text-muted-foreground hover:text-foreground">
									Forum
								</a>
							</li>
							<li>
								<a href="/community/gallery" className="text-muted-foreground hover:text-foreground">
									Car Gallery
								</a>
							</li>
							<li>
								<a href="/videos" className="text-muted-foreground hover:text-foreground">
									Videos
								</a>
							</li>
						</ul>
					</div>
					<div>
						<h3 className="text-lg font-bold">Contact</h3>
						<ul className="mt-2 space-y-2 text-sm">
							<li className="flex items-center">
								<Mail className="mr-2 h-4 w-4 text-muted-foreground" />
								<a href="mailto:info@eurohaus.com" className="text-muted-foreground hover:text-foreground">
									info@eurohaus.com
								</a>
							</li>
						</ul>
					</div>
				</div>
				<div className="mt-12 border-t pt-6">
					<div className="flex flex-col items-center justify-between gap-4 md:flex-row">
						<p className="text-sm text-muted-foreground">
							© {new Date().getFullYear()} Euro Haus. All rights reserved.
						</p>
						<ul className="flex space-x-4 text-sm">
							<li>
								<a href="/terms" className="text-muted-foreground hover:text-foreground">
									Terms
								</a>
							</li>
							<li>
								<a href="/privacy" className="text-muted-foreground hover:text-foreground">
									Privacy
								</a>
							</li>
							<li>
								<a href="/cookies" className="text-muted-foreground hover:text-foreground">
									Cookies
								</a>
							</li>
						</ul>
					</div>
				</div>
			</div>
		</footer>
	);
}
